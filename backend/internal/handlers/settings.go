package handlers

import (
	"net/http"
	"strings"

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
	var firstName, lastName, email, role string
	err := h.pool.QueryRow(c.Request.Context(),
		`SELECT COALESCE(first_name,''), COALESCE(last_name,''), email, role FROM users WHERE id = $1`, userID,
	).Scan(&firstName, &lastName, &email, &role)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":         userID,
		"first_name": firstName,
		"last_name":  lastName,
		"name":       firstName + " " + lastName,
		"email":      email,
		"role":       role,
	})
}

func (h *SettingsHandler) UpdateProfile(c *gin.Context) {
	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	if req.Name != "" {
		parts := strings.SplitN(strings.TrimSpace(req.Name), " ", 2)
		firstName := parts[0]
		lastName := ""
		if len(parts) > 1 {
			lastName = parts[1]
		}
		if _, err := h.pool.Exec(c.Request.Context(),
			`UPDATE users SET first_name = $1, last_name = $2 WHERE id = $3`, firstName, lastName, userID); err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update name"})
			return
		}
	}
	if req.Email != "" {
		if _, err := h.pool.Exec(c.Request.Context(),
			`UPDATE users SET email = $1 WHERE id = $2`, req.Email, userID); err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update email"})
			return
		}
	}
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to hash password"})
			return
		}
		if _, err := h.pool.Exec(c.Request.Context(),
			`UPDATE users SET password_hash = $1 WHERE id = $2`, string(hash), userID); err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update password"})
			return
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Profile updated"})
}
