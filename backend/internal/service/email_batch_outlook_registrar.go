package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/worker/bridge"
)

const outlookBatchRegistrarScriptDir = "_email_batch_outlook"

type OutlookEmailBatchRegistrar struct {
	runner     bridge.PythonBridge
	scriptPath string
}

func NewOutlookEmailBatchRegistrar(runner bridge.PythonBridge, moduleDir string) *OutlookEmailBatchRegistrar {
	trimmedDir := strings.TrimSpace(moduleDir)
	scriptPath := filepath.Join(trimmedDir, outlookBatchRegistrarScriptDir, "main.py")
	return &OutlookEmailBatchRegistrar{
		runner:     runner,
		scriptPath: scriptPath,
	}
}

func (r *OutlookEmailBatchRegistrar) Prepare(
	ctx context.Context,
	input dto.BatchRegisterEmailRequest,
) (EmailBatchPreparedResult, error) {
	if strings.TrimSpace(r.scriptPath) == "" {
		return EmailBatchPreparedResult{}, errors.New("outlook batch registrar script path is not configured")
	}

	provider := strings.ToLower(strings.TrimSpace(input.Provider))
	switch provider {
	case "", "outlook", "hotmail", "live", "msn":
	default:
		return EmailBatchPreparedResult{}, fmt.Errorf("provider %q is not supported yet", input.Provider)
	}

	operator := strings.TrimSpace(input.Operator)
	if operator == "" {
		operator = "batch-operator"
	}

	params := map[string]any{
		"provider":    provider,
		"count":       input.Count,
		"prefix":      strings.TrimSpace(input.Prefix),
		"domain":      strings.TrimSpace(input.Domain),
		"start_index": input.StartIndex,
		"status":      input.Status,
	}
	if len(input.Options) > 0 {
		params["options"] = input.Options
	}
	if len(input.GraphDefaults) > 0 {
		params["graph_defaults"] = string(input.GraphDefaults)
	}

	requestID := fmt.Sprintf("batch-email-register:%s", operator)
	if token, tokenErr := generateSecureToken(6); tokenErr == nil {
		requestID = fmt.Sprintf("batch-email-register:%s:%s", operator, token)
	}

	output, err := r.runner.ExecuteWithScript(ctx, r.scriptPath, bridge.Input{
		Action: "BATCH_REGISTER_EMAIL",
		Account: bridge.InputAccount{
			Identifier: operator,
			Spec: map[string]any{
				"provider": provider,
			},
		},
		Params: params,
		Context: bridge.InputContext{
			RequestID: requestID,
			Protocol:  "ndjson.v1",
		},
	})
	if err != nil {
		return EmailBatchPreparedResult{}, err
	}

	if !strings.EqualFold(output.Status, "success") {
		message := strings.TrimSpace(output.ErrorMessage)
		if message == "" {
			message = "outlook batch register returned error"
		}
		code := strings.TrimSpace(output.ErrorCode)
		if code != "" {
			return EmailBatchPreparedResult{}, fmt.Errorf("%s: %s", code, message)
		}
		return EmailBatchPreparedResult{}, errors.New(message)
	}

	candidates, failures, parsedProvider, requested, err := parseBatchRegisterCandidates(output.Result)
	if err != nil {
		return EmailBatchPreparedResult{}, err
	}

	if parsedProvider == "" {
		parsedProvider = "outlook"
	}

	return EmailBatchPreparedResult{
		Requested:  requested,
		Provider:   parsedProvider,
		Candidates: candidates,
		Failures:   failures,
	}, nil
}

var _ EmailBatchRegistrar = (*OutlookEmailBatchRegistrar)(nil)
