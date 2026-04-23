package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/config"
)

type StripeHandler struct {
	pool          *pgxpool.Pool
	cfg           *config.Config
	stripeBaseURL string
}

func NewStripeHandler(pool *pgxpool.Pool, cfg *config.Config) *StripeHandler {
	return &StripeHandler{pool: pool, cfg: cfg, stripeBaseURL: "https://api.stripe.com/v1"}
}

// POST /api/v1/billing/checkout
func (h *StripeHandler) CreateCheckout(c *gin.Context) {
	if h.cfg.StripeSecretKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Stripe billing not configured"})
		return
	}

	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	// Get plan stripe price ID
	var priceID, planName string
	var stripeCustomerID *string
	if err := h.pool.QueryRow(c.Request.Context(),
		`SELECT COALESCE(stripe_price_id, ''), name FROM billing_plans WHERE id = $1`, req.PlanID,
	).Scan(&priceID, &planName); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}
	if priceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plan has no Stripe price configured"})
		return
	}

	// Get or create Stripe customer
	h.pool.QueryRow(c.Request.Context(),
		`SELECT stripe_customer_id FROM users WHERE id = $1`, userID,
	).Scan(&stripeCustomerID)

	successURL := h.cfg.FrontendURL + "/billing?success=1"
	cancelURL := h.cfg.FrontendURL + "/billing?cancelled=1"

	params := map[string]string{
		"mode":                               "subscription",
		"line_items[0][price]":               priceID,
		"line_items[0][quantity]":            "1",
		"success_url":                        successURL,
		"cancel_url":                         cancelURL,
		"metadata[user_id]":                  userID.String(),
		"metadata[plan_id]":                  req.PlanID,
		"subscription_data[metadata][user_id]": userID.String(),
	}
	if stripeCustomerID != nil && *stripeCustomerID != "" {
		params["customer"] = *stripeCustomerID
	}

	session, err := h.stripePost("/checkout/sessions", params)
	if err != nil {
		log.Printf("Stripe checkout error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create checkout session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"checkout_url": session["url"],
		"session_id":   session["id"],
	})
}

// GET /api/v1/billing/subscription
func (h *StripeHandler) GetSubscription(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var planName, planType string
	var expiresAt *time.Time
	var subscriptionID *string
	err := h.pool.QueryRow(c.Request.Context(),
		`SELECT bp.name, bp.plan_type, u.plan_expires_at, u.stripe_subscription_id
		 FROM users u LEFT JOIN billing_plans bp ON bp.id = u.plan_id
		 WHERE u.id = $1`, userID,
	).Scan(&planName, &planType, &expiresAt, &subscriptionID)

	if err != nil || planName == "" {
		c.JSON(http.StatusOK, gin.H{
			"plan":   "Community",
			"type":   "community",
			"status": "active",
		})
		return
	}

	resp := gin.H{
		"plan": planName,
		"type": planType,
	}
	if expiresAt != nil {
		resp["expires_at"] = expiresAt.Format(time.RFC3339)
		if time.Now().After(*expiresAt) {
			resp["status"] = "expired"
		} else {
			resp["status"] = "active"
		}
	} else {
		resp["status"] = "active"
	}
	if subscriptionID != nil {
		resp["subscription_id"] = *subscriptionID
	}
	c.JSON(http.StatusOK, resp)
}

// DELETE /api/v1/billing/subscription
func (h *StripeHandler) CancelSubscription(c *gin.Context) {
	if h.cfg.StripeSecretKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Stripe billing not configured"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	var subscriptionID string
	if err := h.pool.QueryRow(c.Request.Context(),
		`SELECT COALESCE(stripe_subscription_id, '') FROM users WHERE id = $1`, userID,
	).Scan(&subscriptionID); err != nil || subscriptionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active subscription"})
		return
	}

	// Cancel at period end
	_, err := h.stripePost(fmt.Sprintf("/subscriptions/%s", subscriptionID), map[string]string{
		"cancel_at_period_end": "true",
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "subscription will be cancelled at end of billing period"})
}

// POST /api/v1/billing/webhook  (no JWT — Stripe-Signature validated)
func (h *StripeHandler) Webhook(c *gin.Context) {
	if h.cfg.StripeWebhookSecret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhook secret not configured"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	if !h.validateStripeSignature(body, c.GetHeader("Stripe-Signature"), h.cfg.StripeWebhookSecret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	var event struct {
		Type string                 `json:"type"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	go h.processWebhookEvent(event.Type, event.Data)
	c.JSON(http.StatusOK, gin.H{"received": true})
}

func (h *StripeHandler) processWebhookEvent(eventType string, data map[string]interface{}) {
	obj, _ := data["object"].(map[string]interface{})
	if obj == nil {
		return
	}

	switch eventType {
	case "checkout.session.completed":
		h.handleCheckoutCompleted(obj)
	case "invoice.paid":
		h.handleInvoicePaid(obj)
	case "invoice.payment_failed":
		h.handlePaymentFailed(obj)
	case "customer.subscription.deleted":
		h.handleSubscriptionDeleted(obj)
	}
}

func (h *StripeHandler) handleCheckoutCompleted(obj map[string]interface{}) {
	metadata, _ := obj["metadata"].(map[string]interface{})
	if metadata == nil {
		return
	}
	userIDStr, _ := metadata["user_id"].(string)
	planIDStr, _ := metadata["plan_id"].(string)
	subscriptionID, _ := obj["subscription"].(string)
	customerID, _ := obj["customer"].(string)

	if userIDStr == "" || planIDStr == "" {
		return
	}

	expiresAt := time.Now().AddDate(0, 1, 0) // 1 month default
	_, err := h.pool.Exec(context.Background(),
		`UPDATE users SET plan_id = $1::uuid, stripe_subscription_id = $2, stripe_customer_id = $3, plan_expires_at = $4
		 WHERE id = $5::uuid`,
		planIDStr, subscriptionID, customerID, expiresAt, userIDStr,
	)
	if err != nil {
		log.Printf("Stripe webhook: failed to update user plan: %v", err)
	} else {
		log.Printf("✅ Stripe: activated plan %s for user %s", planIDStr, userIDStr)
	}
}

func (h *StripeHandler) handleInvoicePaid(obj map[string]interface{}) {
	subscriptionID, _ := obj["subscription"].(string)
	if subscriptionID == "" {
		return
	}

	// Extend plan by 1 month
	_, err := h.pool.Exec(context.Background(),
		`UPDATE users SET plan_expires_at = COALESCE(plan_expires_at, NOW()) + INTERVAL '1 month'
		 WHERE stripe_subscription_id = $1`, subscriptionID,
	)
	if err != nil {
		log.Printf("Stripe webhook: failed to extend plan: %v", err)
	}
}

func (h *StripeHandler) handlePaymentFailed(obj map[string]interface{}) {
	subscriptionID, _ := obj["subscription"].(string)
	if subscriptionID == "" {
		return
	}

	// Count failed attempts
	var failCount int
	h.pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM invoices WHERE stripe_subscription_id = $1 AND status = 'failed'`,
		subscriptionID,
	).Scan(&failCount)

	if failCount >= 3 {
		h.pool.Exec(context.Background(),
			`UPDATE users SET status = 'suspended' WHERE stripe_subscription_id = $1`, subscriptionID)
		log.Printf("⚠ Stripe: suspended user after 3 payment failures (sub: %s)", subscriptionID)
	}
}

func (h *StripeHandler) handleSubscriptionDeleted(obj map[string]interface{}) {
	subscriptionID, _ := obj["id"].(string)
	if subscriptionID == "" {
		return
	}

	// Reset to community plan
	_, err := h.pool.Exec(context.Background(),
		`UPDATE users SET plan_id = (SELECT id FROM billing_plans WHERE plan_type = 'community' LIMIT 1),
		 stripe_subscription_id = NULL, plan_expires_at = NULL
		 WHERE stripe_subscription_id = $1`, subscriptionID,
	)
	if err != nil {
		log.Printf("Stripe webhook: failed to reset plan: %v", err)
	} else {
		log.Printf("✅ Stripe: reset user to community plan (sub cancelled: %s)", subscriptionID)
	}
}

func (h *StripeHandler) stripePost(path string, params map[string]string) (map[string]interface{}, error) {
	formData := strings.NewReader(encodeParams(params))
	req, err := http.NewRequest("POST", h.stripeBaseURL+path, formData)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+h.cfg.StripeSecretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse Stripe response: %w", err)
	}
	if resp.StatusCode >= 400 {
		errObj, _ := result["error"].(map[string]interface{})
		msg, _ := errObj["message"].(string)
		return nil, fmt.Errorf("stripe error: %s", msg)
	}
	return result, nil
}

func (h *StripeHandler) validateStripeSignature(payload []byte, sigHeader, secret string) bool {
	// Parse Stripe-Signature header: t=timestamp,v1=signature,...
	var timestamp string
	var signatures []string
	for _, part := range strings.Split(sigHeader, ",") {
		if strings.HasPrefix(part, "t=") {
			timestamp = strings.TrimPrefix(part, "t=")
		} else if strings.HasPrefix(part, "v1=") {
			signatures = append(signatures, strings.TrimPrefix(part, "v1="))
		}
	}
	if timestamp == "" || len(signatures) == 0 {
		return false
	}

	// Validate timestamp within 5 minutes
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		return false
	}

	// Compute expected signature
	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	for _, sig := range signatures {
		if hmac.Equal([]byte(sig), []byte(expected)) {
			return true
		}
	}
	return false
}

func encodeParams(params map[string]string) string {
	var parts []string
	for k, v := range params {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, "&")
}
