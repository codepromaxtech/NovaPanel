package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CloudflareService handles Cloudflare API v4 integration
type CloudflareService struct {
	pool   *pgxpool.Pool
	client *http.Client
}

const cfBaseURL = "https://api.cloudflare.com/client/v4"

func NewCloudflareService(pool *pgxpool.Pool) *CloudflareService {
	return &CloudflareService{
		pool:   pool,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// cfRequest makes an authenticated request to Cloudflare API
func (s *CloudflareService) cfRequest(ctx context.Context, method, path string, apiKey, email string, body interface{}) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	}

	url := cfBaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	// Support both API Token and API Key + Email
	if email != "" {
		req.Header.Set("X-Auth-Key", apiKey)
		req.Header.Set("X-Auth-Email", email)
	} else {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloudflare API error: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// ──── Account & Zones ────

// VerifyToken verifies the Cloudflare API token/key is valid
func (s *CloudflareService) VerifyToken(ctx context.Context, apiKey, email string) (map[string]interface{}, error) {
	if email != "" {
		return s.cfRequest(ctx, "GET", "/user", apiKey, email, nil)
	}
	return s.cfRequest(ctx, "GET", "/user/tokens/verify", apiKey, "", nil)
}

// ListZones lists all zones (domains) in the account
func (s *CloudflareService) ListZones(ctx context.Context, apiKey, email string, page int) (map[string]interface{}, error) {
	if page <= 0 {
		page = 1
	}
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/zones?page=%d&per_page=50&order=name", page), apiKey, email, nil)
}

// GetZone gets zone details
func (s *CloudflareService) GetZone(ctx context.Context, apiKey, email, zoneID string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/zones/%s", zoneID), apiKey, email, nil)
}

// ──── DNS Records ────

// ListDNSRecords lists DNS records for a zone
func (s *CloudflareService) ListDNSRecords(ctx context.Context, apiKey, email, zoneID string, page int) (map[string]interface{}, error) {
	if page <= 0 {
		page = 1
	}
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/zones/%s/dns_records?page=%d&per_page=100", zoneID, page), apiKey, email, nil)
}

// CreateDNSRecord creates a new DNS record
func (s *CloudflareService) CreateDNSRecord(ctx context.Context, apiKey, email, zoneID string, record map[string]interface{}) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "POST", fmt.Sprintf("/zones/%s/dns_records", zoneID), apiKey, email, record)
}

// UpdateDNSRecord updates a DNS record
func (s *CloudflareService) UpdateDNSRecord(ctx context.Context, apiKey, email, zoneID, recordID string, record map[string]interface{}) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "PUT", fmt.Sprintf("/zones/%s/dns_records/%s", zoneID, recordID), apiKey, email, record)
}

// DeleteDNSRecord deletes a DNS record
func (s *CloudflareService) DeleteDNSRecord(ctx context.Context, apiKey, email, zoneID, recordID string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "DELETE", fmt.Sprintf("/zones/%s/dns_records/%s", zoneID, recordID), apiKey, email, nil)
}

// ──── SSL/TLS ────

// GetSSLSetting gets current SSL/TLS mode
func (s *CloudflareService) GetSSLSetting(ctx context.Context, apiKey, email, zoneID string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/zones/%s/settings/ssl", zoneID), apiKey, email, nil)
}

// SetSSLSetting sets SSL/TLS mode (off, flexible, full, strict)
func (s *CloudflareService) SetSSLSetting(ctx context.Context, apiKey, email, zoneID, mode string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "PATCH", fmt.Sprintf("/zones/%s/settings/ssl", zoneID), apiKey, email, map[string]string{"value": mode})
}

// GetSSLVerification gets SSL certificate verification status
func (s *CloudflareService) GetSSLVerification(ctx context.Context, apiKey, email, zoneID string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/zones/%s/ssl/verification", zoneID), apiKey, email, nil)
}

// ──── CDN / Cache ────

// PurgeAllCache purges entire cache for a zone
func (s *CloudflareService) PurgeAllCache(ctx context.Context, apiKey, email, zoneID string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "POST", fmt.Sprintf("/zones/%s/purge_cache", zoneID), apiKey, email, map[string]bool{"purge_everything": true})
}

// PurgeURLs purges specific URLs from cache
func (s *CloudflareService) PurgeURLs(ctx context.Context, apiKey, email, zoneID string, urls []string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "POST", fmt.Sprintf("/zones/%s/purge_cache", zoneID), apiKey, email, map[string][]string{"files": urls})
}

// GetCacheSetting gets browser cache TTL
func (s *CloudflareService) GetCacheSetting(ctx context.Context, apiKey, email, zoneID string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/zones/%s/settings/browser_cache_ttl", zoneID), apiKey, email, nil)
}

// SetCacheTTL sets browser cache TTL
func (s *CloudflareService) SetCacheTTL(ctx context.Context, apiKey, email, zoneID string, ttl int) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "PATCH", fmt.Sprintf("/zones/%s/settings/browser_cache_ttl", zoneID), apiKey, email, map[string]int{"value": ttl})
}

// GetDevMode gets development mode status
func (s *CloudflareService) GetDevMode(ctx context.Context, apiKey, email, zoneID string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/zones/%s/settings/development_mode", zoneID), apiKey, email, nil)
}

// SetDevMode enables/disables development mode
func (s *CloudflareService) SetDevMode(ctx context.Context, apiKey, email, zoneID, value string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "PATCH", fmt.Sprintf("/zones/%s/settings/development_mode", zoneID), apiKey, email, map[string]string{"value": value})
}

// ──── Security ────

// GetSecurityLevel gets security level
func (s *CloudflareService) GetSecurityLevel(ctx context.Context, apiKey, email, zoneID string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/zones/%s/settings/security_level", zoneID), apiKey, email, nil)
}

// SetSecurityLevel sets security level (essentially_off, low, medium, high, under_attack)
func (s *CloudflareService) SetSecurityLevel(ctx context.Context, apiKey, email, zoneID, level string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "PATCH", fmt.Sprintf("/zones/%s/settings/security_level", zoneID), apiKey, email, map[string]string{"value": level})
}

// ListFirewallRules lists firewall rules
func (s *CloudflareService) ListFirewallRules(ctx context.Context, apiKey, email, zoneID string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/zones/%s/firewall/rules", zoneID), apiKey, email, nil)
}

// CreateFirewallRule creates a firewall rule
func (s *CloudflareService) CreateFirewallRule(ctx context.Context, apiKey, email, zoneID string, rules []map[string]interface{}) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "POST", fmt.Sprintf("/zones/%s/firewall/rules", zoneID), apiKey, email, rules)
}

// DeleteFirewallRule deletes a firewall rule
func (s *CloudflareService) DeleteFirewallRule(ctx context.Context, apiKey, email, zoneID, ruleID string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "DELETE", fmt.Sprintf("/zones/%s/firewall/rules/%s", zoneID, ruleID), apiKey, email, nil)
}

// ──── Page Rules ────

// ListPageRules lists page rules
func (s *CloudflareService) ListPageRules(ctx context.Context, apiKey, email, zoneID string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/zones/%s/pagerules", zoneID), apiKey, email, nil)
}

// ──── Analytics ────

// GetAnalytics gets zone analytics
func (s *CloudflareService) GetAnalytics(ctx context.Context, apiKey, email, zoneID string, since int) (map[string]interface{}, error) {
	if since <= 0 {
		since = -1440 // Last 24 hours
	}
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/zones/%s/analytics/dashboard?since=%d", zoneID, since), apiKey, email, nil)
}

// ──── Settings ────

// GetMinify gets minification settings
func (s *CloudflareService) GetMinify(ctx context.Context, apiKey, email, zoneID string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/zones/%s/settings/minify", zoneID), apiKey, email, nil)
}

// SetMinify sets minification settings (js, css, html)
func (s *CloudflareService) SetMinify(ctx context.Context, apiKey, email, zoneID string, settings map[string]string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "PATCH", fmt.Sprintf("/zones/%s/settings/minify", zoneID), apiKey, email, map[string]interface{}{"value": settings})
}

// GetAlwaysHTTPS gets always use HTTPS setting
func (s *CloudflareService) GetAlwaysHTTPS(ctx context.Context, apiKey, email, zoneID string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/zones/%s/settings/always_use_https", zoneID), apiKey, email, nil)
}

// SetAlwaysHTTPS sets always use HTTPS
func (s *CloudflareService) SetAlwaysHTTPS(ctx context.Context, apiKey, email, zoneID, value string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "PATCH", fmt.Sprintf("/zones/%s/settings/always_use_https", zoneID), apiKey, email, map[string]string{"value": value})
}

// GetRocketLoader gets Rocket Loader setting
func (s *CloudflareService) GetRocketLoader(ctx context.Context, apiKey, email, zoneID string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/zones/%s/settings/rocket_loader", zoneID), apiKey, email, nil)
}

// SetRocketLoader sets Rocket Loader
func (s *CloudflareService) SetRocketLoader(ctx context.Context, apiKey, email, zoneID, value string) (map[string]interface{}, error) {
	return s.cfRequest(ctx, "PATCH", fmt.Sprintf("/zones/%s/settings/rocket_loader", zoneID), apiKey, email, map[string]string{"value": value})
}
