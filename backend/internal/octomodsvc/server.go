package octomodsvc

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"octomanger/backend/config"
	"octomanger/backend/internal/worker/bridge"
)

func Run(ctx context.Context, cfg config.Config, log *zap.Logger) error {
	if log == nil {
		log = zap.NewNop()
	}

	runner := bridge.PythonBridge{
		Binary:  cfg.Python.Bin,
		Timeout: cfg.Python.Timeout(),
	}

	serviceToken := strings.TrimSpace(cfg.OctoModuleService.Token)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
		})
	})
	mux.HandleFunc("/v1/execute", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, bridge.ServiceExecuteResponse{
				Error: "method not allowed",
			})
			return
		}
		if !authorizeRequest(r, serviceToken) {
			writeJSON(w, http.StatusUnauthorized, bridge.ServiceExecuteResponse{
				Error: "unauthorized",
			})
			return
		}

		var req bridge.ServiceExecuteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, bridge.ServiceExecuteResponse{
				Error: fmt.Sprintf("invalid request body: %v", err),
			})
			return
		}
		if strings.TrimSpace(req.ScriptPath) == "" {
			writeJSON(w, http.StatusBadRequest, bridge.ServiceExecuteResponse{
				Error: "script_path is required",
			})
			return
		}

		output, runErr := runner.ExecuteWithScript(r.Context(), req.ScriptPath, req.Input)
		if runErr != nil {
			var execErr *bridge.ExecutionError
			if errors.As(runErr, &execErr) {
				writeJSON(w, http.StatusBadGateway, bridge.ServiceExecuteResponse{
					Error:  runErr.Error(),
					Logs:   append([]string(nil), execErr.Logs...),
					Stdout: execErr.Stdout,
					Stderr: execErr.Stderr,
				})
				return
			}
			writeJSON(w, http.StatusInternalServerError, bridge.ServiceExecuteResponse{
				Error: runErr.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, bridge.ServiceExecuteResponse{
			Output: &output,
		})
	})

	addr := normalizeListenAddr(cfg.OctoModuleService.Listen)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  0,
		WriteTimeout: 0,
		IdleTimeout:  0,
	}

	errC := make(chan error, 1)
	go func() {
		log.Info("octomodule service started", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errC <- err
			return
		}
		errC <- nil
	}()

	select {
	case err := <-errC:
		if err != nil {
			return fmt.Errorf("octomodule service failed: %w", err)
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("octomodule service shutdown failed: %w", err)
		}
		<-errC
		log.Info("octomodule service stopped")
		return nil
	}
}

func authorizeRequest(r *http.Request, token string) bool {
	expected := strings.TrimSpace(token)
	if expected == "" {
		return true
	}
	actual := strings.TrimSpace(r.Header.Get("X-Octo-Service-Token"))
	if actual == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func normalizeListenAddr(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ":8091"
	}
	if strings.Contains(trimmed, ":") {
		return trimmed
	}
	return ":" + trimmed
}
