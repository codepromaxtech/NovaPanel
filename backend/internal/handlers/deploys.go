package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/services"
)

type DeployHandler struct {
	service *services.DeployService
}

func NewDeployHandler(service *services.DeployService) *DeployHandler {
	return &DeployHandler{service: service}
}

// POST /api/v1/deployments — Create AND trigger a deployment
func (h *DeployHandler) Create(c *gin.Context) {
	var req models.CreateDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	d, err := h.service.TriggerDeploy(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, d)
}

// GET /api/v1/deployments
func (h *DeployHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("user_role").(string)
	resp, err := h.service.List(c.Request.Context(), userID, role, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/v1/deployments/:id
func (h *DeployHandler) GetByID(c *gin.Context) {
	d, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Deployment not found"})
		return
	}
	c.JSON(http.StatusOK, d)
}

// POST /api/v1/deployments/:id/redeploy
func (h *DeployHandler) Redeploy(c *gin.Context) {
	d, err := h.service.Redeploy(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, d)
}

// GET /api/v1/deployments/:id/logs
func (h *DeployHandler) GetLogs(c *gin.Context) {
	logs, status, err := h.service.GetLogs(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Deployment not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs, "status": status})
}
