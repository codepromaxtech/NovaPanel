package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/services"
)

type BillingHandler struct {
	service *services.BillingService
}

func NewBillingHandler(service *services.BillingService) *BillingHandler {
	return &BillingHandler{service: service}
}

func (h *BillingHandler) CreatePlan(c *gin.Context) {
	var req models.CreateBillingPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	plan, err := h.service.CreatePlan(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, plan)
}

func (h *BillingHandler) ListPlans(c *gin.Context) {
	plans, err := h.service.ListPlans(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, plans)
}

func (h *BillingHandler) ListInvoices(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	invoices, err := h.service.ListInvoices(c.Request.Context(), userID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, invoices)
}
