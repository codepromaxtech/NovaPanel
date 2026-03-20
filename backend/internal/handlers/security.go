package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/services"
)

type SecurityHandler struct {
	service *services.SecurityService
}

func NewSecurityHandler(service *services.SecurityService) *SecurityHandler {
	return &SecurityHandler{service: service}
}

func (h *SecurityHandler) CreateRule(c *gin.Context) {
	var req models.CreateFirewallRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	rule, err := h.service.CreateRule(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, rule)
}

func (h *SecurityHandler) ListRules(c *gin.Context) {
	serverID := c.Query("server_id")
	rules, err := h.service.ListRules(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, rules)
}

func (h *SecurityHandler) DeleteRule(c *gin.Context) {
	if err := h.service.DeleteRule(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Rule deleted"})
}

func (h *SecurityHandler) ListEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "50"))
	resp, err := h.service.ListEvents(c.Request.Context(), page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
