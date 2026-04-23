package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/novapanel/novapanel/internal/middleware"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/services"
)

type AuthHandler struct {
	service      *services.AuthService
	smtpSvc      *services.SMTPService
	frontendURL  string
	loginLimiter *middleware.LoginLimiter
}

func NewAuthHandler(service *services.AuthService, smtpSvc *services.SMTPService, frontendURL string, loginLimiter *middleware.LoginLimiter) *AuthHandler {
	return &AuthHandler{service: service, smtpSvc: smtpSvc, frontendURL: frontendURL, loginLimiter: loginLimiter}
}

// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}
	resp, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusConflict, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}
	resp, err := h.service.Login(c.Request.Context(), req, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if h.loginLimiter != nil {
			h.loginLimiter.RecordFailure(c.ClientIP())
		}
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: err.Error()})
		return
	}
	if h.loginLimiter != nil {
		h.loginLimiter.RecordSuccess(c.ClientIP())
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	user, err := h.service.GetUserByID(c.Request.Context(), userID.String())
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	resp, err := h.service.Refresh(c.Request.Context(), userID.String(), c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	tokenID, _ := c.Get("token_id")
	if id, ok := tokenID.(string); ok && id != "" {
		_ = h.service.RevokeToken(c.Request.Context(), id, 0)
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Logged out successfully"})
}

// ─── 2FA / TOTP ──────────────────────────────────────────────────────────────

// POST /api/v1/auth/2fa/setup
func (h *AuthHandler) TOTPSetup(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	resp, err := h.service.TOTPSetup(c.Request.Context(), userID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// POST /api/v1/auth/2fa/enable
func (h *AuthHandler) TOTPEnable(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	backupCodes, err := h.service.TOTPEnable(c.Request.Context(), userID.String(), req.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":      "Two-factor authentication enabled",
		"backup_codes": backupCodes,
	})
}

// POST /api/v1/auth/2fa/verify  (unauthenticated — post-login step when 2FA required)
func (h *AuthHandler) TOTPVerify(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
		Code  string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.service.TOTPVerify(c.Request.Context(), req.Email, req.Code, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// DELETE /api/v1/auth/2fa
func (h *AuthHandler) TOTPDisable(c *gin.Context) {
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	if err := h.service.TOTPDisable(c.Request.Context(), userID.String(), req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Two-factor authentication disabled"})
}

// ─── Password Reset ───────────────────────────────────────────────────────────

// POST /api/v1/auth/forgot-password
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rawToken, email, err := h.service.CreatePasswordReset(c.Request.Context(), req.Email)
	if err == nil && rawToken != "" && h.smtpSvc != nil {
		resetURL := h.frontendURL + "/reset-password?token=" + rawToken
		go h.smtpSvc.SendPasswordReset(email, resetURL)
	}
	// Always 200 — prevents email enumeration
	c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset link has been sent"})
}

// POST /api/v1/auth/reset-password
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// ─── Session Management ───────────────────────────────────────────────────────

// GET /api/v1/auth/sessions
func (h *AuthHandler) ListSessions(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	sessions, err := h.service.ListSessions(c.Request.Context(), userID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sessions})
}

// DELETE /api/v1/auth/sessions/:id
func (h *AuthHandler) RevokeSession(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	if err := h.service.RevokeSession(c.Request.Context(), c.Param("id"), userID.String()); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Session revoked"})
}

// DELETE /api/v1/auth/sessions  (revoke all other sessions)
func (h *AuthHandler) RevokeAllOtherSessions(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	currentJTI, _ := c.Get("token_id")
	jti, _ := currentJTI.(string)
	if err := h.service.RevokeAllOtherSessions(c.Request.Context(), jti, userID.String()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "All other sessions revoked"})
}
