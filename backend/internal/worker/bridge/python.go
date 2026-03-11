package bridge

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"octomanger/backend/internal/octomodule"
)

type Input struct {
	Action  string         `json:"action"`
	Account InputAccount   `json:"account"`
	Params  map[string]any `json:"params"`
	Context InputContext   `json:"context"`
}

type InputAccount struct {
	Identifier string         `json:"identifier"`
	Spec       map[string]any `json:"spec"`
}

type InputContext struct {
	TenantID  string `json:"tenant_id"`
	RequestID string `json:"request_id"`
	Protocol  string `json:"protocol,omitempty"`
	APIURL    string `json:"api_url,omitempty"`
	APIToken  string `json:"api_token,omitempty"`
}

type Output struct {
	Status       string         `json:"status"`
	Result       map[string]any `json:"result,omitempty"`
	Session      *OutputSession `json:"session,omitempty"`
	Logs         []string       `json:"logs,omitempty"`
	ErrorCode    string         `json:"error_code,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
}

type OutputSession struct {
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload"`
	ExpiresAt string         `json:"expires_at,omitempty"`
}

type PythonBridge struct {
	Binary       string
	Script       string
	Timeout      time.Duration
	OnLog        func(source, level, message string)
	ServiceURL   string
	ServiceToken string
	ForceRemote  bool
}

// ExecutionError preserves stderr/stdout context and parsed runtime logs when
// module execution fails.
type ExecutionError struct {
	Err    error
	Logs   []string
	Stdout string
	Stderr string
}

func (e *ExecutionError) Error() string {
	if e == nil || e.Err == nil {
		return "python execution failed"
	}
	return e.Err.Error()
}

func (e *ExecutionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type outputLine struct {
	Status       string         `json:"status"`
	Level        string         `json:"level,omitempty"`
	Message      string         `json:"message,omitempty"`
	Result       map[string]any `json:"result,omitempty"`
	Session      *OutputSession `json:"session,omitempty"`
	ErrorCode    string         `json:"error_code,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
}

func resolveVenvPython(dir string) string {
	var candidate string
	if runtime.GOOS == "windows" {
		candidate = filepath.Join(dir, ".venv", "Scripts", "python.exe")
	} else {
		candidate = filepath.Join(dir, ".venv", "bin", "python")
	}
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

func resolveOctoSDKRoot(scriptPath string) string {
	dir := filepath.Dir(scriptPath)
	for {
		candidate := filepath.Join(dir, "octo.py")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func mergePythonPath(existing string, extra string) string {
	trimmedExtra := strings.TrimSpace(extra)
	if trimmedExtra == "" {
		return existing
	}
	trimmedExisting := strings.TrimSpace(existing)
	if trimmedExisting == "" {
		return trimmedExtra
	}
	return trimmedExisting + string(os.PathListSeparator) + trimmedExtra
}

func (p PythonBridge) Execute(ctx context.Context, input Input) (Output, error) {
	return p.ExecuteWithScript(ctx, p.Script, input)
}

func (p PythonBridge) ExecuteWithScript(ctx context.Context, scriptPath string, input Input) (Output, error) {
	if scriptPath == "" {
		return Output{}, fmt.Errorf("python script path is required")
	}
	if strings.TrimSpace(p.ServiceURL) != "" {
		return p.executeWithRemoteService(ctx, scriptPath, input)
	}
	if p.ForceRemote {
		return Output{}, fmt.Errorf("octomodule service url is required when remote mode is enforced")
	}

	bootstrapCtx := ctx
	bootstrapCancel := func() {}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		bootstrapCtx, bootstrapCancel = context.WithTimeout(ctx, 10*time.Minute)
	}
	defer bootstrapCancel()

	runtimeInfo, err := octomodule.EnsureRuntime(bootstrapCtx, scriptPath, p.Binary)
	if err != nil {
		return Output{}, fmt.Errorf("failed to prepare module runtime: %w", err)
	}

	execCtx := ctx
	cancel := func() {}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && p.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, p.Timeout)
	}
	defer cancel()

	input.Context.Protocol = "ndjson.v1"
	runBridge := p
	if strings.TrimSpace(runtimeInfo.PythonPath) != "" {
		runBridge.Binary = runtimeInfo.PythonPath
	}
	return sharedPythonServePool.Execute(execCtx, runBridge, scriptPath, input)
}

func (p PythonBridge) executeWithRemoteService(ctx context.Context, scriptPath string, input Input) (Output, error) {
	baseURL := strings.TrimSpace(p.ServiceURL)
	if baseURL == "" {
		return Output{}, fmt.Errorf("remote octomodule service url is empty")
	}

	requestPayload := ServiceExecuteRequest{
		ScriptPath: scriptPath,
		Input:      input,
	}
	rawRequest, err := json.Marshal(requestPayload)
	if err != nil {
		return Output{}, fmt.Errorf("marshal remote request: %w", err)
	}

	timeout := p.Timeout
	execCtx := ctx
	cancel := func() {}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	endpoint := strings.TrimRight(baseURL, "/") + "/v1/execute"
	req, err := http.NewRequestWithContext(execCtx, http.MethodPost, endpoint, bytes.NewReader(rawRequest))
	if err != nil {
		return Output{}, fmt.Errorf("build remote request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if token := strings.TrimSpace(p.ServiceToken); token != "" {
		req.Header.Set("X-Octo-Service-Token", token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Output{}, fmt.Errorf("remote octomodule service request failed: %w", err)
	}
	defer resp.Body.Close()

	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return Output{}, fmt.Errorf("read remote response: %w", err)
	}

	var payload ServiceExecuteResponse
	if len(strings.TrimSpace(string(rawResponse))) > 0 {
		if err := json.Unmarshal(rawResponse, &payload); err != nil {
			return Output{}, fmt.Errorf("decode remote response failed: %w", err)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(payload.Error)
		if message == "" {
			message = strings.TrimSpace(string(rawResponse))
		}
		if message == "" {
			message = fmt.Sprintf("remote service returned status %d", resp.StatusCode)
		}
		return Output{}, &ExecutionError{
			Err:    fmt.Errorf("%s", message),
			Logs:   append([]string(nil), payload.Logs...),
			Stdout: payload.Stdout,
			Stderr: payload.Stderr,
		}
	}

	if payload.Error != "" {
		return Output{}, &ExecutionError{
			Err:    fmt.Errorf("%s", strings.TrimSpace(payload.Error)),
			Logs:   append([]string(nil), payload.Logs...),
			Stdout: payload.Stdout,
			Stderr: payload.Stderr,
		}
	}

	if payload.Output == nil {
		return Output{}, fmt.Errorf("remote octomodule service returned empty output")
	}

	output := *payload.Output
	replayLogs(output.Logs, p.OnLog)
	return output, nil
}

func scanLines(reader io.Reader, onLine func(string), onError func(error)) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		onLine(scanner.Text())
	}
	if err := scanner.Err(); err != nil && onError != nil {
		onError(err)
	}
}

func formatLogLine(source, level, message string) string {
	return fmt.Sprintf("[%s][%s] %s", strings.TrimSpace(source), normalizeLogLevel(level), strings.TrimSpace(message))
}

func normalizeLogLevel(level string) string {
	normalized := strings.ToLower(strings.TrimSpace(level))
	switch normalized {
	case "debug", "info", "warn", "warning", "error":
		if normalized == "warning" {
			return "warn"
		}
		return normalized
	default:
		return "info"
	}
}

func replayLogs(lines []string, sink func(source, level, message string)) {
	if sink == nil || len(lines) == 0 {
		return
	}
	for _, line := range lines {
		source, level, message := parseFormattedLogLine(line)
		sink(source, level, message)
	}
}

func parseFormattedLogLine(line string) (source, level, message string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "module", "info", ""
	}
	if !strings.HasPrefix(trimmed, "[") {
		return "module", "info", trimmed
	}

	firstClose := strings.Index(trimmed, "]")
	if firstClose <= 1 {
		return "module", "info", trimmed
	}
	secondStart := firstClose + 1
	if secondStart >= len(trimmed) || trimmed[secondStart] != '[' {
		return "module", "info", trimmed
	}
	secondClose := strings.Index(trimmed[secondStart:], "]")
	if secondClose <= 1 {
		return "module", "info", trimmed
	}

	source = strings.TrimSpace(trimmed[1:firstClose])
	levelStart := secondStart + 1
	levelEnd := secondStart + secondClose
	level = normalizeLogLevel(trimmed[levelStart:levelEnd])
	message = strings.TrimSpace(trimmed[levelEnd+1:])
	if source == "" {
		source = "module"
	}
	if message == "" {
		message = trimmed
	}
	return source, level, message
}
