package service

import (
	"encoding/json"
	"strings"
	"time"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/model"
)

func jobRunToResponse(item model.JobRunWithJob) dto.JobRunResponse {
	return buildJobRunResponse(item.JobRun, item.JobTypeKey, item.JobActionKey)
}

func buildJobRunResponse(item model.JobRun, jobTypeKey, jobActionKey string) dto.JobRunResponse {
	return dto.JobRunResponse{
		ID:           item.ID,
		JobID:        item.JobID,
		JobTypeKey:   jobTypeKey,
		JobActionKey: jobActionKey,
		AccountID:    item.AccountID,
		WorkerID:     item.WorkerID,
		Attempt:      item.Attempt,
		Status:       jobRunStatus(item.ErrorCode, item.ErrorMessage, item.EndedAt),
		Result:       item.Result,
		Logs:         decodeJobRunLogs(item.Logs),
		ErrorCode:    item.ErrorCode,
		ErrorMessage: item.ErrorMessage,
		StartedAt:    item.StartedAt,
		EndedAt:      item.EndedAt,
	}
}

func jobRunStatus(errorCode, errorMessage string, endedAt *time.Time) string {
	if endedAt == nil {
		return "running"
	}
	if strings.TrimSpace(errorCode) != "" || strings.TrimSpace(errorMessage) != "" {
		return "failed"
	}
	return "success"
}

func decodeJobRunLogs(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var logs []string
	if err := json.Unmarshal(raw, &logs); err != nil {
		return nil
	}
	if len(logs) == 0 {
		return nil
	}
	return logs
}
