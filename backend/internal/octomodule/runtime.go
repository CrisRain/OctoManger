package octomodule

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defaultBootstrapTimeout = 10 * time.Minute
	requirementsFileName    = "requirements.txt"
	requirementsHashFile    = ".octo_requirements.sha256"
)

type RuntimeInfo struct {
	PythonPath       string
	ModuleDir        string
	VenvCreated      bool
	DepsInstalled    bool
	RequirementsHash string
}

var runtimeLocks sync.Map

func EnsureRuntime(ctx context.Context, scriptPath string, pythonBin string) (RuntimeInfo, error) {
	trimmedScript := strings.TrimSpace(scriptPath)
	if trimmedScript == "" {
		return RuntimeInfo{}, fmt.Errorf("script path is required")
	}
	return EnsureRuntimeByModuleDir(ctx, filepath.Dir(trimmedScript), pythonBin)
}

func EnsureRuntimeByModuleDir(ctx context.Context, moduleDir string, pythonBin string) (RuntimeInfo, error) {
	trimmedModuleDir := strings.TrimSpace(moduleDir)
	if trimmedModuleDir == "" {
		return RuntimeInfo{}, fmt.Errorf("module directory is required")
	}

	absModuleDir, err := filepath.Abs(trimmedModuleDir)
	if err != nil {
		return RuntimeInfo{}, fmt.Errorf("resolve module directory: %w", err)
	}

	info, err := os.Stat(absModuleDir)
	if err != nil {
		return RuntimeInfo{}, fmt.Errorf("stat module directory: %w", err)
	}
	if !info.IsDir() {
		return RuntimeInfo{}, fmt.Errorf("module directory is not a directory: %s", absModuleDir)
	}

	basePython := strings.TrimSpace(pythonBin)
	if basePython == "" {
		basePython = "python"
	}

	runCtx := ctx
	cancel := func() {}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		runCtx, cancel = context.WithTimeout(ctx, defaultBootstrapTimeout)
	}
	defer cancel()

	lock := moduleRuntimeLock(absModuleDir)
	lock.Lock()
	defer lock.Unlock()

	result := RuntimeInfo{
		PythonPath: VenvPythonPath(absModuleDir),
		ModuleDir:  absModuleDir,
	}

	if !VenvExists(absModuleDir) {
		if err := createVenv(runCtx, absModuleDir, basePython); err != nil {
			return RuntimeInfo{}, err
		}
		result.VenvCreated = true
	}

	reqPath := filepath.Join(absModuleDir, requirementsFileName)
	if !FileExists(reqPath) {
		return result, nil
	}

	reqHash, err := fileSHA256(reqPath)
	if err != nil {
		return RuntimeInfo{}, fmt.Errorf("calculate requirements hash: %w", err)
	}
	result.RequirementsHash = reqHash

	hashPath := filepath.Join(VenvDir(absModuleDir), requirementsHashFile)
	currentHash, _ := readTrimmedFile(hashPath)
	if currentHash == reqHash {
		return result, nil
	}

	if err := ensurePip(runCtx, absModuleDir); err != nil {
		return RuntimeInfo{}, err
	}

	installOutput, err := runCommand(runCtx, absModuleDir, result.PythonPath, []string{"-m", "pip", "install", "-r", reqPath})
	if err != nil {
		return RuntimeInfo{}, fmt.Errorf("install dependencies failed: %w (output=%s)", err, summarizeOutput(installOutput))
	}

	if err := os.WriteFile(hashPath, []byte(reqHash+"\n"), 0o644); err != nil {
		return RuntimeInfo{}, fmt.Errorf("write requirements hash marker: %w", err)
	}
	result.DepsInstalled = true
	return result, nil
}

func moduleRuntimeLock(moduleDir string) *sync.Mutex {
	actual, _ := runtimeLocks.LoadOrStore(moduleDir, &sync.Mutex{})
	lock, _ := actual.(*sync.Mutex)
	if lock == nil {
		return &sync.Mutex{}
	}
	return lock
}

func createVenv(ctx context.Context, moduleDir string, basePython string) error {
	output, err := runCommand(ctx, moduleDir, basePython, []string{"-m", "venv", ".venv"})
	if err == nil {
		return nil
	}

	fallbackOutput, fallbackErr := runCommand(ctx, moduleDir, basePython, []string{"-m", "venv", "--without-pip", ".venv"})
	if fallbackErr != nil {
		return fmt.Errorf("create venv failed: %w (output=%s)", err, summarizeOutput(output))
	}

	venvPython := VenvPythonPath(moduleDir)
	ensurePipOutput, ensurePipErr := runCommand(ctx, moduleDir, venvPython, []string{"-m", "ensurepip", "--upgrade"})
	if ensurePipErr != nil {
		return fmt.Errorf(
			"create venv fallback failed: %w (venv_output=%s, ensurepip_output=%s)",
			ensurePipErr,
			summarizeOutput(fallbackOutput),
			summarizeOutput(ensurePipOutput),
		)
	}
	return nil
}

func ensurePip(ctx context.Context, moduleDir string) error {
	if FileExists(VenvPipPath(moduleDir)) {
		return nil
	}
	venvPython := VenvPythonPath(moduleDir)
	output, err := runCommand(ctx, moduleDir, venvPython, []string{"-m", "ensurepip", "--upgrade"})
	if err != nil {
		return fmt.Errorf("ensure pip failed: %w (output=%s)", err, summarizeOutput(output))
	}
	return nil
}

func runCommand(ctx context.Context, dir string, binary string, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	env := os.Environ()
	env = append(env, "PYTHONIOENCODING=UTF-8")
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return string(output), fmt.Errorf("command timed out")
		}
		return string(output), err
	}
	return string(output), nil
}

func fileSHA256(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

func readTrimmedFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func summarizeOutput(output string) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return "<empty>"
	}
	const maxLen = 800
	if len(trimmed) <= maxLen {
		return trimmed
	}
	return trimmed[:maxLen] + "...(truncated)"
}
