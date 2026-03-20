package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type SettingsHandler struct {
	pool *pgxpool.Pool
}

func NewSettingsHandler(pool *pgxpool.Pool) *SettingsHandler {
	return &SettingsHandler{pool: pool}
}

func (h *SettingsHandler) GetProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	var name, email, role string
	err := h.pool.QueryRow(c.Request.Context(),
		`SELECT name, email, role FROM users WHERE id = $1`, userID,
	).Scan(&name, &email, &role)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": userID, "name": name, "email": email, "role": role})
}

func (h *SettingsHandler) UpdateProfile(c *gin.Context) {
	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	if req.Name != "" {
		h.pool.Exec(c.Request.Context(), `UPDATE users SET name = $1 WHERE id = $2`, req.Name, userID)
	}
	if req.Email != "" {
		h.pool.Exec(c.Request.Context(), `UPDATE users SET email = $1 WHERE id = $2`, req.Email, userID)
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to hash password"})
			return
		}
		h.pool.Exec(c.Request.Context(), `UPDATE users SET password_hash = $1 WHERE id = $2`, string(hash), userID)
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Profile updated"})
}
