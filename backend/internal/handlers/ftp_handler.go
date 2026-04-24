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

// GET /api/v1/servers/:id/ftp/:ftpID/keys
func (h *FTPHandler) ListKeys(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("user_role").(string)
	keys, err := h.service.ListSFTPKeys(c.Request.Context(), c.Param("ftpID"), c.Param("id"), userID, role)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": keys})
}

// POST /api/v1/servers/:id/ftp/:ftpID/keys
func (h *FTPHandler) AddKey(c *gin.Context) {
	var req services.AddSFTPKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("user_role").(string)
	key, err := h.service.AddSFTPKey(c.Request.Context(), c.Param("ftpID"), c.Param("id"), userID, role, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, key)
}

// DELETE /api/v1/servers/:id/ftp/:ftpID/keys/:keyID
func (h *FTPHandler) DeleteKey(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("user_role").(string)
	if err := h.service.DeleteSFTPKey(c.Request.Context(), c.Param("keyID"), c.Param("ftpID"), c.Param("id"), userID, role); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "SSH key removed"})
}
