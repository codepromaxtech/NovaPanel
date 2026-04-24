package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/novapanel/novapanel/internal/services"
)

type ResellerHandler struct {
	service *services.ResellerService
}

func NewResellerHandler(service *services.ResellerService) *ResellerHandler {
	return &ResellerHandler{service: service}
}

// POST /api/v1/reseller/clients
func (h *ResellerHandler) AllocateClient(c *gin.Context) {
	var req services.AllocateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resellerID := c.MustGet("user_id").(uuid.UUID)
	client, err := h.service.AllocateClient(c.Request.Context(), resellerID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, client)
}

// GET /api/v1/reseller/clients
func (h *ResellerHandler) ListClients(c *gin.Context) {
	resellerID := c.MustGet("user_id").(uuid.UUID)
	clients, err := h.service.ListClients(c.Request.Context(), resellerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if clients == nil {
		clients = []services.ClientWithQuota{}
	}
	c.JSON(http.StatusOK, gin.H{"data": clients})
}

// PUT /api/v1/reseller/clients/:id
func (h *ResellerHandler) UpdateClientQuota(c *gin.Context) {
	var req services.AllocateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resellerID := c.MustGet("user_id").(uuid.UUID)
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client id"})
		return
	}
	quota, err := h.service.UpdateClientQuota(c.Request.Context(), resellerID, clientID, req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, quota)
}

// DELETE /api/v1/reseller/clients/:id
func (h *ResellerHandler) DeleteClient(c *gin.Context) {
	resellerID := c.MustGet("user_id").(uuid.UUID)
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client id"})
		return
	}
	if err := h.service.DeleteClient(c.Request.Context(), resellerID, clientID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Client removed"})
}

// GET /api/v1/reseller/clients/:id/usage
func (h *ResellerHandler) GetClientUsage(c *gin.Context) {
	resellerID := c.MustGet("user_id").(uuid.UUID)
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client id"})
		return
	}
	usage, err := h.service.GetClientUsage(c.Request.Context(), resellerID, clientID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, usage)
}
