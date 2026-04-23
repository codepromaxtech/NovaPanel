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

type DatabaseHandler struct {
	service *services.DatabaseService
	pool    *pgxpool.Pool
}

func NewDatabaseHandler(service *services.DatabaseService, pool *pgxpool.Pool) *DatabaseHandler {
	return &DatabaseHandler{service: service, pool: pool}
}

func (h *DatabaseHandler) Create(c *gin.Context) {
	var req models.CreateDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	db, err := h.service.Create(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	writeAudit(c.Request.Context(), h.pool, userID, "create", "database", db.ID.String(), c.ClientIP(),
		map[string]interface{}{"name": db.Name, "engine": db.Engine})
	c.JSON(http.StatusCreated, db)
}

func (h *DatabaseHandler) List(c *gin.Context) {
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

func (h *DatabaseHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("user_role").(string)
	if err := h.service.Delete(c.Request.Context(), id, userID, role); err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}
	writeAudit(c.Request.Context(), h.pool, userID, "delete", "database", id, c.ClientIP(), nil)
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Database deleted successfully"})
}
