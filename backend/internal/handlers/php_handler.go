package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/services"
)

type PHPHandler struct {
	service *services.PHPService
}

func NewPHPHandler(service *services.PHPService) *PHPHandler {
	return &PHPHandler{service: service}
}

// GET /api/v1/servers/:id/php
func (h *PHPHandler) ListVersions(c *gin.Context) {
	serverID := c.Param("id")
	versions, err := h.service.ListInstalled(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": versions})
}

// POST /api/v1/servers/:id/php
func (h *PHPHandler) Install(c *gin.Context) {
	serverID := c.Param("id")
	var req struct {
		Version string `json:"version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "version is required"})
		return
	}

	pv, err := h.service.Install(c.Request.Context(), serverID, req.Version)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"data": pv, "message": "PHP installation started"})
}

// PUT /api/v1/servers/:id/php/default
func (h *PHPHandler) SetDefault(c *gin.Context) {
	serverID := c.Param("id")
	var req struct {
		Version string `json:"version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "version is required"})
		return
	}

	if err := h.service.SetDefault(c.Request.Context(), serverID, req.Version); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Default PHP version updated"})
}

// DELETE /api/v1/servers/:id/php/:version
func (h *PHPHandler) Uninstall(c *gin.Context) {
	serverID := c.Param("id")
	version := c.Param("version")

	if err := h.service.Uninstall(c.Request.Context(), serverID, version); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "PHP version uninstall started"})
}

// PUT /api/v1/domains/:id/php
func (h *PHPHandler) SwitchDomain(c *gin.Context) {
	domainID := c.Param("id")
	var req struct {
		Version string `json:"version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "version is required"})
		return
	}

	if err := h.service.SwitchDomain(c.Request.Context(), domainID, req.Version); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "PHP version switch initiated"})
}
