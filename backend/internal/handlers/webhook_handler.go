package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/services"
)

// WebhookHandler handles incoming webhooks from GitHub/GitLab for auto-deploy
type WebhookHandler struct {
	pool      *pgxpool.Pool
	deploySvc *services.DeployService
}

func NewWebhookHandler(pool *pgxpool.Pool, deploySvc *services.DeployService) *WebhookHandler {
	return &WebhookHandler{pool: pool, deploySvc: deploySvc}
}

// HandleGitHub processes GitHub push webhook events
// POST /api/v1/webhooks/github/:app_id
func (h *WebhookHandler) HandleGitHub(c *gin.Context) {
	appID := c.Param("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id required"})
		return
	}

	// Read the request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// Fetch the application and its webhook secret
	var webhookSecret, gitBranch string
	var userID uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT user_id, COALESCE(webhook_secret, ''), COALESCE(git_branch, 'main')
		 FROM applications WHERE id = $1`, appID,
	).Scan(&userID, &webhookSecret, &gitBranch)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}

	// Require a webhook secret to be configured — reject requests for apps without one
	if webhookSecret == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "webhook secret not configured for this application"})
		return
	}

	// Validate HMAC signature
	signature := c.GetHeader("X-Hub-Signature-256")
	if signature == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing X-Hub-Signature-256 header"})
		return
	}
	if !validateGitHubSignature(body, webhookSecret, signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	// Parse the push event
	event := c.GetHeader("X-GitHub-Event")
	if event != "push" {
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("ignored event type: %s", event)})
		return
	}

	var payload struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Extract branch from ref (refs/heads/main -> main)
	pushBranch := strings.TrimPrefix(payload.Ref, "refs/heads/")

	// Only deploy if the pushed branch matches the app's configured branch
	if pushBranch != gitBranch {
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("ignored push to branch %s (configured: %s)", pushBranch, gitBranch)})
		return
	}

	// Trigger deployment
	d, err := h.deploySvc.TriggerDeploy(c.Request.Context(), userID, models.CreateDeploymentRequest{
		AppID:  appID,
		Branch: pushBranch,
	})
	if err != nil {
		log.Printf("❌ Webhook deploy failed for app %s: %v", appID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "deployment failed"})
		return
	}

	log.Printf("✅ Webhook triggered deployment %s for app %s (branch: %s)", d.ID, appID, pushBranch)
	c.JSON(http.StatusOK, gin.H{
		"message":       "deployment triggered",
		"deployment_id": d.ID,
		"branch":        pushBranch,
	})
}

// HandleGitLab processes GitLab push webhook events
// POST /api/v1/webhooks/gitlab/:app_id
func (h *WebhookHandler) HandleGitLab(c *gin.Context) {
	appID := c.Param("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app_id required"})
		return
	}

	// Fetch the application and its webhook secret
	var webhookSecret, gitBranch string
	var userID uuid.UUID
	err := h.pool.QueryRow(c.Request.Context(),
		`SELECT user_id, COALESCE(webhook_secret, ''), COALESCE(git_branch, 'main')
		 FROM applications WHERE id = $1`, appID,
	).Scan(&userID, &webhookSecret, &gitBranch)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}

	// Require a webhook secret to be configured
	if webhookSecret == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "webhook secret not configured for this application"})
		return
	}

	// Validate GitLab token header
	token := c.GetHeader("X-Gitlab-Token")
	if token != webhookSecret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Parse the push event
	var payload struct {
		ObjectKind string `json:"object_kind"`
		Ref        string `json:"ref"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	if payload.ObjectKind != "push" && payload.ObjectKind != "" {
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("ignored event type: %s", payload.ObjectKind)})
		return
	}

	// Extract branch from ref
	pushBranch := strings.TrimPrefix(payload.Ref, "refs/heads/")

	if pushBranch != gitBranch {
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("ignored push to branch %s (configured: %s)", pushBranch, gitBranch)})
		return
	}

	// Trigger deployment
	d, err := h.deploySvc.TriggerDeploy(c.Request.Context(), userID, models.CreateDeploymentRequest{
		AppID:  appID,
		Branch: pushBranch,
	})
	if err != nil {
		log.Printf("❌ GitLab webhook deploy failed for app %s: %v", appID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "deployment failed"})
		return
	}

	log.Printf("✅ GitLab webhook triggered deployment %s for app %s (branch: %s)", d.ID, appID, pushBranch)
	c.JSON(http.StatusOK, gin.H{
		"message":       "deployment triggered",
		"deployment_id": d.ID,
		"branch":        pushBranch,
	})
}

// validateGitHubSignature checks the HMAC-SHA256 signature from GitHub
func validateGitHubSignature(payload []byte, secret, signature string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
