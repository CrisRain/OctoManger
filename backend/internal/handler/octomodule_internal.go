package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"octomanger/backend/internal/dto"
	"octomanger/backend/internal/service"
	"octomanger/backend/pkg/response"
)

type OctoModuleInternalHandler struct {
	svc service.OctoModuleInternalService
}

func NewOctoModuleInternalHandler(svc service.OctoModuleInternalService) *OctoModuleInternalHandler {
	return &OctoModuleInternalHandler{svc: svc}
}

func (h *OctoModuleInternalHandler) GetAccount(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid account id")
		return
	}
	item, err := h.svc.GetAccount(c.Request.Context(), id)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *OctoModuleInternalHandler) GetAccountByIdentifier(c *gin.Context) {
	var query dto.OctoModuleInternalFindAccountQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.GetAccountByIdentifier(c.Request.Context(), query.TypeKey, query.Identifier)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *OctoModuleInternalHandler) PatchAccountSpec(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid account id")
		return
	}
	var req dto.OctoModuleInternalPatchAccountSpecRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.svc.PatchAccountSpec(c.Request.Context(), id, &req)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *OctoModuleInternalHandler) GetLatestEmail(c *gin.Context) {
	id, err := parseUint64Param(c, "id")
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid email account id")
		return
	}
	result, err := h.svc.GetLatestEmail(c.Request.Context(), id, strings.TrimSpace(c.Query("mailbox")))
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.Success(c, result)
}
