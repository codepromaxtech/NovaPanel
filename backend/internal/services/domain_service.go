package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/provisioner"
)

type DomainService struct {
	db *pgxpool.Pool
}

func NewDomainService(db *pgxpool.Pool) *DomainService {
	return &DomainService{db: db}
}

// ─── SSH Helper ───

func (s *DomainService) getServerSSH(ctx context.Context, serverID string) (provisioner.ServerInfo, error) {
	return GetServerInfo(ctx, s.db, serverID)
}

// ─── Vhost Config Generation ───

func generateNginxVhost(domain, docRoot, phpVersion string, sslEnabled, isLoadBalancer bool, targetIPs []string, proxyPass string) string {
	safeDomain := strings.ReplaceAll(domain, ".", "_")
	var b strings.Builder

	// Upstream block for load balancers
	if isLoadBalancer && len(targetIPs) > 0 {
		b.WriteString(fmt.Sprintf("upstream backend_pool_%s {\n    least_conn;\n", safeDomain))
		for _, ip := range targetIPs {
			b.WriteString(fmt.Sprintf("    server %s;\n", ip))
		}
		b.WriteString("}\n\n")
	}

	// HTTP server block
	b.WriteString("server {\n")
	b.WriteString("    listen 80;\n")
	b.WriteString("    listen [::]:80;\n")
	b.WriteString(fmt.Sprintf("    server_name %s www.%s;\n\n", domain, domain))

	if sslEnabled {
		// Redirect HTTP to HTTPS
		b.WriteString("    location /.well-known/acme-challenge/ {\n")
		b.WriteString(fmt.Sprintf("        root %s;\n", docRoot))
		b.WriteString("    }\n\n")
		b.WriteString("    location / {\n")
		b.WriteString("        return 301 https://$host$request_uri;\n")
		b.WriteString("    }\n")
	} else {
		writeNginxLocationBlocks(&b, domain, docRoot, phpVersion, isLoadBalancer, targetIPs, proxyPass)
	}
	b.WriteString("}\n")

	// HTTPS server block
	if sslEnabled {
		b.WriteString("\nserver {\n")
		b.WriteString("    listen 443 ssl http2;\n")
		b.WriteString("    listen [::]:443 ssl http2;\n")
		b.WriteString(fmt.Sprintf("    server_name %s www.%s;\n\n", domain, domain))
		b.WriteString(fmt.Sprintf("    ssl_certificate /etc/letsencrypt/live/%s/fullchain.pem;\n", domain))
		b.WriteString(fmt.Sprintf("    ssl_certificate_key /etc/letsencrypt/live/%s/privkey.pem;\n", domain))
		b.WriteString("    ssl_protocols TLSv1.2 TLSv1.3;\n")
		b.WriteString("    ssl_ciphers HIGH:!aNULL:!MD5;\n")
		b.WriteString("    ssl_prefer_server_ciphers on;\n\n")
		b.WriteString("    add_header Strict-Transport-Security \"max-age=31536000; includeSubDomains\" always;\n")

		writeNginxLocationBlocks(&b, domain, docRoot, phpVersion, isLoadBalancer, targetIPs, proxyPass)
		b.WriteString("}\n")
	}

	return b.String()
}

func writeNginxLocationBlocks(b *strings.Builder, domain, docRoot, phpVersion string, isLoadBalancer bool, targetIPs []string, proxyPass string) {
	safeDomain := strings.ReplaceAll(domain, ".", "_")

	b.WriteString(fmt.Sprintf("\n    root %s;\n", docRoot))
	b.WriteString("    index index.php index.html index.htm;\n\n")
	b.WriteString(fmt.Sprintf("    access_log /var/log/nginx/%s.access.log;\n", domain))
	b.WriteString(fmt.Sprintf("    error_log /var/log/nginx/%s.error.log;\n\n", domain))

	// Security headers
	b.WriteString("    add_header X-Frame-Options \"SAMEORIGIN\" always;\n")
	b.WriteString("    add_header X-Content-Type-Options \"nosniff\" always;\n")
	b.WriteString("    add_header X-XSS-Protection \"1; mode=block\" always;\n")
	b.WriteString("    add_header Referrer-Policy \"strict-origin-when-cross-origin\" always;\n\n")

	if isLoadBalancer && len(targetIPs) > 0 {
		b.WriteString("    location / {\n")
		b.WriteString(fmt.Sprintf("        proxy_pass http://backend_pool_%s;\n", safeDomain))
		b.WriteString("        proxy_http_version 1.1;\n")
		b.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
		b.WriteString("        proxy_set_header Connection 'upgrade';\n")
		b.WriteString("        proxy_set_header Host $host;\n")
		b.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
		b.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
		b.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
		b.WriteString("        proxy_cache_bypass $http_upgrade;\n")
		b.WriteString("    }\n")
	} else if proxyPass != "" {
		b.WriteString("    location / {\n")
		b.WriteString(fmt.Sprintf("        proxy_pass %s;\n", proxyPass))
		b.WriteString("        proxy_http_version 1.1;\n")
		b.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
		b.WriteString("        proxy_set_header Connection 'upgrade';\n")
		b.WriteString("        proxy_set_header Host $host;\n")
		b.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
		b.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
		b.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
		b.WriteString("        proxy_cache_bypass $http_upgrade;\n")
		b.WriteString("    }\n")
	} else {
		b.WriteString("    location / {\n")
		b.WriteString("        try_files $uri $uri/ /index.php?$query_string;\n")
		b.WriteString("    }\n\n")

		if phpVersion != "" {
			b.WriteString(fmt.Sprintf("    location ~ \\.php$ {\n"))
			b.WriteString("        include snippets/fastcgi-php.conf;\n")
			b.WriteString(fmt.Sprintf("        fastcgi_pass unix:/run/php/php%s-fpm.sock;\n", phpVersion))
			b.WriteString("        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;\n")
			b.WriteString("        include fastcgi_params;\n")
			b.WriteString("    }\n\n")
		}

		b.WriteString("    location ~ /\\. {\n")
		b.WriteString("        deny all;\n")
		b.WriteString("    }\n\n")

		b.WriteString("    location ~* \\.(jpg|jpeg|png|gif|ico|css|js|woff2|woff|ttf|svg)$ {\n")
		b.WriteString("        expires 30d;\n")
		b.WriteString("        add_header Cache-Control \"public, immutable\";\n")
		b.WriteString("    }\n")
	}
}

func generateApacheVhost(domain, docRoot, phpVersion string, sslEnabled bool) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("<VirtualHost *:80>\n"))
	b.WriteString(fmt.Sprintf("    ServerName %s\n", domain))
	b.WriteString(fmt.Sprintf("    ServerAlias www.%s\n", domain))
	b.WriteString(fmt.Sprintf("    DocumentRoot %s\n\n", docRoot))

	if sslEnabled {
		b.WriteString("    RewriteEngine On\n")
		b.WriteString("    RewriteCond %%{HTTPS} off\n")
		b.WriteString("    RewriteRule ^ https://%%{HTTP_HOST}%%{REQUEST_URI} [L,R=301]\n")
	} else {
		writeApacheDirectoryBlock(&b, docRoot)
		writeApacheLogsAndHeaders(&b, domain)
	}
	b.WriteString("</VirtualHost>\n")

	if sslEnabled {
		b.WriteString(fmt.Sprintf("\n<VirtualHost *:443>\n"))
		b.WriteString(fmt.Sprintf("    ServerName %s\n", domain))
		b.WriteString(fmt.Sprintf("    ServerAlias www.%s\n", domain))
		b.WriteString(fmt.Sprintf("    DocumentRoot %s\n\n", docRoot))
		b.WriteString("    SSLEngine on\n")
		b.WriteString(fmt.Sprintf("    SSLCertificateFile /etc/letsencrypt/live/%s/fullchain.pem\n", domain))
		b.WriteString(fmt.Sprintf("    SSLCertificateKeyFile /etc/letsencrypt/live/%s/privkey.pem\n\n", domain))
		b.WriteString("    Header always set Strict-Transport-Security \"max-age=31536000; includeSubDomains\"\n")

		writeApacheDirectoryBlock(&b, docRoot)
		writeApacheLogsAndHeaders(&b, domain)
		b.WriteString("</VirtualHost>\n")
	}

	return b.String()
}

func writeApacheDirectoryBlock(b *strings.Builder, docRoot string) {
	b.WriteString(fmt.Sprintf("\n    <Directory %s>\n", docRoot))
	b.WriteString("        Options -Indexes +FollowSymLinks\n")
	b.WriteString("        AllowOverride All\n")
	b.WriteString("        Require all granted\n")
	b.WriteString("    </Directory>\n\n")
}

func writeApacheLogsAndHeaders(b *strings.Builder, domain string) {
	b.WriteString(fmt.Sprintf("    ErrorLog ${APACHE_LOG_DIR}/%s.error.log\n", domain))
	b.WriteString(fmt.Sprintf("    CustomLog ${APACHE_LOG_DIR}/%s.access.log combined\n\n", domain))
	b.WriteString("    Header always set X-Frame-Options \"SAMEORIGIN\"\n")
	b.WriteString("    Header always set X-Content-Type-Options \"nosniff\"\n")
	b.WriteString("    Header always set X-XSS-Protection \"1; mode=block\"\n")
}

// ─── Provisioning Scripts ───

func (s *DomainService) provisionDomain(server provisioner.ServerInfo, domain *models.Domain, targetIPs []string) error {
	var vhostConfig string

	switch domain.WebServer {
	case "apache":
		vhostConfig = generateApacheVhost(domain.Name, domain.DocumentRoot, domain.PHPVersion, domain.SSLEnabled)
	default: // nginx
		vhostConfig = generateNginxVhost(domain.Name, domain.DocumentRoot, domain.PHPVersion, domain.SSLEnabled, domain.IsLoadBalancer, targetIPs, "")
	}

	// Escape the config for embedding in a heredoc
	escapedConfig := strings.ReplaceAll(vhostConfig, "'", "'\"'\"'")

	var script string
	switch domain.WebServer {
	case "apache":
		script = fmt.Sprintf(`
# Create document root
mkdir -p '%s'
chown -R www-data:www-data '%s'

# Write default index page if none exists
if [ ! -f '%s/index.html' ]; then
cat > '%s/index.html' << 'INDEXEOF'
<html><body><h1>Welcome to %s</h1><p>Powered by NovaPanel</p></body></html>
INDEXEOF
fi

# Write Apache vhost config
cat > '/etc/apache2/sites-available/%s.conf' << 'VHOSTEOF'
%s
VHOSTEOF

# Enable site
a2ensite '%s.conf' 2>&1

# Test and reload
apache2ctl configtest 2>&1 && systemctl reload apache2 2>&1
echo "VHOST_PROVISIONED"
`, domain.DocumentRoot, domain.DocumentRoot, domain.DocumentRoot, domain.DocumentRoot,
			domain.Name, domain.Name, escapedConfig, domain.Name)

	default: // nginx
		script = fmt.Sprintf(`
# Create document root
mkdir -p '%s'
chown -R www-data:www-data '%s'

# Write default index page if none exists
if [ ! -f '%s/index.html' ]; then
cat > '%s/index.html' << 'INDEXEOF'
<html><body><h1>Welcome to %s</h1><p>Powered by NovaPanel</p></body></html>
INDEXEOF
fi

# Write Nginx vhost config
cat > '/etc/nginx/sites-available/%s' << 'VHOSTEOF'
%s
VHOSTEOF

# Enable site (symlink)
ln -sf '/etc/nginx/sites-available/%s' '/etc/nginx/sites-enabled/%s'

# Test and reload
nginx -t 2>&1 && systemctl reload nginx 2>&1
echo "VHOST_PROVISIONED"
`, domain.DocumentRoot, domain.DocumentRoot, domain.DocumentRoot, domain.DocumentRoot,
			domain.Name, domain.Name, escapedConfig, domain.Name, domain.Name)
	}

	output, err := provisioner.RunScript(server, script)
	if err != nil {
		return fmt.Errorf("vhost provisioning failed: %w\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "VHOST_PROVISIONED") {
		return fmt.Errorf("vhost provisioning did not complete: %s", output)
	}

	return nil
}

func (s *DomainService) provisionSSL(server provisioner.ServerInfo, domain *models.Domain) error {
	script := fmt.Sprintf(`
# Install certbot if not present
if ! command -v certbot &> /dev/null; then
    apt-get update -qq
    apt-get install -y certbot python3-certbot-nginx 2>&1
fi

# Issue SSL certificate
certbot certonly --non-interactive --agree-tos --register-unsafely-without-email \
    --webroot --webroot-path '%s' \
    -d '%s' -d 'www.%s' 2>&1

if [ $? -ne 0 ]; then
    echo "SSL_FAILED"
    exit 1
fi

# Enable automatic renewal via systemd timer (preferred) or cron fallback
if systemctl list-unit-files certbot.timer &>/dev/null; then
    systemctl enable certbot.timer 2>&1
    systemctl start certbot.timer 2>&1
else
    # Cron fallback: renew twice daily
    CRON_JOB="0 */12 * * * root certbot renew --quiet --deploy-hook 'systemctl reload nginx'"
    grep -qF "certbot renew" /etc/cron.d/certbot 2>/dev/null || \
        echo "$CRON_JOB" > /etc/cron.d/certbot
fi

echo "SSL_ISSUED"
`, domain.DocumentRoot, domain.Name, domain.Name)

	output, err := provisioner.RunScript(server, script)
	if err != nil {
		return fmt.Errorf("SSL provisioning failed: %w\nOutput: %s", err, output)
	}

	if strings.Contains(output, "SSL_FAILED") {
		return fmt.Errorf("certbot failed: %s", output)
	}

	return nil
}

func (s *DomainService) deprovisionDomain(ctx context.Context, domain *models.Domain) error {
	if domain.ServerID == nil {
		return nil // No server to deprovision from
	}

	server, err := s.getServerSSH(ctx, domain.ServerID.String())
	if err != nil {
		return fmt.Errorf("failed to get server SSH info: %w", err)
	}

	var script string
	switch domain.WebServer {
	case "apache":
		script = fmt.Sprintf(`
a2dissite '%s.conf' 2>/dev/null || true
rm -f '/etc/apache2/sites-available/%s.conf'
apache2ctl configtest 2>&1 && systemctl reload apache2 2>&1
echo "VHOST_REMOVED"
`, domain.Name, domain.Name)
	default: // nginx
		script = fmt.Sprintf(`
rm -f '/etc/nginx/sites-enabled/%s'
rm -f '/etc/nginx/sites-available/%s'
nginx -t 2>&1 && systemctl reload nginx 2>&1
echo "VHOST_REMOVED"
`, domain.Name, domain.Name)
	}

	output, err := provisioner.RunScript(server, script)
	if err != nil {
		log.Printf("Warning: vhost removal for %s failed: %v (output: %s)", domain.Name, err, output)
		// Don't block DB deletion on SSH failures
	}

	return nil
}

// ─── CRUD Operations ───

func (s *DomainService) Create(ctx context.Context, userID string, req models.CreateDomainRequest) (*models.Domain, error) {
	// Validate domain name
	req.Name = strings.ToLower(strings.TrimSpace(req.Name))
	if req.Name == "" {
		return nil, fmt.Errorf("domain name is required")
	}

	// Check if domain already exists
	var exists bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM domains WHERE name = $1)", req.Name).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("domain %s already exists", req.Name)
	}

	// Defaults
	domainType := "primary"
	if req.Type != "" {
		domainType = req.Type
	}
	webServer := "nginx"
	if req.WebServer != "" {
		webServer = req.WebServer
	}
	phpVersion := "8.2"
	if req.PHPVersion != "" {
		phpVersion = req.PHPVersion
	}
	docRoot := "/var/www/" + req.Name
	if req.DocumentRoot != "" {
		docRoot = req.DocumentRoot
	}

	// Generate a unique system user for this domain (e.g., "nova_example_com")
	systemUser := "nova_" + strings.ReplaceAll(strings.ReplaceAll(req.Name, ".", "_"), "-", "_")
	if len(systemUser) > 32 {
		systemUser = systemUser[:32]
	}

	// Begin transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var serverID *uuid.UUID
	if !req.IsLoadBalancer && req.ServerID != "" {
		id, err := uuid.Parse(req.ServerID)
		if err != nil {
			return nil, fmt.Errorf("invalid server_id")
		}
		serverID = &id
	}

	uid, _ := uuid.Parse(userID)
	domain := &models.Domain{}
	err = tx.QueryRow(ctx,
		`INSERT INTO domains (user_id, server_id, name, type, document_root, web_server, php_version, status, is_load_balancer, system_user)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id, user_id, server_id, name, type, document_root, web_server, php_version, ssl_enabled, status, system_user, is_load_balancer, created_at, updated_at`,
		uid, serverID, req.Name, domainType, docRoot, webServer, phpVersion, "provisioning", req.IsLoadBalancer, systemUser,
	).Scan(&domain.ID, &domain.UserID, &domain.ServerID, &domain.Name, &domain.Type,
		&domain.DocumentRoot, &domain.WebServer, &domain.PHPVersion, &domain.SSLEnabled,
		&domain.Status, &domain.SystemUser, &domain.IsLoadBalancer, &domain.CreatedAt, &domain.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Insert load balancer backend relationships
	if req.IsLoadBalancer && len(req.BackendServerIDs) > 0 {
		for _, bID := range req.BackendServerIDs {
			backendUUID, err := uuid.Parse(bID)
			if err != nil {
				continue
			}
			_, err = tx.Exec(ctx, "INSERT INTO domain_backend_servers (domain_id, server_id) VALUES ($1, $2)", domain.ID, backendUUID)
			if err == nil {
				domain.BackendServerIDs = append(domain.BackendServerIDs, backendUUID)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// ── Provision on the remote server (async) ──
	if serverID != nil || req.IsLoadBalancer {
		go func() {
			bgCtx := context.Background()

			// Determine which server to provision on
			var sshServerID string
			if serverID != nil {
				sshServerID = serverID.String()
			} else if req.IsLoadBalancer && len(req.BackendServerIDs) > 0 {
				// For LB, provision on the first backend server (or a dedicated LB server)
				sshServerID = req.BackendServerIDs[0]
			}

			if sshServerID == "" {
				s.updateDomainStatus(bgCtx, domain.ID.String(), "error")
				return
			}

			server, err := s.getServerSSH(bgCtx, sshServerID)
			if err != nil {
				log.Printf("❌ Domain %s: failed to get server SSH info: %v", domain.Name, err)
				s.updateDomainStatus(bgCtx, domain.ID.String(), "error")
				return
			}

			// Resolve backend IPs for load balancers
			var targetIPs []string
			if domain.IsLoadBalancer {
				for _, bID := range domain.BackendServerIDs {
					var ip string
					s.db.QueryRow(bgCtx, "SELECT host(ip_address) FROM servers WHERE id = $1", bID).Scan(&ip)
					if ip != "" {
						targetIPs = append(targetIPs, ip)
					}
				}
			}

			// 0. Create isolated system user for this domain
			if domain.SystemUser != "" {
				userScript := fmt.Sprintf(`
id '%s' &>/dev/null || useradd -r -m -d '%s' -s /bin/bash '%s'
usermod -aG www-data '%s' 2>/dev/null || true
echo "SYSTEM_USER_READY"
`, domain.SystemUser, domain.DocumentRoot, domain.SystemUser, domain.SystemUser)
				if output, err := provisioner.RunScript(server, userScript); err != nil {
					log.Printf("⚠️  Domain %s: system user creation failed: %v (output: %s)", domain.Name, err, output)
				} else {
					log.Printf("✅ Domain %s: system user '%s' created", domain.Name, domain.SystemUser)
				}
			}

			// 1. Provision vhost
			if err := s.provisionDomain(server, domain, targetIPs); err != nil {
				log.Printf("❌ Domain %s: vhost provisioning failed: %v", domain.Name, err)
				s.updateDomainStatus(bgCtx, domain.ID.String(), "error")
				return
			}
			log.Printf("✅ Domain %s: vhost provisioned successfully", domain.Name)

			// 2. Provision SSL (attempt, non-blocking on failure)
			if err := s.provisionSSL(server, domain); err != nil {
				log.Printf("⚠️  Domain %s: SSL provisioning failed (non-fatal): %v", domain.Name, err)
				// Still mark as active, SSL can be retried
				s.updateDomainStatus(bgCtx, domain.ID.String(), "active")
			} else {
				log.Printf("✅ Domain %s: SSL provisioned successfully", domain.Name)
				// Re-generate vhost with SSL enabled and reload
				domain.SSLEnabled = true
				if err := s.provisionDomain(server, domain, targetIPs); err != nil {
					log.Printf("⚠️  Domain %s: SSL vhost update failed: %v", domain.Name, err)
				}
				s.db.Exec(bgCtx, "UPDATE domains SET ssl_enabled = true, status = 'active', updated_at = $1 WHERE id = $2", time.Now(), domain.ID)
			}
		}()
	} else {
		// No server attached, just mark as active
		s.updateDomainStatus(ctx, domain.ID.String(), "active")
	}

	return domain, nil
}

func (s *DomainService) updateDomainStatus(ctx context.Context, id, status string) {
	s.db.Exec(ctx, "UPDATE domains SET status = $1, updated_at = $2 WHERE id = $3", status, time.Now(), id)
}

func (s *DomainService) GetByID(ctx context.Context, id string) (*models.Domain, error) {
	domain := &models.Domain{}
	err := s.db.QueryRow(ctx,
		`SELECT id, user_id, server_id, name, type, document_root, web_server,
		        php_version, ssl_enabled, status, is_load_balancer, created_at, updated_at
		 FROM domains WHERE id = $1`,
		id,
	).Scan(&domain.ID, &domain.UserID, &domain.ServerID, &domain.Name, &domain.Type,
		&domain.DocumentRoot, &domain.WebServer, &domain.PHPVersion, &domain.SSLEnabled,
		&domain.Status, &domain.IsLoadBalancer, &domain.CreatedAt, &domain.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("domain not found")
	}

	if domain.IsLoadBalancer {
		rows, _ := s.db.Query(ctx, "SELECT server_id FROM domain_backend_servers WHERE domain_id = $1", domain.ID)
		defer rows.Close()
		for rows.Next() {
			var svrID uuid.UUID
			if err := rows.Scan(&svrID); err == nil {
				domain.BackendServerIDs = append(domain.BackendServerIDs, svrID)
			}
		}
	}

	return domain, nil
}

func (s *DomainService) List(ctx context.Context, userID, role string, page, perPage int) (*models.PaginatedResponse, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var (
		total   int64
		domains []models.Domain
	)

	// Count query
	countQuery := "SELECT COUNT(*) FROM domains"
	listQuery := `SELECT id, user_id, server_id, name, type, document_root, web_server,
	              php_version, ssl_enabled, status, is_load_balancer, created_at, updated_at FROM domains`

	if role != "admin" {
		countQuery += " WHERE user_id = $1"
		listQuery += " WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3"
		s.db.QueryRow(ctx, countQuery, userID).Scan(&total)
		rows, err := s.db.Query(ctx, listQuery, userID, perPage, offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var d models.Domain
			err := rows.Scan(&d.ID, &d.UserID, &d.ServerID, &d.Name, &d.Type,
				&d.DocumentRoot, &d.WebServer, &d.PHPVersion, &d.SSLEnabled,
				&d.Status, &d.IsLoadBalancer, &d.CreatedAt, &d.UpdatedAt)
			if err != nil {
				return nil, err
			}
			domains = append(domains, d)
		}
	} else {
		listQuery += " ORDER BY created_at DESC LIMIT $1 OFFSET $2"
		s.db.QueryRow(ctx, countQuery).Scan(&total)
		rows, err := s.db.Query(ctx, listQuery, perPage, offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var d models.Domain
			err := rows.Scan(&d.ID, &d.UserID, &d.ServerID, &d.Name, &d.Type,
				&d.DocumentRoot, &d.WebServer, &d.PHPVersion, &d.SSLEnabled,
				&d.Status, &d.IsLoadBalancer, &d.CreatedAt, &d.UpdatedAt)
			if err != nil {
				return nil, err
			}
			domains = append(domains, d)
		}
	}

	// Fetch backend arrays for load balancer domains
	var lbIDs []uuid.UUID
	for _, d := range domains {
		if d.IsLoadBalancer {
			lbIDs = append(lbIDs, d.ID)
		}
	}

	if len(lbIDs) > 0 {
		rows, err := s.db.Query(ctx, "SELECT domain_id, server_id FROM domain_backend_servers WHERE domain_id = ANY($1)", lbIDs)
		if err == nil {
			defer rows.Close()
			lbMap := make(map[uuid.UUID][]uuid.UUID)
			for rows.Next() {
				var dID, sID uuid.UUID
				if err := rows.Scan(&dID, &sID); err == nil {
					lbMap[dID] = append(lbMap[dID], sID)
				}
			}
			for i, d := range domains {
				if d.IsLoadBalancer {
					domains[i].BackendServerIDs = lbMap[d.ID]
				}
			}
		}
	}

	if domains == nil {
		domains = []models.Domain{}
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))

	return &models.PaginatedResponse{
		Data:       domains,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

func (s *DomainService) Update(ctx context.Context, id string, req models.UpdateDomainRequest) (*models.Domain, error) {
	// Build dynamic update
	sets := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.WebServer != "" {
		sets = append(sets, fmt.Sprintf("web_server = $%d", argIdx))
		args = append(args, req.WebServer)
		argIdx++
	}
	if req.PHPVersion != "" {
		sets = append(sets, fmt.Sprintf("php_version = $%d", argIdx))
		args = append(args, req.PHPVersion)
		argIdx++
	}
	if req.DocumentRoot != "" {
		sets = append(sets, fmt.Sprintf("document_root = $%d", argIdx))
		args = append(args, req.DocumentRoot)
		argIdx++
	}
	if req.SSLEnabled != nil {
		sets = append(sets, fmt.Sprintf("ssl_enabled = $%d", argIdx))
		args = append(args, *req.SSLEnabled)
		argIdx++
	}
	if req.Status != "" {
		sets = append(sets, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, req.Status)
		argIdx++
	}

	if len(sets) == 0 {
		return s.GetByID(ctx, id)
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now())
	argIdx++

	args = append(args, id)
	query := fmt.Sprintf(
		`UPDATE domains SET %s WHERE id = $%d
		 RETURNING id, user_id, server_id, name, type, document_root, web_server, php_version, ssl_enabled, status, is_load_balancer, created_at, updated_at`,
		strings.Join(sets, ", "), argIdx,
	)

	domain := &models.Domain{}
	err := s.db.QueryRow(ctx, query, args...).Scan(
		&domain.ID, &domain.UserID, &domain.ServerID, &domain.Name, &domain.Type,
		&domain.DocumentRoot, &domain.WebServer, &domain.PHPVersion, &domain.SSLEnabled,
		&domain.Status, &domain.IsLoadBalancer, &domain.CreatedAt, &domain.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("domain not found")
	}

	// ── Re-provision vhost if web server config changed ──
	needsReprovision := req.WebServer != "" || req.PHPVersion != "" || req.SSLEnabled != nil
	if needsReprovision && domain.ServerID != nil {
		go func() {
			bgCtx := context.Background()
			server, err := s.getServerSSH(bgCtx, domain.ServerID.String())
			if err != nil {
				log.Printf("⚠️  Domain %s: re-provision skipped, server SSH error: %v", domain.Name, err)
				return
			}

			// Handle SSL toggle
			if req.SSLEnabled != nil && *req.SSLEnabled && !domain.SSLEnabled {
				if err := s.provisionSSL(server, domain); err != nil {
					log.Printf("⚠️  Domain %s: SSL provisioning failed: %v", domain.Name, err)
				} else {
					domain.SSLEnabled = true
					s.db.Exec(bgCtx, "UPDATE domains SET ssl_enabled = true WHERE id = $1", domain.ID)
				}
			}

			if err := s.provisionDomain(server, domain, nil); err != nil {
				log.Printf("⚠️  Domain %s: vhost re-provisioning failed: %v", domain.Name, err)
			} else {
				log.Printf("✅ Domain %s: vhost re-provisioned after update", domain.Name)
			}
		}()
	}

	return domain, nil
}

func (s *DomainService) Delete(ctx context.Context, id string, userID uuid.UUID, role string) error {
	// Fetch the domain first so we can deprovision
	domain, err := s.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("domain not found")
	}

	// Ownership check — non-admins can only delete their own domains
	if role != "admin" && domain.UserID != userID {
		return fmt.Errorf("domain not found")
	}

	// Deprovision from server (remove vhost config)
	if err := s.deprovisionDomain(ctx, domain); err != nil {
		log.Printf("⚠️  Domain %s: deprovisioning warning: %v", domain.Name, err)
		// Continue with DB deletion even if deprovisioning fails
	}

	result, err := s.db.Exec(ctx, "DELETE FROM domains WHERE id = $1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("domain not found")
	}
	return nil
}

// ProvisionWildcardSSL issues a wildcard TLS certificate via Certbot + Cloudflare DNS challenge.
func (s *DomainService) ProvisionWildcardSSL(ctx context.Context, domainID string) error {
	var domainName, serverIDStr string
	var cfToken *string
	err := s.db.QueryRow(ctx,
		`SELECT d.name, d.server_id::text, COALESCE(ci.api_token, '')
		 FROM domains d
		 LEFT JOIN cloudflare_integrations ci ON ci.user_id = d.user_id
		 WHERE d.id = $1`, domainID,
	).Scan(&domainName, &serverIDStr, &cfToken)
	if err != nil {
		return fmt.Errorf("domain not found: %w", err)
	}

	srv, err := s.getServerSSH(ctx, serverIDStr)
	if err != nil {
		return fmt.Errorf("server not available: %w", err)
	}

	var adminEmail string
	s.db.QueryRow(ctx, `SELECT email FROM users WHERE role = 'admin' LIMIT 1`).Scan(&adminEmail)
	if adminEmail == "" {
		adminEmail = "admin@" + domainName
	}

	cfAPIToken := ""
	if cfToken != nil {
		cfAPIToken = *cfToken
	}
	if cfAPIToken == "" {
		return fmt.Errorf("Cloudflare API token required for wildcard SSL DNS challenge")
	}

	script := fmt.Sprintf(`
set -e
apt-get install -y certbot python3-certbot-dns-cloudflare 2>/dev/null || true
mkdir -p /root/.secrets
cat > /root/.secrets/cloudflare.ini << 'CFEOF'
dns_cloudflare_api_token = %s
CFEOF
chmod 600 /root/.secrets/cloudflare.ini
certbot certonly --dns-cloudflare \
  --dns-cloudflare-credentials /root/.secrets/cloudflare.ini \
  -d "*.%s" -d "%s" \
  --non-interactive --agree-tos -m "%s" \
  --preferred-challenges dns-01
echo "WILDCARD_SSL_ISSUED"
`, cfAPIToken, domainName, domainName, adminEmail)

	out, err := provisioner.RunScript(srv, script)
	if err != nil || !strings.Contains(out, "WILDCARD_SSL_ISSUED") {
		return fmt.Errorf("wildcard SSL failed: %w\n%s", err, out)
	}

	// Update domain to ssl_enabled with wildcard cert paths
	s.db.Exec(ctx,
		`UPDATE domains SET ssl_enabled = true, ssl_certificate = $1, ssl_key = $2 WHERE id = $3`,
		fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", domainName),
		fmt.Sprintf("/etc/letsencrypt/live/%s/privkey.pem", domainName),
		domainID,
	)

	log.Printf("✅ Wildcard SSL issued for *.%s", domainName)
	return nil
}
