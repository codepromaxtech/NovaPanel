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

// LicenseStatus is the cached state of this installation's license.
type LicenseStatus struct {
	Valid     bool      `json:"valid"`
	PlanType  string    `json:"plan_type"`  // "community", "trial", "enterprise", "reseller"
	ExpiresAt string    `json:"expires_at"` // ISO string, empty = lifetime
	DaysLeft  int       `json:"days_left"`  // -1 = lifetime
	Features  LicenseFeatures `json:"features"`
	CheckedAt time.Time `json:"checked_at"`
	Message   string    `json:"message,omitempty"`
}

// LicenseFeatures mirrors the feature-flag JSON returned by the license server.
// The license server stores these under the `features` column per license.
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

// communityFeatures are the default limits when no license is present.
var communityFeatures = LicenseFeatures{
	MaxServers: 1, MaxDomains: 3, MaxDatabases: 2, MaxEmail: 10,
}

// enterpriseFeatures are the full feature set for a paid license.
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

// GetStatus returns the last-known license status (in-memory cache).
func (s *LicenseService) GetStatus() LicenseStatus {
	return s.status
}

// Verify contacts the license server, stores the result in system_settings, and updates the in-memory cache.
func (s *LicenseService) Verify(ctx context.Context) error {
	licenseKey := s.cfg.LicenseKey

	// No key configured → community edition.
	if licenseKey == "" {
		s.status = LicenseStatus{
			Valid: true, PlanType: "community", DaysLeft: -1,
			Features: communityFeatures, CheckedAt: time.Now(),
			Message: "No license key configured — running Community Edition",
		}
		s.persist(ctx)
		return nil
	}

	domain := s.resolveDomain(ctx)

	payload := map[string]interface{}{
		"license_key": licenseKey,
		"domain":      domain,
		"product_id":  s.cfg.LicenseProductID,
		"server_info": map[string]string{
			"product": "novapanel",
		},
	}

	body, _ := json.Marshal(payload)
	const licenseServerURL = "https://license.codepromax.com.de"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, licenseServerURL, bytes.NewReader(body))
	if err != nil {
		return s.fallback(ctx, fmt.Sprintf("license server unreachable: %v", err))
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return s.fallback(ctx, fmt.Sprintf("license server unreachable: %v", err))
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Valid      bool            `json:"valid"`
			Trial      bool            `json:"trial"`
			ExpiresAt  string          `json:"expires_at"`
			DaysLeft   int             `json:"days_remaining"`
			Features   map[string]interface{} `json:"features"`
		} `json:"data"`
		Error string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return s.fallback(ctx, "failed to parse license response")
	}

	if !result.Success || !result.Data.Valid {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = "license invalid or expired"
		}
		s.status = LicenseStatus{
			Valid: false, PlanType: "community", DaysLeft: -1,
			Features: communityFeatures, CheckedAt: time.Now(),
			Message: errMsg,
		}
		s.persist(ctx)
		log.Printf("⚠️  License invalid: %s — falling back to Community Edition", errMsg)
		return nil
	}

	// Map license features to NovaPanel feature flags.
	features, planType := s.mapFeatures(result.Data.Features, result.Data.Trial)

	daysLeft := result.Data.DaysLeft
	if result.Data.ExpiresAt == "" || result.Data.ExpiresAt == "0000-00-00 00:00:00" {
		daysLeft = -1 // lifetime
	}

	s.status = LicenseStatus{
		Valid:     true,
		PlanType:  planType,
		ExpiresAt: result.Data.ExpiresAt,
		DaysLeft:  daysLeft,
		Features:  features,
		CheckedAt: time.Now(),
	}
	s.persist(ctx)
	log.Printf("✅ License verified: %s plan, domain=%s", planType, domain)
	return nil
}

// RunBackgroundChecker verifies on startup (after 5s) then every 6 hours.
func (s *LicenseService) RunBackgroundChecker(ctx context.Context) {
	// Load last persisted status first so we're not blind until first check.
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

// mapFeatures converts the license server's generic features JSON to NovaPanel feature flags.
// The license server should store features as: {"plan_type":"enterprise","allow_waf":true,...}
// If plan_type is present, we use our predefined flag sets; otherwise we fall back to parsing booleans.
func (s *LicenseService) mapFeatures(raw map[string]interface{}, isTrial bool) (LicenseFeatures, string) {
	if raw == nil {
		if isTrial {
			return communityFeatures, "trial"
		}
		return communityFeatures, "community"
	}

	// Check if the license explicitly specifies a plan type.
	planType := "enterprise"
	if pt, ok := raw["plan_type"].(string); ok && pt != "" {
		planType = pt
	} else if isTrial {
		planType = "trial"
	}

	switch planType {
	case "reseller":
		return resellerFeatures, "reseller"
	case "enterprise":
		return enterpriseFeatures, "enterprise"
	case "trial":
		// Trial gets enterprise features but time-limited.
		return enterpriseFeatures, "trial"
	default:
		return communityFeatures, "community"
	}
}

// resolveDomain returns the domain this panel is running on (stored in system_settings or hostname).
func (s *LicenseService) resolveDomain(ctx context.Context) string {
	var domain string
	s.pool.QueryRow(ctx, `SELECT value FROM system_settings WHERE key = 'panel_domain'`).Scan(&domain)
	if domain == "" {
		return "localhost"
	}
	return strings.TrimPrefix(strings.TrimPrefix(domain, "https://"), "http://")
}

// persist writes the current status to system_settings so it survives restarts.
func (s *LicenseService) persist(ctx context.Context) {
	featJSON, _ := json.Marshal(s.status.Features)
	upsert := func(key, val string) {
		s.pool.Exec(ctx, `
			INSERT INTO system_settings (key, value, encrypted) VALUES ($1, $2, false)
			ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()`,
			key, val)
	}
	upsert("license_valid", fmt.Sprintf("%v", s.status.Valid))
	upsert("license_plan_type", s.status.PlanType)
	upsert("license_expires_at", s.status.ExpiresAt)
	upsert("license_days_left", fmt.Sprintf("%d", s.status.DaysLeft))
	upsert("license_features", string(featJSON))
	upsert("license_message", s.status.Message)
	upsert("license_checked_at", s.status.CheckedAt.Format(time.RFC3339))
}

// loadPersisted restores the last known status from system_settings.
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
	}
}

// fallback is used when the license server is unreachable — keeps last known state
// but limits to community if we've never had a valid license.
func (s *LicenseService) fallback(ctx context.Context, reason string) error {
	log.Printf("⚠️  License check failed (%s) — using last known state", reason)
	if s.status.PlanType == "" {
		s.status = LicenseStatus{
			Valid: true, PlanType: "community", DaysLeft: -1,
			Features: communityFeatures, CheckedAt: time.Now(),
			Message: reason,
		}
	}
	return nil
}
