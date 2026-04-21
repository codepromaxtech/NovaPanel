package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/provisioner"
)

// PHPVersion represents an installed PHP version on a server
type PHPVersion struct {
	ID          uuid.UUID `json:"id"`
	ServerID    uuid.UUID `json:"server_id"`
	Version     string    `json:"version"`
	IsDefault   bool      `json:"is_default"`
	Status      string    `json:"status"`
	InstalledAt time.Time `json:"installed_at"`
}

type PHPService struct {
	db *pgxpool.Pool
}

func NewPHPService(db *pgxpool.Pool) *PHPService {
	return &PHPService{db: db}
}

// getServerSSH retrieves SSH connection info for a server
func (s *PHPService) getServerSSH(ctx context.Context, serverID string) (provisioner.ServerInfo, error) {
	var server provisioner.ServerInfo
	var port int
	err := s.db.QueryRow(ctx,
		`SELECT host(ip_address), port, ssh_user, COALESCE(ssh_key, ''), COALESCE(ssh_password, ''), COALESCE(auth_method, 'password')
		 FROM servers WHERE id = $1`, serverID,
	).Scan(&server.IPAddress, &port, &server.SSHUser, &server.SSHKey, &server.SSHPassword, &server.AuthMethod)
	server.Port = port
	return server, err
}

// ListInstalled returns all PHP versions installed on a server
func (s *PHPService) ListInstalled(ctx context.Context, serverID string) ([]PHPVersion, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, server_id, version, is_default, status, installed_at
		 FROM php_versions WHERE server_id = $1 ORDER BY version`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []PHPVersion
	for rows.Next() {
		var v PHPVersion
		if err := rows.Scan(&v.ID, &v.ServerID, &v.Version, &v.IsDefault, &v.Status, &v.InstalledAt); err != nil {
			continue
		}
		versions = append(versions, v)
	}
	if versions == nil {
		versions = []PHPVersion{}
	}
	return versions, nil
}

// Install installs a PHP version on a server via SSH
func (s *PHPService) Install(ctx context.Context, serverID, version string) (*PHPVersion, error) {
	// Validate version format
	validVersions := []string{"7.4", "8.0", "8.1", "8.2", "8.3", "8.4", "8.5"}
	valid := false
	for _, v := range validVersions {
		if version == v {
			valid = true
			break
		}
	}
	if !valid {
		return nil, fmt.Errorf("unsupported PHP version: %s (supported: %s)", version, strings.Join(validVersions, ", "))
	}

	sid, err := uuid.Parse(serverID)
	if err != nil {
		return nil, fmt.Errorf("invalid server ID")
	}

	// Check if already installed
	var exists bool
	s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM php_versions WHERE server_id = $1 AND version = $2)", sid, version).Scan(&exists)
	if exists {
		return nil, fmt.Errorf("PHP %s is already installed on this server", version)
	}

	// Insert record with status 'installing'
	pv := &PHPVersion{}
	err = s.db.QueryRow(ctx,
		`INSERT INTO php_versions (server_id, version, status)
		 VALUES ($1, $2, 'installing')
		 RETURNING id, server_id, version, is_default, status, installed_at`,
		sid, version,
	).Scan(&pv.ID, &pv.ServerID, &pv.Version, &pv.IsDefault, &pv.Status, &pv.InstalledAt)
	if err != nil {
		return nil, err
	}

	// Install async via SSH
	go func() {
		bgCtx := context.Background()
		server, err := s.getServerSSH(bgCtx, serverID)
		if err != nil {
			log.Printf("❌ PHP %s install: failed to get SSH info: %v", version, err)
			s.db.Exec(bgCtx, "UPDATE php_versions SET status = 'error' WHERE id = $1", pv.ID)
			return
		}

		script := fmt.Sprintf(`
# Add PHP PPA if not present
if ! ls /etc/apt/sources.list.d/ondrej-ubuntu-php* &>/dev/null; then
    apt-get update -qq
    apt-get install -y software-properties-common 2>&1
    add-apt-repository -y ppa:ondrej/php 2>&1
    apt-get update -qq
fi

# Install PHP and common extensions
apt-get install -y \
    php%s php%s-fpm php%s-cli php%s-common \
    php%s-mysql php%s-pgsql php%s-sqlite3 \
    php%s-curl php%s-gd php%s-mbstring \
    php%s-xml php%s-zip php%s-bcmath \
    php%s-intl php%s-readline php%s-opcache 2>&1

# Start and enable PHP-FPM
systemctl start php%s-fpm 2>&1
systemctl enable php%s-fpm 2>&1

# Verify installation
php%s -v 2>&1 && echo "PHP_INSTALL_OK"
`, version, version, version, version,
			version, version, version,
			version, version, version,
			version, version, version,
			version, version, version,
			version, version, version)

		output, err := provisioner.RunScript(server, script)
		if err != nil || !strings.Contains(output, "PHP_INSTALL_OK") {
			log.Printf("❌ PHP %s install failed: %v (output: %s)", version, err, output)
			s.db.Exec(bgCtx, "UPDATE php_versions SET status = 'error' WHERE id = $1", pv.ID)
			return
		}

		log.Printf("✅ PHP %s installed on server %s", version, serverID)
		s.db.Exec(bgCtx, "UPDATE php_versions SET status = 'active' WHERE id = $1", pv.ID)
	}()

	return pv, nil
}

// SetDefault sets a PHP version as default on a server
func (s *PHPService) SetDefault(ctx context.Context, serverID, version string) error {
	sid, err := uuid.Parse(serverID)
	if err != nil {
		return fmt.Errorf("invalid server ID")
	}

	// Verify it's installed
	var status string
	err = s.db.QueryRow(ctx,
		"SELECT status FROM php_versions WHERE server_id = $1 AND version = $2", sid, version).Scan(&status)
	if err != nil {
		return fmt.Errorf("PHP %s is not installed on this server", version)
	}
	if status != "active" {
		return fmt.Errorf("PHP %s is not ready (status: %s)", version, status)
	}

	// Update default flag
	s.db.Exec(ctx, "UPDATE php_versions SET is_default = false WHERE server_id = $1", sid)
	s.db.Exec(ctx, "UPDATE php_versions SET is_default = true WHERE server_id = $1 AND version = $2", sid, version)

	// Set CLI default via SSH
	go func() {
		bgCtx := context.Background()
		server, err := s.getServerSSH(bgCtx, serverID)
		if err != nil {
			return
		}

		script := fmt.Sprintf(`
update-alternatives --set php /usr/bin/php%s 2>&1 || true
update-alternatives --set php-config /usr/bin/php-config%s 2>&1 || true
update-alternatives --set phpize /usr/bin/phpize%s 2>&1 || true
echo "PHP_DEFAULT_SET"
`, version, version, version)

		output, _ := provisioner.RunScript(server, script)
		log.Printf("PHP %s set as default: %s", version, strings.TrimSpace(output))
	}()

	return nil
}

// SwitchDomain changes the PHP version for a specific domain's vhost
func (s *PHPService) SwitchDomain(ctx context.Context, domainID, version string) error {
	did, err := uuid.Parse(domainID)
	if err != nil {
		return fmt.Errorf("invalid domain ID")
	}

	// Get domain and its server
	var serverID uuid.UUID
	var domainName, webServer, currentPHP string
	err = s.db.QueryRow(ctx,
		`SELECT server_id, name, web_server, php_version FROM domains WHERE id = $1`, did,
	).Scan(&serverID, &domainName, &webServer, &currentPHP)
	if err != nil {
		return fmt.Errorf("domain not found")
	}

	if currentPHP == version {
		return nil // Already using this version
	}

	// Verify the version is installed on the server
	var status string
	err = s.db.QueryRow(ctx,
		"SELECT status FROM php_versions WHERE server_id = $1 AND version = $2", serverID, version).Scan(&status)
	if err != nil || status != "active" {
		return fmt.Errorf("PHP %s is not installed/active on the server", version)
	}

	// Update domain record
	_, err = s.db.Exec(ctx, "UPDATE domains SET php_version = $1, updated_at = $2 WHERE id = $3", version, time.Now(), did)
	if err != nil {
		return err
	}

	// Update vhost config via SSH to use the new PHP-FPM socket
	go func() {
		bgCtx := context.Background()
		server, err := s.getServerSSH(bgCtx, serverID.String())
		if err != nil {
			return
		}

		var script string
		switch webServer {
		case "apache":
			script = fmt.Sprintf(`
# Disable old PHP-FPM proxy, enable new one
a2disconf php%s-fpm 2>/dev/null || true
a2enconf php%s-fpm 2>/dev/null || true

# Update the vhost to use new PHP-FPM socket
sed -i 's|php%s-fpm.sock|php%s-fpm.sock|g' '/etc/apache2/sites-available/%s.conf'
apache2ctl configtest 2>&1 && systemctl reload apache2 2>&1
echo "PHP_SWITCH_OK"
`, currentPHP, version, currentPHP, version, domainName)
		default: // nginx
			script = fmt.Sprintf(`
# Update the vhost to use new PHP-FPM socket
sed -i 's|php%s-fpm.sock|php%s-fpm.sock|g' '/etc/nginx/sites-available/%s'
nginx -t 2>&1 && systemctl reload nginx 2>&1
echo "PHP_SWITCH_OK"
`, currentPHP, version, domainName)
		}

		output, err := provisioner.RunScript(server, script)
		if err != nil {
			log.Printf("⚠️  PHP switch for %s failed: %v", domainName, err)
		} else {
			log.Printf("✅ PHP %s → %s for domain %s: %s", currentPHP, version, domainName, strings.TrimSpace(output))
		}
	}()

	return nil
}

// Uninstall removes a PHP version from a server
func (s *PHPService) Uninstall(ctx context.Context, serverID, version string) error {
	sid, err := uuid.Parse(serverID)
	if err != nil {
		return fmt.Errorf("invalid server ID")
	}

	// Check if any domains are using this version
	var domainCount int
	s.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM domains WHERE server_id = $1 AND php_version = $2", sid, version).Scan(&domainCount)
	if domainCount > 0 {
		return fmt.Errorf("cannot uninstall PHP %s: %d domain(s) are still using it", version, domainCount)
	}

	// Check if it's the default
	var isDefault bool
	s.db.QueryRow(ctx,
		"SELECT is_default FROM php_versions WHERE server_id = $1 AND version = $2", sid, version).Scan(&isDefault)
	if isDefault {
		return fmt.Errorf("cannot uninstall the default PHP version — switch default first")
	}

	// Remove from DB
	result, err := s.db.Exec(ctx, "DELETE FROM php_versions WHERE server_id = $1 AND version = $2", sid, version)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("PHP %s is not installed on this server", version)
	}

	// Uninstall via SSH (async)
	go func() {
		bgCtx := context.Background()
		server, err := s.getServerSSH(bgCtx, serverID)
		if err != nil {
			return
		}

		script := fmt.Sprintf(`
systemctl stop php%s-fpm 2>/dev/null || true
systemctl disable php%s-fpm 2>/dev/null || true
apt-get purge -y 'php%s-*' 2>&1
apt-get autoremove -y 2>&1
echo "PHP_UNINSTALL_OK"
`, version, version, version)

		output, _ := provisioner.RunScript(server, script)
		log.Printf("PHP %s uninstalled from %s: %s", version, serverID, strings.TrimSpace(output))
	}()

	return nil
}
