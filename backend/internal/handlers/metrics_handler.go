package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/novapanel/novapanel/internal/services"
)

type MetricsHandler struct {
	metricsService *services.MetricsService
}

func NewMetricsHandler(metricsService *services.MetricsService) *MetricsHandler {
	return &MetricsHandler{metricsService: metricsService}
}

// LiveMetrics returns the current system resource snapshot.
func (h *MetricsHandler) LiveMetrics(c *gin.Context) {
	metrics, err := h.metricsService.CollectHostMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to collect metrics"})
		return
	}
	c.JSON(http.StatusOK, metrics)
}

// HistoryMetrics returns historical metrics for a server.
func (h *MetricsHandler) HistoryMetrics(c *gin.Context) {
	serverID := c.Param("id")
	hoursStr := c.DefaultQuery("hours", "1")
	hours, err := strconv.Atoi(hoursStr)
	if err != nil || hours < 1 {
		hours = 1
	}
	if hours > 168 { // max 7 days
		hours = 168
	}

	history, err := h.metricsService.GetHistory(c.Request.Context(), serverID, hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get metrics history"})
		return
	}
	if history == nil {
		history = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, gin.H{"data": history, "hours": hours})
}

// ServiceStatuses returns the health status of known services.
func (h *MetricsHandler) ServiceStatuses(c *gin.Context) {
	statuses := h.metricsService.GetServiceStatuses()
	c.JSON(http.StatusOK, gin.H{"services": statuses})
}
