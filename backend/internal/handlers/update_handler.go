package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/novapanel/novapanel/internal/services"
)

// UpdateHandler exposes update-check and apply endpoints (admin-only).
type UpdateHandler struct {
	svc *services.UpdateService
}

func NewUpdateHandler(svc *services.UpdateService) *UpdateHandler {
	return &UpdateHandler{svc: svc}
}

// GetStatus returns the cached update status without hitting GitHub.
func (h *UpdateHandler) GetStatus(c *gin.Context) {
	status := h.svc.GetStatus(c.Request.Context())
	c.JSON(http.StatusOK, status)
}

// CheckNow forces an immediate GitHub check and returns the new status.
func (h *UpdateHandler) CheckNow(c *gin.Context) {
	status, err := h.svc.CheckForUpdates(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

// ApplyUpdate pulls the latest images and restarts containers.
// Returns 202 immediately; progress streamed via WebSocket (update:progress / update:complete / update:error).
func (h *UpdateHandler) ApplyUpdate(c *gin.Context) {
	if err := h.svc.ApplyUpdate(c.Request.Context()); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "Update started — follow progress via WebSocket"})
}
