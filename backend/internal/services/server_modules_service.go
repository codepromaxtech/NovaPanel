package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	novacrypto "github.com/novapanel/novapanel/internal/crypto"
	"github.com/novapanel/novapanel/internal/provisioner"
)

var moduleInstallScripts = map[string]string{
	"web-nginx":         `DEBIAN_FRONTEND=noninteractive apt-get install -y nginx && systemctl enable --now nginx && echo "MODULE_INSTALLED"`,
	"web-apache":        `DEBIAN_FRONTEND=noninteractive apt-get install -y apache2 && systemctl enable --now apache2 && echo "MODULE_INSTALLED"`,
	"database-mysql":    `DEBIAN_FRONTEND=noninteractive apt-get install -y mysql-server && systemctl enable --now mysql && echo "MODULE_INSTALLED"`,
	"database-postgres": `DEBIAN_FRONTEND=noninteractive apt-get install -y postgresql postgresql-contrib && systemctl enable --now postgresql && echo "MODULE_INSTALLED"`,
	"database-redis":    `DEBIAN_FRONTEND=noninteractive apt-get install -y redis-server && systemctl enable --now redis-server && echo "MODULE_INSTALLED"`,
	"database-mongo": `
curl -fsSL https://www.mongodb.org/static/pgp/server-7.0.asc | gpg -o /usr/share/keyrings/mongodb-server-7.0.gpg --dearmor
echo "deb [ arch=amd64,arm64 signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg ] https://repo.mongodb.org/apt/ubuntu $(lsb_release -cs)/mongodb-org/7.0 multiverse" \
  > /etc/apt/sources.list.d/mongodb-org-7.0.list
apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y mongodb-org && systemctl enable --now mongod && echo "MODULE_INSTALLED"`,
	"docker": `curl -fsSL https://get.docker.com | sh && systemctl enable --now docker && echo "MODULE_INSTALLED"`,
	"kubernetes": `curl -sfL https://get.k3s.io | sh - && echo "MODULE_INSTALLED"`,
	"monitoring": `
ARCH=$(dpkg --print-architecture)
VERSION=$(curl -s https://api.github.com/repos/prometheus/node_exporter/releases/latest | grep tag_name | cut -d '"' -f4)
curl -sL "https://github.com/prometheus/node_exporter/releases/download/${VERSION}/node_exporter-${VERSION#v}.linux-${ARCH}.tar.gz" \
  | tar xz --strip-components=1 -C /usr/local/bin/ node_exporter-${VERSION#v}.linux-${ARCH}/node_exporter
useradd -rs /bin/false node_exporter 2>/dev/null || true
cat > /etc/systemd/system/node_exporter.service << 'SVC'
[Unit]
Description=Node Exporter
[Service]
User=node_exporter
ExecStart=/usr/local/bin/node_exporter
[Install]
WantedBy=multi-user.target
SVC
systemctl daemon-reload && systemctl enable --now node_exporter && echo "MODULE_INSTALLED"`,
	"mail": `DEBIAN_FRONTEND=noninteractive apt-get install -y postfix dovecot-core dovecot-imapd dovecot-pop3d opendkim opendkim-tools && systemctl enable --now postfix dovecot && echo "MODULE_INSTALLED"`,
	"firewall": `apt-get install -y ufw && ufw --force enable && ufw default deny incoming && ufw default allow outgoing && ufw allow 22/tcp && echo "MODULE_INSTALLED"`,
	"dns":      `DEBIAN_FRONTEND=noninteractive apt-get install -y bind9 bind9utils && systemctl enable --now bind9 && echo "MODULE_INSTALLED"`,
}

var moduleUninstallScripts = map[string]string{
	"web-nginx":         `systemctl stop nginx 2>/dev/null; apt-get remove -y nginx nginx-common && apt-get autoremove -y && echo "MODULE_REMOVED"`,
	"web-apache":        `systemctl stop apache2 2>/dev/null; apt-get remove -y apache2 && apt-get autoremove -y && echo "MODULE_REMOVED"`,
	"database-mysql":    `systemctl stop mysql 2>/dev/null; apt-get remove -y mysql-server && apt-get autoremove -y && echo "MODULE_REMOVED"`,
	"database-postgres": `systemctl stop postgresql 2>/dev/null; apt-get remove -y postgresql && apt-get autoremove -y && echo "MODULE_REMOVED"`,
	"database-redis":    `systemctl stop redis-server 2>/dev/null; apt-get remove -y redis-server && apt-get autoremove -y && echo "MODULE_REMOVED"`,
	"database-mongo":    `systemctl stop mongod 2>/dev/null; apt-get remove -y mongodb-org && apt-get autoremove -y && echo "MODULE_REMOVED"`,
	"docker":            `systemctl stop docker 2>/dev/null; apt-get remove -y docker-ce docker-ce-cli containerd.io && apt-get autoremove -y && echo "MODULE_REMOVED"`,
	"monitoring":        `systemctl stop node_exporter 2>/dev/null; systemctl disable node_exporter 2>/dev/null; rm -f /usr/local/bin/node_exporter /etc/systemd/system/node_exporter.service && systemctl daemon-reload && echo "MODULE_REMOVED"`,
	"mail":              `systemctl stop postfix dovecot 2>/dev/null; apt-get remove -y postfix dovecot-core dovecot-imapd dovecot-pop3d && apt-get autoremove -y && echo "MODULE_REMOVED"`,
	"firewall":          `ufw disable 2>/dev/null; apt-get remove -y ufw && echo "MODULE_REMOVED"`,
	"dns":               `systemctl stop bind9 2>/dev/null; apt-get remove -y bind9 && apt-get autoremove -y && echo "MODULE_REMOVED"`,
}

// Available modules
var AvailableModules = []ModuleInfo{
	{ID: "web-nginx", Label: "Nginx Web Server", Category: "web", Icon: "🌐"},
	{ID: "web-apache", Label: "Apache Web Server", Category: "web", Icon: "🌐"},
	{ID: "database-mysql", Label: "MySQL / MariaDB", Category: "database", Icon: "🗄️"},
	{ID: "database-postgres", Label: "PostgreSQL", Category: "database", Icon: "🗄️"},
	{ID: "database-mongo", Label: "MongoDB", Category: "database", Icon: "🗄️"},
	{ID: "database-redis", Label: "Redis", Category: "database", Icon: "🗄️"},
	{ID: "docker", Label: "Docker Engine", Category: "containers", Icon: "🐳"},
	{ID: "kubernetes", Label: "Kubernetes", Category: "containers", Icon: "☸️"},
	{ID: "monitoring", Label: "Monitoring Agent", Category: "system", Icon: "📊"},
	{ID: "mail", Label: "Mail Server", Category: "services", Icon: "📧"},
	{ID: "firewall", Label: "Firewall (UFW)", Category: "system", Icon: "🔥"},
	{ID: "dns", Label: "DNS Server", Category: "services", Icon: "🌍"},
}

type ModuleInfo struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Category string `json:"category"`
	Icon     string `json:"icon"`
}

type ServerModule struct {
	ID          string    `json:"id"`
	ServerID    string    `json:"server_id"`
	Module      string    `json:"module"`
	Enabled     bool      `json:"enabled"`
	Config      string    `json:"config"`
	InstalledAt time.Time `json:"installed_at"`
}

type ServerModulesService struct {
	db        *pgxpool.Pool
	cryptoKey []byte
}

func NewServerModulesService(db *pgxpool.Pool, encryptionKey string) *ServerModulesService {
	return &ServerModulesService{db: db, cryptoKey: novacrypto.DeriveKey(encryptionKey)}
}

func (s *ServerModulesService) getServerSSH(ctx context.Context, serverID string) (provisioner.ServerInfo, error) {
	var srv provisioner.ServerInfo
	var port int
	var encKey, encPassword string
	err := s.db.QueryRow(ctx,
		`SELECT host(ip_address), port, ssh_user,
		        COALESCE(ssh_key, ''), COALESCE(ssh_password, ''), COALESCE(auth_method, 'password'), COALESCE(is_local, FALSE)
		 FROM servers WHERE id = $1`, serverID,
	).Scan(&srv.IPAddress, &port, &srv.SSHUser, &encKey, &encPassword, &srv.AuthMethod, &srv.IsLocal)
	if err != nil {
		return srv, fmt.Errorf("server not found: %w", err)
	}
	srv.Port = port
	if encKey != "" {
		if dec, err := novacrypto.Decrypt(encKey, s.cryptoKey); err == nil {
			srv.SSHKey = dec
		}
	}
	if encPassword != "" {
		if dec, err := novacrypto.Decrypt(encPassword, s.cryptoKey); err == nil {
			srv.SSHPassword = dec
		}
	}
	return srv, nil
}

func (s *ServerModulesService) runModuleScript(serverID uuid.UUID, module, scriptType string) {
	scripts := moduleInstallScripts
	if scriptType == "uninstall" {
		scripts = moduleUninstallScripts
	}
	script, ok := scripts[module]
	if !ok {
		return
	}
	ctx := context.Background()
	srv, err := s.getServerSSH(ctx, serverID.String())
	if err != nil {
		log.Printf("module %s %s: SSH error: %v", scriptType, module, err)
		return
	}
	out, err := provisioner.RunScript(srv, script)
	if err != nil {
		log.Printf("module %s %s error: %v — %s", scriptType, module, err, out)
	} else {
		log.Printf("module %s %s: %s", scriptType, module, out)
	}
}

func (s *ServerModulesService) EnableModule(ctx context.Context, serverID uuid.UUID, module string) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO server_modules (server_id, module, enabled) VALUES ($1, $2, true)
		 ON CONFLICT (server_id, module) DO UPDATE SET enabled = true`,
		serverID, module)
	if err != nil {
		return err
	}
	go s.runModuleScript(serverID, module, "install")
	return nil
}

func (s *ServerModulesService) DisableModule(ctx context.Context, serverID uuid.UUID, module string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE server_modules SET enabled = false WHERE server_id = $1 AND module = $2`,
		serverID, module)
	if err != nil {
		return err
	}
	go s.runModuleScript(serverID, module, "uninstall")
	return nil
}

func (s *ServerModulesService) RemoveModule(ctx context.Context, serverID uuid.UUID, module string) error {
	_, err := s.db.Exec(ctx,
		`DELETE FROM server_modules WHERE server_id = $1 AND module = $2`,
		serverID, module)
	return err
}

func (s *ServerModulesService) ListModules(ctx context.Context, serverID uuid.UUID) ([]ServerModule, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, server_id, module, enabled, COALESCE(config::text, '{}'), installed_at
		 FROM server_modules WHERE server_id = $1 AND enabled = true ORDER BY installed_at`,
		serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []ServerModule
	for rows.Next() {
		var m ServerModule
		rows.Scan(&m.ID, &m.ServerID, &m.Module, &m.Enabled, &m.Config, &m.InstalledAt)
		modules = append(modules, m)
	}
	return modules, nil
}

func (s *ServerModulesService) GetEnabledModulesForServer(ctx context.Context, serverID uuid.UUID) ([]string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT module FROM server_modules WHERE server_id = $1 AND enabled = true`,
		serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []string
	for rows.Next() {
		var m string
		rows.Scan(&m)
		modules = append(modules, m)
	}
	return modules, nil
}

func (s *ServerModulesService) GetActiveModulesGlobal(ctx context.Context) ([]string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT DISTINCT module FROM server_modules WHERE enabled = true ORDER BY module`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []string
	for rows.Next() {
		var m string
		rows.Scan(&m)
		modules = append(modules, m)
	}
	return modules, nil
}

func (s *ServerModulesService) GetServersForModule(ctx context.Context, module string) ([]string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT server_id FROM server_modules WHERE module = $1 AND enabled = true`,
		module)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *ServerModulesService) GetModuleCounts(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.Query(ctx,
		`SELECT module, COUNT(*) FROM server_modules WHERE enabled = true GROUP BY module`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var m string
		var c int
		rows.Scan(&m, &c)
		counts[m] = c
	}
	return counts, nil
}

func (s *ServerModulesService) SetModules(ctx context.Context, serverID uuid.UUID, modules []string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Disable all existing
	_, err = tx.Exec(ctx, `UPDATE server_modules SET enabled = false WHERE server_id = $1`, serverID)
	if err != nil {
		return err
	}

	// Enable selected
	for _, m := range modules {
		_, err = tx.Exec(ctx,
			`INSERT INTO server_modules (server_id, module, enabled) VALUES ($1, $2, true)
			 ON CONFLICT (server_id, module) DO UPDATE SET enabled = true`,
			serverID, m)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
