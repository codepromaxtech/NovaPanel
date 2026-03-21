package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/queue"
	"github.com/novapanel/novapanel/internal/services"
)

type ServerHandler struct {
	service   *services.ServerService
	taskQueue *queue.TaskQueue
}

func NewServerHandler(service *services.ServerService, taskQueue ...*queue.TaskQueue) *ServerHandler {
	h := &ServerHandler{service: service}
	if len(taskQueue) > 0 {
		h.taskQueue = taskQueue[0]
	}
	return h
}

// POST /api/v1/servers
func (h *ServerHandler) Create(c *gin.Context) {
	var req models.CreateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	server, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	// Auto-trigger provisioning if modules are selected
	if h.taskQueue != nil && len(req.Modules) > 0 {
		payload := map[string]interface{}{
			"server_id": server.ID.String(),
			"modules":   req.Modules,
		}
		taskID, err := h.taskQueue.Enqueue(c.Request.Context(), "server_setup", payload, 1, server.ID.String(), "")
		if err != nil {
			log.Printf("Warning: failed to enqueue server setup task: %v", err)
		} else {
			log.Printf("⚡ Enqueued server_setup task %s for server %s with modules: %v", taskID, server.ID, req.Modules)
		}
	}

	c.JSON(http.StatusCreated, server)
}

// GET /api/v1/servers
func (h *ServerHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	resp, err := h.service.List(c.Request.Context(), page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GET /api/v1/servers/:id
func (h *ServerHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	server, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, server)
}

// POST /api/v1/servers/:id/heartbeat
func (h *ServerHandler) Heartbeat(c *gin.Context) {
	id := c.Param("id")
	var metrics models.ServerMetrics
	if err := c.ShouldBindJSON(&metrics); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid metrics data"})
		return
	}

	if err := h.service.UpdateHeartbeat(c.Request.Context(), id, metrics); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Heartbeat received"})
}

// DELETE /api/v1/servers/:id
func (h *ServerHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Server deleted successfully"})
}

// GET /api/v1/dashboard/stats
func (h *ServerHandler) DashboardStats(c *gin.Context) {
	stats, err := h.service.GetDashboardStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// POST /api/v1/servers/test-connection
func (h *ServerHandler) TestConnection(c *gin.Context) {
	var req models.TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	output, err := h.service.TestConnection(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "output": output})
}

// PUT /api/v1/servers/:id
func (h *ServerHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	server, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, server)
}

// GET /api/v1/servers/:id/metrics/latest
func (h *ServerHandler) LatestMetrics(c *gin.Context) {
	id := c.Param("id")
	metrics, err := h.service.GetLatestMetrics(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"available": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"available": true, "metrics": metrics})
}
