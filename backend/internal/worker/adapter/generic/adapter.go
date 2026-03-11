package generic

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"octomanger/backend/internal/worker/adapter"
	"octomanger/backend/internal/worker/bridge"
)

type Adapter struct {
	typeKey string
	bridge  bridge.PythonBridge
}

func New(typeKey string, bridge bridge.PythonBridge) *Adapter {
	return &Adapter{
		typeKey: typeKey,
		bridge:  bridge,
	}
}

func (a *Adapter) TypeKey() string {
	return a.typeKey
}

func (a *Adapter) ValidateSpec(spec map[string]any) error {
	if spec == nil {
		return errors.New("spec is required")
	}
	return nil
}

func (a *Adapter) ExecuteAction(ctx context.Context, request adapter.ActionRequest) (adapter.Result, error) {
	scriptPath := strings.TrimSpace(request.ModuleScript)
	var (
		output bridge.Output
		err    error
	)

	runBridge := a.bridge
	if request.LogSink != nil {
		runBridge.OnLog = request.LogSink
	}

	input := bridge.Input{
		Action: request.Action,
		Account: bridge.InputAccount{
			Identifier: request.Account.Identifier,
			Spec:       request.Account.Spec,
		},
		Params: request.Params,
		Context: bridge.InputContext{
			TenantID:  request.TenantID,
			RequestID: request.RequestID,
			Protocol:  "ndjson.v1",
			APIURL:    request.APIURL,
			APIToken:  request.APIToken,
		},
	}

	if scriptPath != "" {
		output, err = runBridge.ExecuteWithScript(ctx, scriptPath, input)
	} else {
		output, err = runBridge.Execute(ctx, input)
	}
	if err != nil {
		var runErr *bridge.ExecutionError
		if errors.As(err, &runErr) {
			return adapter.Result{Logs: append([]string(nil), runErr.Logs...)}, err
		}
		return adapter.Result{}, err
	}

	if output.Status != "success" {
		errCode := strings.TrimSpace(output.ErrorCode)
		errMessage := strings.TrimSpace(output.ErrorMessage)
		if errCode == "" {
			errCode = "EXECUTION_FAILED"
		}
		if errMessage == "" {
			errMessage = "module execution failed"
		}
		return adapter.Result{
			Status: output.Status,
			Result: output.Result,
			Logs:   append([]string(nil), output.Logs...),
		}, fmt.Errorf("%s: %s", errCode, errMessage)
	}

	var session *adapter.Session
	if output.Session != nil {
		session = &adapter.Session{
			Type:      output.Session.Type,
			Payload:   output.Session.Payload,
			ExpiresAt: output.Session.ExpiresAt,
		}
	}

	return adapter.Result{
		Status:  output.Status,
		Result:  output.Result,
		Session: session,
		Logs:    append([]string(nil), output.Logs...),
	}, nil
}
