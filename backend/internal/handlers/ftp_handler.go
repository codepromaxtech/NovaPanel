package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/novapanel/novapanel/internal/services"
)

type FTPHandler struct {
	service *services.FTPService
}

func NewFTPHandler(service *services.FTPService) *FTPHandler {
	return &FTPHandler{service: service}
}

// GET /api/v1/servers/:id/ftp  or  GET /api/v1/ftp
func (h *FTPHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("user_role").(string)
	serverID := c.Param("id") // empty string when called from /ftp route
	accounts, err := h.service.List(c.Request.Context(), userID, role, serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": accounts})
}

// POST /api/v1/servers/:id/ftp
func (h *FTPHandler) Create(c *gin.Context) {
	var req services.CreateFTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ServerID = c.Param("id")
	userID := c.MustGet("user_id").(uuid.UUID)
	acc, err := h.service.Create(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, acc)
}

// DELETE /api/v1/servers/:id/ftp/:ftpID
func (h *FTPHandler) Delete(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("user_role").(string)
	if err := h.service.Delete(c.Request.Context(), c.Param("ftpID"), userID, role); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "FTP account deleted"})
}
