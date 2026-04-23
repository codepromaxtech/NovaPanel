package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/config"
)

const licenseServerURL = "https://license.codepromax.com.de"

// LicenseStatus is the cached runtime state of this installation's license.
type LicenseStatus struct {
	Valid     bool            `json:"valid"`
	PlanType  string          `json:"plan_type"`  // "community" | "trial" | "enterprise" | "reseller"
	ExpiresAt string          `json:"expires_at"` // empty = lifetime
	DaysLeft  int             `json:"days_left"`  // -1 = lifetime/community
	Features  LicenseFeatures `json:"features"`
	CheckedAt time.Time       `json:"checked_at"`
	Message   string          `json:"message,omitempty"`
	IsTrial   bool            `json:"is_trial"`
}

// LicenseFeatures are the feature flags and resource quotas for this installation.
type LicenseFeatures struct {
	AllowWAF         bool `json:"allow_waf"`
	AllowFirewall    bool `json:"allow_firewall"`
	AllowCloudflare  bool `json:"allow_cloudflare"`
	AllowTeam        bool `json:"allow_team"`
	AllowAPIKeys     bool `json:"allow_api_keys"`
	AllowK8s         bool `json:"allow_k8s"`
	AllowDocker      bool `json:"allow_docker"`
	AllowWildcardSSL bool `json:"allow_wildcard_ssl"`
	AllowFTP         bool `json:"allow_ftp"`
	AllowReseller    bool `json:"allow_reseller"`
	AllowMultiDeploy bool `json:"allow_multi_deploy"`
	MaxServers       int  `json:"max_servers"`
	MaxDomains       int  `json:"max_domains"`
	MaxDatabases     int  `json:"max_databases"`
	MaxEmail         int  `json:"max_email"`
}

var communityFeatures = LicenseFeatures{
	MaxServers: 1, MaxDomains: 3, MaxDatabases: 2, MaxEmail: 10,
}

var enterpriseFeatures = LicenseFeatures{
	AllowWAF: true, AllowFirewall: true, AllowCloudflare: true, AllowTeam: true,
	AllowAPIKeys: true, AllowK8s: true, AllowDocker: true, AllowWildcardSSL: true,
	AllowFTP: true, AllowMultiDeploy: true,
	MaxServers: 999, MaxDomains: 9999, MaxDatabases: 9999, MaxEmail: 9999,
}

var resellerFeatures = LicenseFeatures{
	AllowWAF: true, AllowFirewall: true, AllowCloudflare: true, AllowTeam: true,
	AllowAPIKeys: true, AllowK8s: true, AllowDocker: true, AllowWildcardSSL: true,
	AllowFTP: true, AllowReseller: true, AllowMultiDeploy: true,
	MaxServers: 999, MaxDomains: 9999, MaxDatabases: 9999, MaxEmail: 9999,
}

type LicenseService struct {
	pool   *pgxpool.Pool
	cfg    *config.Config
	status LicenseStatus
}

func NewLicenseService(pool *pgxpool.Pool, cfg *config.Config) *LicenseService {
	return &LicenseService{pool: pool, cfg: cfg}
}

// GetStatus returns the in-memory cached license status.
func (s *LicenseService) GetStatus() LicenseStatus {
	return s.status
}

// SaveLicenseKey stores a user-provided license key in system_settings and immediately re-verifies.
func (s *LicenseService) SaveLicenseKey(ctx context.Context, key string) error {
	key = strings.TrimSpace(key)
	s.upsert(ctx, "license_paid_key", key)
	return s.Verify(ctx)
}

// Verify resolves which key to use (env → paid-key in DB → trial key in DB → request trial),
// verifies against the license server, updates in-memory status and persists to system_settings.
func (s *LicenseService) Verify(ctx context.Context) error {
	key, isTrial := s.resolveKey(ctx)

	if key == "" {
		// No key at all and trial request failed — community.
		s.setCommunity(ctx, "No license key found — running Community Edition")
		return nil
	}

	domain := s.resolveDomain(ctx)

	payload := map[string]interface{}{
		"license_key": key,
		"domain":      domain,
		"product_id":  s.cfg.LicenseProductID,
		"server_info": map[string]string{"product": "novapanel"},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, licenseServerURL, bytes.NewReader(body))
	if err != nil {
		return s.keepLastOrCommunity(ctx, fmt.Sprintf("license server unreachable: %v", err))
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return s.keepLastOrCommunity(ctx, fmt.Sprintf("license server unreachable: %v", err))
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Valid      bool                   `json:"valid"`
			Trial      bool                   `json:"trial"`
			ExpiresAt  string                 `json:"expires_at"`
			DaysLeft   int                    `json:"days_remaining"`
			Features   map[string]interface{} `json:"features"`
		} `json:"data"`
		Error string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return s.keepLastOrCommunity(ctx, "failed to parse license response")
	}

	if !result.Success || !result.Data.Valid {
		msg := result.Error
		if msg == "" {
			msg = "license invalid or expired"
		}
		// License is definitively invalid/expired — downgrade to community immediately.
		s.setCommunity(ctx, msg)
		log.Printf("⚠️  License invalid/expired: %s — switched to Community Edition", msg)
		return nil
	}

	features, planType := s.mapFeatures(result.Data.Features, isTrial || result.Data.Trial)

	daysLeft := result.Data.DaysLeft
	if result.Data.ExpiresAt == "" || result.Data.ExpiresAt == "0000-00-00 00:00:00" {
		daysLeft = -1
	}

	s.status = LicenseStatus{
		Valid:     true,
		PlanType:  planType,
		ExpiresAt: result.Data.ExpiresAt,
		DaysLeft:  daysLeft,
		Features:  features,
		CheckedAt: time.Now(),
		IsTrial:   isTrial || result.Data.Trial,
	}
	s.persist(ctx)
	log.Printf("✅ License verified: %s plan (domain=%s, expires=%s)", planType, domain, result.Data.ExpiresAt)
	return nil
}

// RunBackgroundChecker loads persisted state, then verifies on startup and every 6 hours.
func (s *LicenseService) RunBackgroundChecker(ctx context.Context) {
	s.loadPersisted(ctx)

	go func() {
		time.Sleep(5 * time.Second)
		if err := s.Verify(ctx); err != nil {
			log.Printf("License check error: %v", err)
		}

		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.Verify(ctx); err != nil {
					log.Printf("License check error: %v", err)
				}
			}
		}
	}()
}

// resolveKey returns the best available license key and whether it's a trial.
// Priority: env var → paid key stored in DB → trial key stored in DB → request new trial.
func (s *LicenseService) resolveKey(ctx context.Context) (string, bool) {
	// 1. Env var takes highest priority (set by operator).
	if s.cfg.LicenseKey != "" {
		return s.cfg.LicenseKey, false
	}

	// 2. Paid key saved via Settings UI.
	var paidKey string
	s.pool.QueryRow(ctx, `SELECT value FROM system_settings WHERE key = 'license_paid_key'`).Scan(&paidKey)
	if strings.TrimSpace(paidKey) != "" {
		return strings.TrimSpace(paidKey), false
	}

	// 3. Trial key already obtained.
	var trialKey string
	s.pool.QueryRow(ctx, `SELECT value FROM system_settings WHERE key = 'license_trial_key'`).Scan(&trialKey)
	if strings.TrimSpace(trialKey) != "" {
		return strings.TrimSpace(trialKey), true
	}

	// 4. No key at all — request a 15-day trial automatically.
	trialKey = s.requestTrial(ctx)
	if trialKey != "" {
		return trialKey, true
	}

	return "", false
}

// requestTrial sends a TRIAL_REQUEST to the license server and stores the returned key.
func (s *LicenseService) requestTrial(ctx context.Context) string {
	domain := s.resolveDomain(ctx)
	log.Printf("🆓 Requesting 15-day trial license for domain: %s", domain)

	payload := map[string]interface{}{
		"license_key": "TRIAL_REQUEST",
		"domain":      domain,
		"product_id":  s.cfg.LicenseProductID,
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, licenseServerURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("Trial request failed: %v", err)
		return ""
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Trial request failed: %v", err)
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Trial      bool   `json:"trial"`
			LicenseKey string `json:"license_key"`
			ExpiresAt  string `json:"expires_at"`
			DaysLeft   int    `json:"days_remaining"`
			Message    string `json:"message"`
		} `json:"data"`
		Error string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || !result.Success {
		log.Printf("Trial request failed: %s", result.Error)
		return ""
	}

	key := result.Data.LicenseKey
	if key == "" {
		return ""
	}

	// Persist trial key so we don't request again on next restart.
	s.upsert(ctx, "license_trial_key", key)
	s.upsert(ctx, "license_trial_expires_at", result.Data.ExpiresAt)
	log.Printf("✅ Trial activated: %s (expires %s)", key, result.Data.ExpiresAt)
	return key
}

// mapFeatures converts license server features JSON to NovaPanel feature flags.
func (s *LicenseService) mapFeatures(raw map[string]interface{}, isTrial bool) (LicenseFeatures, string) {
	planType := "enterprise"
	if raw != nil {
		if pt, ok := raw["plan_type"].(string); ok && pt != "" {
			planType = pt
		}
	}
	if isTrial {
		planType = "trial"
	}

	switch planType {
	case "reseller":
		return resellerFeatures, "reseller"
	case "trial":
		return enterpriseFeatures, "trial" // trial = full enterprise features, time-limited
	case "enterprise":
		return enterpriseFeatures, "enterprise"
	default:
		return communityFeatures, "community"
	}
}

// setCommunity sets community status and persists it.
func (s *LicenseService) setCommunity(ctx context.Context, msg string) {
	s.status = LicenseStatus{
		Valid:     true,
		PlanType:  "community",
		DaysLeft:  -1,
		Features:  communityFeatures,
		CheckedAt: time.Now(),
		Message:   msg,
	}
	s.persist(ctx)
}

// keepLastOrCommunity is used when the license server is unreachable.
// If we have a previous valid status we keep it; otherwise fall back to community.
// This avoids punishing users for temporary network issues.
func (s *LicenseService) keepLastOrCommunity(ctx context.Context, reason string) error {
	log.Printf("⚠️  License server unreachable (%s) — keeping last known state", reason)
	if s.status.PlanType == "" {
		s.setCommunity(ctx, reason)
	}
	return nil
}

// resolveDomain returns the panel's public domain from system_settings.
func (s *LicenseService) resolveDomain(ctx context.Context) string {
	var domain string
	s.pool.QueryRow(ctx, `SELECT value FROM system_settings WHERE key = 'panel_domain'`).Scan(&domain)
	if domain == "" {
		return "localhost"
	}
	return strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(domain)), "https://"), "http://")
}

// persist writes current status to system_settings so it survives restarts.
func (s *LicenseService) persist(ctx context.Context) {
	featJSON, _ := json.Marshal(s.status.Features)
	s.upsert(ctx, "license_valid", fmt.Sprintf("%v", s.status.Valid))
	s.upsert(ctx, "license_plan_type", s.status.PlanType)
	s.upsert(ctx, "license_expires_at", s.status.ExpiresAt)
	s.upsert(ctx, "license_days_left", fmt.Sprintf("%d", s.status.DaysLeft))
	s.upsert(ctx, "license_features", string(featJSON))
	s.upsert(ctx, "license_message", s.status.Message)
	s.upsert(ctx, "license_is_trial", fmt.Sprintf("%v", s.status.IsTrial))
	s.upsert(ctx, "license_checked_at", s.status.CheckedAt.Format(time.RFC3339))
}

// loadPersisted restores the last known status from system_settings on startup.
func (s *LicenseService) loadPersisted(ctx context.Context) {
	rows, err := s.pool.Query(ctx, `SELECT key, value FROM system_settings WHERE key LIKE 'license_%'`)
	if err != nil {
		return
	}
	defer rows.Close()

	kv := map[string]string{}
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		kv[k] = v
	}

	if kv["license_plan_type"] == "" {
		return
	}

	var features LicenseFeatures
	json.Unmarshal([]byte(kv["license_features"]), &features)

	daysLeft := -1
	fmt.Sscanf(kv["license_days_left"], "%d", &daysLeft)

	checkedAt, _ := time.Parse(time.RFC3339, kv["license_checked_at"])

	s.status = LicenseStatus{
		Valid:     kv["license_valid"] == "true",
		PlanType:  kv["license_plan_type"],
		ExpiresAt: kv["license_expires_at"],
		DaysLeft:  daysLeft,
		Features:  features,
		CheckedAt: checkedAt,
		Message:   kv["license_message"],
		IsTrial:   kv["license_is_trial"] == "true",
	}
}

func (s *LicenseService) upsert(ctx context.Context, key, value string) {
	s.pool.Exec(ctx, `
		INSERT INTO system_settings (key, value, encrypted) VALUES ($1, $2, false)
		ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()`,
		key, value)
}
