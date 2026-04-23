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
	novacrypto "github.com/novapanel/novapanel/internal/crypto"
	"github.com/novapanel/novapanel/internal/provisioner"
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

// ──── Cloudflare Tunnels ────

func (s *CloudflareService) getAccountID(ctx context.Context, apiKey, email string) (string, error) {
	r, err := s.cfRequest(ctx, "GET", "/accounts?page=1&per_page=5", apiKey, email, nil)
	if err != nil {
		return "", err
	}
	if results, ok := r["result"].([]interface{}); ok && len(results) > 0 {
		if acc, ok := results[0].(map[string]interface{}); ok {
			if id, ok := acc["id"].(string); ok {
				return id, nil
			}
		}
	}
	return "", fmt.Errorf("no account found")
}

func (s *CloudflareService) getServer(ctx context.Context, serverID string) (provisioner.ServerInfo, error) {
	var server provisioner.ServerInfo
	var port int
	var encKey, encPassword string
	err := s.pool.QueryRow(ctx,
		`SELECT host(ip_address), port, ssh_user, COALESCE(ssh_key, ''), COALESCE(ssh_password, ''), COALESCE(auth_method, 'password'), COALESCE(is_local, FALSE)
		 FROM servers WHERE id = $1`, serverID,
	).Scan(&server.IPAddress, &port, &server.SSHUser, &encKey, &encPassword, &server.AuthMethod, &server.IsLocal)
	if err != nil {
		return server, err
	}
	server.Port = port
	if cryptoKey, kerr := novacrypto.GetEncryptionKey(); kerr == nil {
		if encKey != "" {
			if dec, derr := novacrypto.Decrypt(encKey, cryptoKey); derr == nil {
				encKey = dec
			}
		}
		if encPassword != "" {
			if dec, derr := novacrypto.Decrypt(encPassword, cryptoKey); derr == nil {
				encPassword = dec
			}
		}
	}
	server.SSHKey = encKey
	server.SSHPassword = encPassword
	return server, nil
}

// ListTunnels lists all CF tunnels in the account
func (s *CloudflareService) ListTunnels(ctx context.Context, apiKey, email string) (map[string]interface{}, error) {
	accID, err := s.getAccountID(ctx, apiKey, email)
	if err != nil {
		return nil, err
	}
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/accounts/%s/cfd_tunnel?is_deleted=false", accID), apiKey, email, nil)
}

// CreateTunnel creates a new Cloudflare Tunnel
func (s *CloudflareService) CreateTunnel(ctx context.Context, apiKey, email, name, tunnelSecret string) (map[string]interface{}, error) {
	accID, err := s.getAccountID(ctx, apiKey, email)
	if err != nil {
		return nil, err
	}
	body := map[string]interface{}{
		"name":          name,
		"tunnel_secret": tunnelSecret,
	}
	return s.cfRequest(ctx, "POST", fmt.Sprintf("/accounts/%s/cfd_tunnel", accID), apiKey, email, body)
}

// DeleteTunnel deletes a tunnel
func (s *CloudflareService) DeleteTunnel(ctx context.Context, apiKey, email, tunnelID string) (map[string]interface{}, error) {
	accID, err := s.getAccountID(ctx, apiKey, email)
	if err != nil {
		return nil, err
	}
	return s.cfRequest(ctx, "DELETE", fmt.Sprintf("/accounts/%s/cfd_tunnel/%s", accID, tunnelID), apiKey, email, nil)
}

// GetTunnel gets tunnel details
func (s *CloudflareService) GetTunnel(ctx context.Context, apiKey, email, tunnelID string) (map[string]interface{}, error) {
	accID, err := s.getAccountID(ctx, apiKey, email)
	if err != nil {
		return nil, err
	}
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/accounts/%s/cfd_tunnel/%s", accID, tunnelID), apiKey, email, nil)
}

// GetTunnelToken gets the token for running cloudflared with this tunnel
func (s *CloudflareService) GetTunnelToken(ctx context.Context, apiKey, email, tunnelID string) (map[string]interface{}, error) {
	accID, err := s.getAccountID(ctx, apiKey, email)
	if err != nil {
		return nil, err
	}
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/accounts/%s/cfd_tunnel/%s/token", accID, tunnelID), apiKey, email, nil)
}

// ListTunnelConnections lists active connections for a tunnel
func (s *CloudflareService) ListTunnelConnections(ctx context.Context, apiKey, email, tunnelID string) (map[string]interface{}, error) {
	accID, err := s.getAccountID(ctx, apiKey, email)
	if err != nil {
		return nil, err
	}
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/accounts/%s/cfd_tunnel/%s/connections", accID, tunnelID), apiKey, email, nil)
}

// UpdateTunnelConfig updates ingress rules for a tunnel
func (s *CloudflareService) UpdateTunnelConfig(ctx context.Context, apiKey, email, tunnelID string, config map[string]interface{}) (map[string]interface{}, error) {
	accID, err := s.getAccountID(ctx, apiKey, email)
	if err != nil {
		return nil, err
	}
	return s.cfRequest(ctx, "PUT", fmt.Sprintf("/accounts/%s/cfd_tunnel/%s/configurations", accID, tunnelID), apiKey, email, config)
}

// GetTunnelConfig gets the current config for a tunnel
func (s *CloudflareService) GetTunnelConfig(ctx context.Context, apiKey, email, tunnelID string) (map[string]interface{}, error) {
	accID, err := s.getAccountID(ctx, apiKey, email)
	if err != nil {
		return nil, err
	}
	return s.cfRequest(ctx, "GET", fmt.Sprintf("/accounts/%s/cfd_tunnel/%s/configurations", accID, tunnelID), apiKey, email, nil)
}

// CreateTunnelDNSRoute creates a CNAME DNS record pointing to the tunnel
func (s *CloudflareService) CreateTunnelDNSRoute(ctx context.Context, apiKey, email, zoneID, hostname, tunnelID string) (map[string]interface{}, error) {
	record := map[string]interface{}{
		"type":    "CNAME",
		"name":    hostname,
		"content": tunnelID + ".cfargotunnel.com",
		"ttl":     1,
		"proxied": true,
	}
	return s.cfRequest(ctx, "POST", fmt.Sprintf("/zones/%s/dns_records", zoneID), apiKey, email, record)
}

// ──── SSH-based cloudflared management ────

// InstallCloudflared installs cloudflared on a remote server via SSH
func (s *CloudflareService) InstallCloudflared(ctx context.Context, serverID string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := `#!/bin/bash
set -e
if command -v cloudflared &>/dev/null; then
    echo "cloudflared already installed: $(cloudflared --version)"
    exit 0
fi
echo "Installing cloudflared..."
# Detect arch
ARCH=$(uname -m)
case $ARCH in
    x86_64)  CF_ARCH="amd64" ;;
    aarch64) CF_ARCH="arm64" ;;
    armv7l)  CF_ARCH="arm" ;;
    *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac
# Download and install
curl -fsSL "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-${CF_ARCH}.deb" -o /tmp/cloudflared.deb 2>/dev/null && dpkg -i /tmp/cloudflared.deb 2>/dev/null && rm /tmp/cloudflared.deb || {
    curl -fsSL "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-${CF_ARCH}" -o /usr/local/bin/cloudflared
    chmod +x /usr/local/bin/cloudflared
}
echo "Installed: $(cloudflared --version)"
`
	return provisioner.RunScript(server, script)
}

// RunTunnel starts a cloudflared tunnel on a remote server as a systemd service
func (s *CloudflareService) RunTunnel(ctx context.Context, serverID, tunnelToken, tunnelName string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`#!/bin/bash
set -e
# Install cloudflared if needed
if ! command -v cloudflared &>/dev/null; then
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) CF_ARCH="amd64" ;; aarch64) CF_ARCH="arm64" ;; *) CF_ARCH="amd64" ;;
    esac
    curl -fsSL "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-${CF_ARCH}" -o /usr/local/bin/cloudflared
    chmod +x /usr/local/bin/cloudflared
fi

# Create systemd service
cat > /etc/systemd/system/cloudflared-%s.service << 'UNIT'
[Unit]
Description=Cloudflare Tunnel %s
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/cloudflared tunnel --no-autoupdate run --token %s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable cloudflared-%s.service
systemctl restart cloudflared-%s.service
sleep 2
systemctl status cloudflared-%s.service --no-pager 2>&1 || true
echo "Tunnel %s is running"
`, tunnelName, tunnelName, tunnelToken, tunnelName, tunnelName, tunnelName, tunnelName)
	return provisioner.RunScript(server, script)
}

// StopTunnel stops a cloudflared tunnel on a remote server
func (s *CloudflareService) StopTunnel(ctx context.Context, serverID, tunnelName string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`systemctl stop cloudflared-%s.service 2>&1
systemctl disable cloudflared-%s.service 2>&1
rm -f /etc/systemd/system/cloudflared-%s.service
systemctl daemon-reload
echo "Tunnel %s stopped and removed"`, tunnelName, tunnelName, tunnelName, tunnelName)
	return provisioner.RunScript(server, script)
}

// TunnelStatus checks cloudflared tunnel status on a server
func (s *CloudflareService) TunnelStatus(ctx context.Context, serverID, tunnelName string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := fmt.Sprintf(`systemctl status cloudflared-%s.service --no-pager 2>&1 || echo "Tunnel not running"`, tunnelName)
	return provisioner.RunScript(server, script)
}
