package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/services"
)

type AppHandler struct {
	service *services.AppService
	pool    *pgxpool.Pool
}

func NewAppHandler(service *services.AppService, pool *pgxpool.Pool) *AppHandler {
	return &AppHandler{service: service, pool: pool}
}

// POST /api/v1/apps
func (h *AppHandler) Create(c *gin.Context) {
	var req models.CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	app, err := h.service.Create(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	writeAudit(c.Request.Context(), h.pool, userID, "create", "application", app.ID.String(), c.ClientIP(),
		map[string]interface{}{"name": app.Name, "runtime": app.Runtime})
	c.JSON(http.StatusCreated, app)
}

// GET /api/v1/apps
func (h *AppHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("user_role").(string)
	resp, err := h.service.List(c.Request.Context(), userID, role, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/v1/apps/:id
func (h *AppHandler) GetByID(c *gin.Context) {
	app, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Application not found"})
		return
	}
	c.JSON(http.StatusOK, app)
}

// PUT /api/v1/apps/:id
func (h *AppHandler) Update(c *gin.Context) {
	var req models.UpdateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("user_role").(string)
	app, err := h.service.Update(c.Request.Context(), c.Param("id"), req, userID, role)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, app)
}

// DELETE /api/v1/apps/:id
func (h *AppHandler) Delete(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("user_role").(string)
	appID := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), appID, userID, role); err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}
	writeAudit(c.Request.Context(), h.pool, userID, "delete", "application", appID, c.ClientIP(), nil)
	c.JSON(http.StatusOK, gin.H{"message": "Application deleted"})
}
