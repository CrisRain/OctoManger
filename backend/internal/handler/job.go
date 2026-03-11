package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/service"
	"octomanger/backend/pkg/response"
)

type JobHandler struct {
	svc service.JobService
}

func NewJobHandler(svc service.JobService) *JobHandler {
	return &JobHandler{svc: svc}
}

func (h *JobHandler) List(c *gin.Context) {
	limit, offset, err := resolvePagination(c, 50, 500)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid pagination parameters")
		return
	}
	result, err := h.svc.ListPaged(c.Request.Context(), limit, offset)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *JobHandler) Summary(c *gin.Context) {
	result, err := h.svc.Summary(c.Request.Context())
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *JobHandler) ListRuns(c *gin.Context) {
	limit, offset, err := resolvePagination(c, 50, 500)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid pagination parameters")
		return
	}

	var jobID *uint64
	if rawID := strings.TrimSpace(c.Param("id")); rawID != "" {
		parsed, parseErr := strconv.ParseUint(rawID, 10, 64)
		if parseErr != nil || parsed == 0 {
			response.Fail(c, http.StatusBadRequest, "invalid job id")
			return
		}
		jobID = &parsed
	} else if rawJobID := strings.TrimSpace(c.Query("job_id")); rawJobID != "" {
		parsed, parseErr := strconv.ParseUint(rawJobID, 10, 64)
		if parseErr != nil || parsed == 0 {
			response.Fail(c, http.StatusBadRequest, "invalid job id")
			return
		}
		jobID = &parsed
	}

	result, err := h.svc.ListRuns(c.Request.Context(), service.JobRunListFilter{
		JobID:     jobID,
		TypeKey:   strings.TrimSpace(c.Query("type_key")),
		ActionKey: strings.TrimSpace(c.Query("action_key")),
		WorkerID:  strings.TrimSpace(c.Query("worker_id")),
		Outcome:   strings.TrimSpace(c.Query("outcome")),
	}, limit, offset)
	if err != nil {
		response.FailWithError(c, err)
		return
	}

	response.Success(c, result)
}

func (h *JobHandler) Get(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid job id")
		return
	}
	item, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *JobHandler) Create(c *gin.Context) {
	var req dto.CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *JobHandler) Patch(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid job id")
		return
	}
	var req dto.PatchJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.Patch(c.Request.Context(), id, &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *JobHandler) PostAction(c *gin.Context) {
	rawID := c.Param("id")

	if idValue, ok := parseColonActionParam(rawID, "cancel"); ok {
		parsed, err := strconv.ParseUint(idValue, 10, 64)
		if err != nil || parsed == 0 {
			response.Fail(c, http.StatusBadRequest, "invalid job id")
			return
		}
		item, err := h.svc.Cancel(c.Request.Context(), parsed)
		if err != nil {
			response.FailWithError(c, err)
			return
		}
		response.Success(c, item)
		return
	}

	if idValue, ok := parseColonActionParam(rawID, "retry"); ok {
		parsed, err := strconv.ParseUint(idValue, 10, 64)
		if err != nil || parsed == 0 {
			response.Fail(c, http.StatusBadRequest, "invalid job id")
			return
		}
		item, err := h.svc.Retry(c.Request.Context(), parsed)
		if err != nil {
			response.FailWithError(c, err)
			return
		}
		response.Success(c, item)
		return
	}

	response.Fail(c, http.StatusBadRequest, "unknown action; expected :cancel or :retry")
}

func (h *JobHandler) Delete(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid job id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
