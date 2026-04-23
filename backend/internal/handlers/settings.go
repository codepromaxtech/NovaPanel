package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	novacrypto "github.com/novapanel/novapanel/internal/crypto"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/services"
	"golang.org/x/crypto/bcrypt"
)

type SettingsHandler struct {
	pool         *pgxpool.Pool
	smtpService  *services.SMTPService
	cryptoKey    string
}

func NewSettingsHandler(pool *pgxpool.Pool, smtpService *services.SMTPService, cryptoKey string) *SettingsHandler {
	return &SettingsHandler{pool: pool, smtpService: smtpService, cryptoKey: cryptoKey}
}

// GetSystemSettings returns SMTP and Stripe config (admin only, secrets masked)
func (h *SettingsHandler) GetSystemSettings(c *gin.Context) {
	rows, err := h.pool.Query(c.Request.Context(), `SELECT key, value, encrypted FROM system_settings ORDER BY key`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to load settings"})
		return
	}
	defer rows.Close()

	result := map[string]string{}
	for rows.Next() {
		var key, value string
		var encrypted bool
		if err := rows.Scan(&key, &value, &encrypted); err != nil {
			continue
		}
		if encrypted && value != "" {
			plain, err := novacrypto.Decrypt(value, []byte(h.cryptoKey))
			if err == nil && plain != "" {
				result[key] = "••••••••" // mask — confirm it is set
			} else {
				result[key] = ""
			}
		} else {
			result[key] = value
		}
	}
	c.JSON(http.StatusOK, result)
}

// UpdateSystemSettings saves SMTP and Stripe settings, reloads live services
func (h *SettingsHandler) UpdateSystemSettings(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	encryptedKeys := map[string]bool{
		"smtp_password":         true,
		"stripe_secret_key":     true,
		"stripe_webhook_secret": true,
	}

	for key, value := range req {
		if value == "••••••••" {
			continue // client echoed back the mask — skip (no change)
		}
		storeValue := value
		if encryptedKeys[key] && value != "" {
			enc, err := novacrypto.Encrypt(value, []byte(h.cryptoKey))
			if err != nil {
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Encryption failed"})
				return
			}
			storeValue = enc
		}
		if _, err := h.pool.Exec(c.Request.Context(),
			`INSERT INTO system_settings (key, value, encrypted, updated_at)
			 VALUES ($1, $2, $3, NOW())
			 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			key, storeValue, encryptedKeys[key],
		); err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to save " + key})
			return
		}
	}

	// Reload SMTP live so new settings take effect immediately
	if h.smtpService != nil {
		h.smtpService.ReloadFromDB(c.Request.Context(), h.pool, h.cryptoKey)
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Settings saved"})
}

// TestSMTP sends a test email to the requesting user
func (h *SettingsHandler) TestSMTP(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	var email string
	if err := h.pool.QueryRow(c.Request.Context(),
		`SELECT email FROM users WHERE id = $1`, userID).Scan(&email); err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "User not found"})
		return
	}
	if h.smtpService == nil || !h.smtpService.Enabled() {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "SMTP is not configured"})
		return
	}
	if err := h.smtpService.SendTestEmail(email); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "SMTP test failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Test email sent to " + email})
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
