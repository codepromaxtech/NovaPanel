package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/services"
)

type WAFHandler struct {
	service *services.WAFService
}

func NewWAFHandler(service *services.WAFService) *WAFHandler {
	return &WAFHandler{service: service}
}

func (h *WAFHandler) GetConfig(c *gin.Context) {
	serverID := c.Param("server_id")
	cfg, err := h.service.GetConfig(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func (h *WAFHandler) UpdateConfig(c *gin.Context) {
	serverID := c.Param("server_id")
	var req models.UpdateWAFConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	cfg, err := h.service.UpdateConfig(c.Request.Context(), serverID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func (h *WAFHandler) ListDisabledRules(c *gin.Context) {
	serverID := c.Param("server_id")
	rules, err := h.service.ListDisabledRules(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, rules)
}

func (h *WAFHandler) DisableRule(c *gin.Context) {
	serverID := c.Param("server_id")
	var req models.DisableWAFRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	rule, err := h.service.DisableRule(c.Request.Context(), serverID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, rule)
}

func (h *WAFHandler) EnableRule(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.EnableRule(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Rule re-enabled"})
}

func (h *WAFHandler) ListWhitelist(c *gin.Context) {
	serverID := c.Param("server_id")
	items, err := h.service.ListWhitelist(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *WAFHandler) AddWhitelist(c *gin.Context) {
	serverID := c.Param("server_id")
	var req models.CreateWAFWhitelistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	item, err := h.service.AddWhitelist(c.Request.Context(), serverID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *WAFHandler) RemoveWhitelist(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.RemoveWhitelist(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Whitelist entry removed"})
}

func (h *WAFHandler) ListLogs(c *gin.Context) {
	serverID := c.Param("server_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "50"))
	resp, err := h.service.ListLogs(c.Request.Context(), serverID, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
