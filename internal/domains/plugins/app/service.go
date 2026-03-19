package pluginapp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	plugindomain "octomanger/internal/domains/plugins/domain"
)

type Repository interface {
	List(ctx context.Context) ([]plugindomain.Plugin, error)
	Get(ctx context.Context, key string) (*plugindomain.Plugin, error)
}

type Service struct {
	repo      Repository
	pythonBin string
	sdkDir    string
}

func New(repo Repository, pythonBin string, sdkDir string) Service {
	return Service{
		repo:      repo,
		pythonBin: strings.TrimSpace(pythonBin),
		sdkDir:    strings.TrimSpace(sdkDir),
	}
}

func (s Service) List(ctx context.Context) ([]plugindomain.Plugin, error) {
	return s.repo.List(ctx)
}

// AccountTypeSpec is the subset of account_type.{key}.json that maps to an AccountType record.
type AccountTypeSpec struct {
	Key          string         `json:"key"`
	Name         string         `json:"name"`
	Category     string         `json:"category"`
	Schema       map[string]any `json:"schema"`
	Capabilities map[string]any `json:"capabilities"`
}

// SyncAccountTypeFunc is called once per plugin with the parsed account type spec.
// The caller (Bootstrap) wires this to accountTypeService.Upsert.
type SyncAccountTypeFunc func(ctx context.Context, spec AccountTypeSpec) error

// SyncAccountTypes reads every plugin's account_type.{key}.json and calls fn for each one.
// Missing files are silently skipped. Parse errors are returned immediately.
func (s Service) SyncAccountTypes(ctx context.Context, fn SyncAccountTypeFunc) error {
	plugins, err := s.repo.List(ctx)
	if err != nil {
		return fmt.Errorf("list plugins for sync: %w", err)
	}

	for _, plugin := range plugins {
		filename := filepath.Join(plugin.Directory, fmt.Sprintf("account_type.%s.json", plugin.Manifest.Key))
		data, err := os.ReadFile(filename)
		if err != nil {
			if os.IsNotExist(err) {
				continue // no account type file for this plugin — skip
			}
			return fmt.Errorf("read account type file %s: %w", filename, err)
		}

		var spec AccountTypeSpec
		if err := json.Unmarshal(data, &spec); err != nil {
			return fmt.Errorf("parse account type file %s: %w", filename, err)
		}
		if spec.Key == "" {
			spec.Key = plugin.Manifest.Key
		}
		if spec.Name == "" {
			spec.Name = plugin.Manifest.Name
		}
		if spec.Category == "" {
			spec.Category = "generic"
		}

		if err := fn(ctx, spec); err != nil {
			return fmt.Errorf("sync account type %s: %w", spec.Key, err)
		}
	}
	return nil
}

func (s Service) Get(ctx context.Context, key string) (*plugindomain.Plugin, error) {
	return s.repo.Get(ctx, key)
}

func (s Service) Execute(
	ctx context.Context,
	pluginKey string,
	request plugindomain.ExecutionRequest,
	onEvent func(plugindomain.ExecutionEvent),
) error {
	plugin, err := s.repo.Get(ctx, pluginKey)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal plugin request: %w", err)
	}

	entrypoint := plugin.Manifest.Entrypoint
	command := exec.CommandContext(ctx, s.pythonBinary(), entrypoint)
	command.Dir = plugin.Directory
	command.Env = append(command.Environ(), fmt.Sprintf("PYTHONPATH=%s", s.pythonPath()))

	stdout, err := command.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open plugin stdout: %w", err)
	}
	command.Stderr = command.Stdout
	command.Stdin = strings.NewReader(string(payload))

	if err := command.Start(); err != nil {
		return fmt.Errorf("start plugin command: %w", err)
	}

	var receivedError bool
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		var event plugindomain.ExecutionEvent
		if err := json.Unmarshal(line, &event); err != nil {
			// Non-JSON output (e.g. Python traceback on stderr) — forward as a log event
			// so callers can surface the raw text rather than getting an opaque decode error.
			if onEvent != nil {
				onEvent(plugindomain.ExecutionEvent{Type: "log", Message: string(line)})
			}
			continue
		}
		if event.Type == "error" {
			receivedError = true
		}
		if onEvent != nil {
			onEvent(event)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read plugin output: %w", err)
	}

	if err := command.Wait(); err != nil {
		if receivedError {
			// The plugin already communicated the failure via an error event; suppress the
			// redundant non-zero exit-code error so callers don't see it twice.
			return nil
		}
		// Process failed without emitting an error event — surface it as one.
		if onEvent != nil {
			onEvent(plugindomain.ExecutionEvent{Type: "error", Message: err.Error(), Error: err.Error()})
		}
		return fmt.Errorf("wait plugin command: %w", err)
	}

	return nil
}

func (s Service) pythonBinary() string {
	if s.pythonBin != "" {
		return s.pythonBin
	}
	return "python3"
}

func (s Service) pythonPath() string {
	if s.sdkDir == "" {
		return ""
	}
	abs, err := filepath.Abs(s.sdkDir)
	if err != nil {
		return s.sdkDir
	}
	// Preserve any existing PYTHONPATH so system packages remain accessible.
	if existing := os.Getenv("PYTHONPATH"); existing != "" {
		return abs + string(os.PathListSeparator) + existing
	}
	return abs
}
