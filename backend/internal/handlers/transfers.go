package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/services"
)

type TransferHandler struct {
	service *services.TransferService
}

func NewTransferHandler(service *services.TransferService) *TransferHandler {
	return &TransferHandler{service: service}
}

// POST /transfers
func (h *TransferHandler) Create(c *gin.Context) {
	var req models.CreateTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	job, err := h.service.CreateTransfer(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, job)
}

// GET /transfers
func (h *TransferHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	userID := c.MustGet("user_id").(uuid.UUID)
	resp, err := h.service.ListTransfers(c.Request.Context(), userID, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GET /transfers/:id
func (h *TransferHandler) Get(c *gin.Context) {
	id := c.Param("id")
	job, err := h.service.GetTransfer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, job)
}

// POST /transfers/:id/cancel
func (h *TransferHandler) Cancel(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.CancelTransfer(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Transfer cancelled"})
}

// POST /transfers/:id/retry
func (h *TransferHandler) Retry(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.RetryTransfer(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Transfer retried"})
}

// DELETE /transfers/:id
func (h *TransferHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteTransfer(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Transfer deleted"})
}

// POST /transfers/preview — dry-run
func (h *TransferHandler) Preview(c *gin.Context) {
	var req models.CreateTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	job, err := h.service.PreviewTransfer(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, job)
}

// POST /transfers/disk-usage
func (h *TransferHandler) DiskUsage(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Path     string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.service.DiskUsage(c.Request.Context(), req.ServerID, req.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// ──── Schedules ────

// GET /transfers/schedules
func (h *TransferHandler) ListSchedules(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	scheds, err := h.service.ListSchedules(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": scheds})
}

// POST /transfers/schedules
func (h *TransferHandler) CreateSchedule(c *gin.Context) {
	var req models.CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	sched, err := h.service.CreateSchedule(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, sched)
}

// DELETE /transfers/schedules/:id
func (h *TransferHandler) DeleteSchedule(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteSchedule(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Schedule deleted"})
}

// PUT /transfers/schedules/:id/toggle
func (h *TransferHandler) ToggleSchedule(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	if err := h.service.ToggleSchedule(c.Request.Context(), id, req.IsActive); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Schedule updated"})
}
