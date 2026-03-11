package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	sharedPythonServePool = newPythonServePool()
	serveRequestSeq       uint64
)

const serveIPCProtocol = "octo.ipc.v1"

type pythonServePool struct {
	mu       sync.Mutex
	runtimes map[string]*pythonServeRuntime
}

type pythonServeRuntime struct {
	binary     string
	scriptPath string

	requestMu sync.Mutex
	stateMu   sync.Mutex
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	current   *pythonServeRequest

	stderrLines []string
}

type pythonServeRequest struct {
	id        string
	onLog     func(source, level, message string)
	responseC chan Output
	errorC    chan error

	done sync.Once
	mu   sync.Mutex
	logs []string
}

func newPythonServePool() *pythonServePool {
	return &pythonServePool{
		runtimes: make(map[string]*pythonServeRuntime),
	}
}

func (p *pythonServePool) Execute(ctx context.Context, bridge PythonBridge, scriptPath string, input Input) (Output, error) {
	key, binary := serveRuntimeKey(bridge, scriptPath)

	p.mu.Lock()
	runtime, exists := p.runtimes[key]
	if !exists {
		runtime = &pythonServeRuntime{
			binary:     binary,
			scriptPath: filepath.Clean(scriptPath),
		}
		p.runtimes[key] = runtime
	}
	p.mu.Unlock()

	return runtime.Execute(ctx, input, bridge.OnLog)
}

func serveRuntimeKey(bridge PythonBridge, scriptPath string) (string, string) {
	cleanedScript := filepath.Clean(scriptPath)
	binary := strings.TrimSpace(bridge.Binary)
	if venvPython := resolveVenvPython(filepath.Dir(cleanedScript)); venvPython != "" {
		binary = venvPython
	}
	scriptToken := fileModToken(cleanedScript)
	sdkToken := ""
	if sdkRoot := resolveOctoSDKRoot(cleanedScript); sdkRoot != "" {
		sdkToken = fileModToken(filepath.Join(sdkRoot, "octo.py"))
	}
	return binary + "\x00" + cleanedScript + "\x00" + scriptToken + "\x00" + sdkToken, binary
}

func fileModToken(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "0"
	}
	return fmt.Sprintf("%d", info.ModTime().UnixNano())
}

func (r *pythonServeRuntime) Execute(ctx context.Context, input Input, onLog func(source, level, message string)) (Output, error) {
	r.requestMu.Lock()
	defer r.requestMu.Unlock()

	if err := r.ensureStarted(); err != nil {
		return Output{}, err
	}

	active := &pythonServeRequest{
		id:        fmt.Sprintf("serve-%d", atomic.AddUint64(&serveRequestSeq, 1)),
		onLog:     onLog,
		responseC: make(chan Output, 1),
		errorC:    make(chan error, 1),
	}

	if err := r.bindCurrent(active); err != nil {
		return Output{}, err
	}

	if err := r.sendRequest(active.id, input); err != nil {
		r.abortCurrent(active, &ExecutionError{
			Err:    fmt.Errorf("failed to send request to python runtime: %w", err),
			Logs:   active.snapshotLogs(),
			Stderr: r.snapshotStderr(),
		})
		return Output{}, <-active.errorC
	}

	select {
	case output := <-active.responseC:
		r.clearCurrent(active)
		return output, nil
	case err := <-active.errorC:
		r.clearCurrent(active)
		return Output{}, err
	case <-ctx.Done():
		r.abortCurrent(active, &ExecutionError{
			Err:    fmt.Errorf("python runtime request canceled: %w", ctx.Err()),
			Logs:   active.snapshotLogs(),
			Stderr: r.snapshotStderr(),
		})
		return Output{}, <-active.errorC
	}
}

func (r *pythonServeRuntime) ensureStarted() error {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()

	if r.cmd != nil && r.stdin != nil {
		return nil
	}

	cmd := exec.Command(strings.TrimSpace(r.binary), r.scriptPath)
	cmd.Dir = filepath.Dir(r.scriptPath)
	env := os.Environ()
	env = append(env, "PYTHONUNBUFFERED=1", "PYTHONIOENCODING=UTF-8")
	if sdkRoot := resolveOctoSDKRoot(r.scriptPath); sdkRoot != "" {
		env = append(env, "PYTHONPATH="+mergePythonPath(os.Getenv("PYTHONPATH"), sdkRoot))
	}
	cmd.Env = env

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to capture runtime stdout: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to capture runtime stderr: %w", err)
	}
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to capture runtime stdin: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("python runtime failed to start: %w", err)
	}

	r.cmd = cmd
	r.stdin = stdinPipe
	r.stderrLines = nil

	go r.readStdout(cmd, stdoutPipe)
	go r.readStderr(stderrPipe)
	go r.wait(cmd)
	return nil
}

func (r *pythonServeRuntime) bindCurrent(active *pythonServeRequest) error {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()

	if r.cmd == nil || r.stdin == nil {
		return fmt.Errorf("python runtime is not running")
	}
	if r.current != nil {
		return fmt.Errorf("python runtime is busy")
	}
	r.current = active
	return nil
}

func (r *pythonServeRuntime) clearCurrent(active *pythonServeRequest) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()

	if r.current == active {
		r.current = nil
	}
}

func (r *pythonServeRuntime) abortCurrent(active *pythonServeRequest, err error) {
	r.stateMu.Lock()
	cmd := r.cmd
	stdin := r.stdin
	if r.current == active {
		r.current = nil
	}
	r.cmd = nil
	r.stdin = nil
	r.stateMu.Unlock()

	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}

	active.fail(err)
}

func (r *pythonServeRuntime) sendRequest(requestID string, input Input) error {
	r.stateMu.Lock()
	stdin := r.stdin
	r.stateMu.Unlock()

	if stdin == nil {
		return fmt.Errorf("python runtime stdin is unavailable")
	}

	envelope := map[string]any{
		"protocol": serveIPCProtocol,
		"type":     "request",
		"id":       requestID,
		"payload":  input,
	}

	return json.NewEncoder(stdin).Encode(envelope)
}

func (r *pythonServeRuntime) readStdout(cmd *exec.Cmd, reader io.Reader) {
	scanLines(reader, func(line string) {
		r.handleStdoutLine(strings.TrimSpace(line))
	}, func(err error) {
		r.notifyCurrentError(&ExecutionError{
			Err:    fmt.Errorf("python runtime stdout scanner error: %w", err),
			Stderr: r.snapshotStderr(),
		})
	})
}

func (r *pythonServeRuntime) handleStdoutLine(line string) {
	if line == "" {
		return
	}

	current := r.currentRequest()
	if current == nil {
		return
	}

	var envelope struct {
		Protocol string          `json:"protocol,omitempty"`
		Type     string          `json:"type,omitempty"`
		ID       string          `json:"id,omitempty"`
		Payload  json.RawMessage `json:"payload,omitempty"`
	}
	if err := json.Unmarshal([]byte(line), &envelope); err == nil &&
		strings.EqualFold(strings.TrimSpace(envelope.Protocol), serveIPCProtocol) &&
		strings.EqualFold(strings.TrimSpace(envelope.Type), "response") &&
		len(envelope.Payload) > 0 {
		if envelope.ID != "" && envelope.ID != current.id {
			return
		}
		var evt outputLine
		if err := json.Unmarshal(envelope.Payload, &evt); err != nil {
			current.appendLog("stdout", "info", line)
			return
		}
		if output, ok := buildFinalOutput(evt); ok {
			current.complete(output)
			return
		}
		r.handleOutputEvent(current, evt, line)
		return
	}

	var evt outputLine
	if err := json.Unmarshal([]byte(line), &evt); err != nil {
		current.appendLog("stdout", "info", line)
		return
	}
	if output, ok := buildFinalOutput(evt); ok {
		current.complete(output)
		return
	}
	r.handleOutputEvent(current, evt, line)
}

func (r *pythonServeRuntime) handleOutputEvent(current *pythonServeRequest, evt outputLine, raw string) {
	status := strings.ToLower(strings.TrimSpace(evt.Status))
	switch status {
	case "log":
		msg := strings.TrimSpace(evt.Message)
		if msg == "" {
			msg = raw
		}
		current.appendLog("module", evt.Level, msg)
	case "":
		current.appendLog("stdout", "info", raw)
	default:
		msg := strings.TrimSpace(evt.Message)
		if msg == "" {
			msg = raw
		}
		current.appendLog("module", evt.Level, msg)
	}
}

func (r *pythonServeRuntime) readStderr(reader io.Reader) {
	scanLines(reader, func(line string) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			return
		}

		r.stateMu.Lock()
		r.stderrLines = append(r.stderrLines, trimmed)
		current := r.current
		r.stateMu.Unlock()

		if current != nil {
			current.appendLog("stderr", "warn", trimmed)
		}
	}, func(err error) {
		r.notifyCurrentError(&ExecutionError{
			Err:    fmt.Errorf("python runtime stderr scanner error: %w", err),
			Stderr: r.snapshotStderr(),
		})
	})
}

func (r *pythonServeRuntime) wait(cmd *exec.Cmd) {
	err := cmd.Wait()

	r.stateMu.Lock()
	if r.cmd != cmd {
		r.stateMu.Unlock()
		return
	}
	current := r.current
	r.current = nil
	r.cmd = nil
	r.stdin = nil
	stderrRaw := strings.Join(append([]string(nil), r.stderrLines...), "\n")
	r.stderrLines = nil
	r.stateMu.Unlock()

	if current == nil {
		return
	}
	if err != nil {
		current.fail(&ExecutionError{
			Err:    fmt.Errorf("python runtime exited: %w", err),
			Logs:   current.snapshotLogs(),
			Stderr: stderrRaw,
		})
		return
	}

	current.fail(&ExecutionError{
		Err:    fmt.Errorf("python runtime closed before returning a response"),
		Logs:   current.snapshotLogs(),
		Stderr: stderrRaw,
	})
}

func (r *pythonServeRuntime) currentRequest() *pythonServeRequest {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	return r.current
}

func (r *pythonServeRuntime) notifyCurrentError(err error) {
	r.stateMu.Lock()
	current := r.current
	r.stateMu.Unlock()
	if current != nil {
		current.fail(err)
	}
}

func (r *pythonServeRuntime) snapshotStderr() string {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	return strings.Join(append([]string(nil), r.stderrLines...), "\n")
}

func (r *pythonServeRequest) appendLog(source, level, message string) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return
	}

	normalizedLevel := normalizeLogLevel(level)
	formatted := formatLogLine(source, normalizedLevel, trimmed)

	r.mu.Lock()
	r.logs = append(r.logs, formatted)
	r.mu.Unlock()

	if r.onLog != nil {
		r.onLog(source, normalizedLevel, trimmed)
	}
}

func (r *pythonServeRequest) snapshotLogs() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.logs...)
}

func (r *pythonServeRequest) complete(output Output) {
	output.Logs = r.snapshotLogs()
	r.done.Do(func() {
		r.responseC <- output
	})
}

func (r *pythonServeRequest) fail(err error) {
	r.done.Do(func() {
		r.errorC <- err
	})
}

func buildFinalOutput(evt outputLine) (Output, bool) {
	switch strings.ToLower(strings.TrimSpace(evt.Status)) {
	case "success", "error":
		return Output{
			Status:       strings.TrimSpace(evt.Status),
			Result:       evt.Result,
			Session:      evt.Session,
			ErrorCode:    strings.TrimSpace(evt.ErrorCode),
			ErrorMessage: strings.TrimSpace(evt.ErrorMessage),
		}, true
	default:
		return Output{}, false
	}
}
