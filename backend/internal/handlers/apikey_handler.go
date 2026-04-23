package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/novapanel/novapanel/internal/services"
)

type APIKeyHandler struct {
	service *services.APIKeyService
}

func NewAPIKeyHandler(service *services.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{service: service}
}

// GET /api/v1/settings/api-keys
func (h *APIKeyHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	keys, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": keys})
}

// POST /api/v1/settings/api-keys
func (h *APIKeyHandler) Create(c *gin.Context) {
	var req services.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	created, err := h.service.Create(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, created)
}

// DELETE /api/v1/settings/api-keys/:id
func (h *APIKeyHandler) Revoke(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	if err := h.service.Revoke(c.Request.Context(), c.Param("id"), userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "API key revoked"})
}
