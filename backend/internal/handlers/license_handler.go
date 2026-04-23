package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/novapanel/novapanel/internal/services"
)

type LicenseHandler struct {
	svc *services.LicenseService
}

func NewLicenseHandler(svc *services.LicenseService) *LicenseHandler {
	return &LicenseHandler{svc: svc}
}

// GetStatus returns the current license status.
func (h *LicenseHandler) GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.svc.GetStatus())
}

// Activate saves a new license key and re-verifies immediately.
func (h *LicenseHandler) Activate(c *gin.Context) {
	var req struct {
		LicenseKey string `json:"license_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "license_key is required"})
		return
	}

	key := strings.TrimSpace(req.LicenseKey)
	if err := h.svc.SaveLicenseKey(c.Request.Context(), key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	status := h.svc.GetStatus()
	if !status.Valid || status.PlanType == "community" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "invalid_license",
			"message": status.Message,
			"status":  status,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "License activated successfully",
		"status":  status,
	})
}

// Refresh forces an immediate re-verification against the license server.
func (h *LicenseHandler) Refresh(c *gin.Context) {
	if err := h.svc.Verify(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.svc.GetStatus())
}
