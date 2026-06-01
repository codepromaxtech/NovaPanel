package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/services"
)

type deployLicenseChecker interface {
	GetStatus() services.LicenseStatus
}

type DeployHandler struct {
	service *services.DeployService
	license deployLicenseChecker
}

func NewDeployHandler(service *services.DeployService, license deployLicenseChecker) *DeployHandler {
	return &DeployHandler{service: service, license: license}
}

// POST /api/v1/deployments — Create AND trigger a deployment
func (h *DeployHandler) Create(c *gin.Context) {
	var req models.CreateDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Multi-server fan-out requires enterprise licence.
	if len(req.TargetServerIDs) > 1 {
		role, _ := c.Get("user_role")
		if role != "admin" && h.license != nil {
			st := h.license.GetStatus()
			if !st.Features.AllowMultiDeploy {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error":   "upgrade_required",
					"feature": "allow_multi_deploy",
					"message": "Multi-server deployment requires an Enterprise plan.",
				})
				return
			}
		}
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

// GET /api/v1/deployments/:id/targets — per-server results for multi-server deploys
func (h *DeployHandler) ListTargets(c *gin.Context) {
	targets, err := h.service.ListTargets(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"targets": targets})
}
