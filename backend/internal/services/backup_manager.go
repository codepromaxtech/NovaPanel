package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	novacrypto "github.com/novapanel/novapanel/internal/crypto"
	"github.com/novapanel/novapanel/internal/provisioner"
)

// BackupManager handles actual backup execution and restore via SSH
type BackupManager struct {
	pool *pgxpool.Pool
}

func NewBackupManager(pool *pgxpool.Pool) *BackupManager {
	return &BackupManager{pool: pool}
}

func (s *BackupManager) getServer(ctx context.Context, serverID string) (provisioner.ServerInfo, error) {
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

// BackupDatabase creates an actual database backup via SSH
func (s *BackupManager) BackupDatabase(ctx context.Context, userID uuid.UUID, serverID, engine, dbName string) (map[string]interface{}, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupDir := "/var/backups/novapanel"
	var filename, cmd string

	switch engine {
	case "mysql", "mariadb":
		filename = fmt.Sprintf("%s/db_%s_%s.sql.gz", backupDir, dbName, timestamp)
		cmd = fmt.Sprintf(`mkdir -p %s && mysqldump -u root --single-transaction --routines --triggers '%s' 2>/dev/null | gzip > '%s' && ls -lh '%s' && echo 'Backup completed'`, backupDir, dbName, filename, filename)
	case "postgresql", "postgres":
		filename = fmt.Sprintf("%s/db_%s_%s.sql.gz", backupDir, dbName, timestamp)
		cmd = fmt.Sprintf(`mkdir -p %s && sudo -u postgres pg_dump '%s' 2>/dev/null | gzip > '%s' && ls -lh '%s' && echo 'Backup completed'`, backupDir, dbName, filename, filename)
	case "mongodb", "mongo":
		filename = fmt.Sprintf("%s/db_%s_%s", backupDir, dbName, timestamp)
		cmd = fmt.Sprintf(`mkdir -p %s && mongodump --db '%s' --out '%s' 2>&1 && du -sh '%s' && echo 'Backup completed'`, backupDir, dbName, filename, filename)
	default:
		return nil, fmt.Errorf("unsupported engine: %s", engine)
	}

	output, err := provisioner.RunScript(server, cmd)
	if err != nil {
		// Still save the backup record as failed
		s.pool.Exec(ctx,
			`INSERT INTO backups (user_id, server_id, type, storage, status, path)
			 VALUES ($1, $2::uuid, 'database', 'local', 'failed', $3)`,
			userID, serverID, filename)
		return nil, fmt.Errorf("backup failed: %w\nOutput: %s", err, output)
	}

	// Save backup record
	s.pool.Exec(ctx,
		`INSERT INTO backups (user_id, server_id, type, storage, status, path, started_at, completed_at)
		 VALUES ($1, $2::uuid, 'database', 'local', 'completed', $3, NOW(), NOW())`,
		userID, serverID, filename)

	return map[string]interface{}{
		"path":   filename,
		"output": output,
		"engine": engine,
		"db":     dbName,
	}, nil
}

// BackupSite creates a site/directory backup via SSH
func (s *BackupManager) BackupSite(ctx context.Context, userID uuid.UUID, serverID, sitePath, siteName string) (map[string]interface{}, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupDir := "/var/backups/novapanel"
	if siteName == "" {
		siteName = "site"
	}
	filename := fmt.Sprintf("%s/site_%s_%s.tar.gz", backupDir, siteName, timestamp)

	cmd := fmt.Sprintf(`mkdir -p %s && tar -czf '%s' -C '%s' . 2>&1 && ls -lh '%s' && echo 'Site backup completed'`,
		backupDir, filename, sitePath, filename)

	output, err := provisioner.RunScript(server, cmd)
	if err != nil {
		s.pool.Exec(ctx,
			`INSERT INTO backups (user_id, server_id, type, storage, status, path)
			 VALUES ($1, $2::uuid, 'site', 'local', 'failed', $3)`,
			userID, serverID, filename)
		return nil, fmt.Errorf("backup failed: %w\nOutput: %s", err, output)
	}

	s.pool.Exec(ctx,
		`INSERT INTO backups (user_id, server_id, type, storage, status, path, started_at, completed_at)
		 VALUES ($1, $2::uuid, 'site', 'local', 'completed', $3, NOW(), NOW())`,
		userID, serverID, filename)

	return map[string]interface{}{
		"path":   filename,
		"output": output,
		"site":   siteName,
	}, nil
}

// BackupFull creates a full server backup (all DBs + /var/www + configs)
func (s *BackupManager) BackupFull(ctx context.Context, userID uuid.UUID, serverID string) (map[string]interface{}, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupDir := fmt.Sprintf("/var/backups/novapanel/full_%s", timestamp)
	filename := fmt.Sprintf("/var/backups/novapanel/full_%s.tar.gz", timestamp)

	cmd := fmt.Sprintf(`mkdir -p %s
# Backup all MySQL databases
mysqldump -u root --all-databases 2>/dev/null | gzip > %s/mysql_all.sql.gz || true
# Backup all PostgreSQL databases
sudo -u postgres pg_dumpall 2>/dev/null | gzip > %s/postgres_all.sql.gz || true
# Backup web files
tar -czf %s/www.tar.gz /var/www 2>/dev/null || true
# Backup nginx configs
tar -czf %s/nginx.tar.gz /etc/nginx 2>/dev/null || true
# Backup apache configs
tar -czf %s/apache.tar.gz /etc/apache2 2>/dev/null || true
# Backup crontabs
tar -czf %s/crontabs.tar.gz /var/spool/cron 2>/dev/null || true
# Create final archive
tar -czf %s %s 2>&1
rm -rf %s
ls -lh %s && echo 'Full backup completed'`,
		backupDir,
		backupDir, backupDir, backupDir, backupDir, backupDir, backupDir,
		filename, backupDir, backupDir, filename)

	output, err := provisioner.RunScript(server, cmd)
	if err != nil {
		s.pool.Exec(ctx,
			`INSERT INTO backups (user_id, server_id, type, storage, status, path)
			 VALUES ($1, $2::uuid, 'full', 'local', 'failed', $3)`,
			userID, serverID, filename)
		return nil, fmt.Errorf("backup failed: %w\nOutput: %s", err, output)
	}

	s.pool.Exec(ctx,
		`INSERT INTO backups (user_id, server_id, type, storage, status, path, started_at, completed_at)
		 VALUES ($1, $2::uuid, 'full', 'local', 'completed', $3, NOW(), NOW())`,
		userID, serverID, filename)

	return map[string]interface{}{
		"path":   filename,
		"output": output,
	}, nil
}

// RestoreDatabase restores a database from a backup file
func (s *BackupManager) RestoreDatabase(ctx context.Context, serverID, engine, dbName, backupPath string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}

	var cmd string
	switch engine {
	case "mysql", "mariadb":
		if isGzipped(backupPath) {
			cmd = fmt.Sprintf("gunzip -c '%s' | mysql -u root '%s' 2>&1 && echo 'Database restored'", backupPath, dbName)
		} else {
			cmd = fmt.Sprintf("mysql -u root '%s' < '%s' 2>&1 && echo 'Database restored'", dbName, backupPath)
		}
	case "postgresql", "postgres":
		if isGzipped(backupPath) {
			cmd = fmt.Sprintf("gunzip -c '%s' | sudo -u postgres psql -d '%s' 2>&1 && echo 'Database restored'", backupPath, dbName)
		} else {
			cmd = fmt.Sprintf("sudo -u postgres psql -d '%s' < '%s' 2>&1 && echo 'Database restored'", dbName, backupPath)
		}
	case "mongodb", "mongo":
		cmd = fmt.Sprintf("mongorestore --db '%s' --drop '%s' 2>&1 && echo 'Database restored'", dbName, backupPath)
	default:
		return "", fmt.Errorf("unsupported engine")
	}

	return provisioner.RunScript(server, cmd)
}

// RestoreSite restores a site from a tar.gz backup
func (s *BackupManager) RestoreSite(ctx context.Context, serverID, sitePath, backupPath string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}

	cmd := fmt.Sprintf(`mkdir -p '%s' && tar -xzf '%s' -C '%s' 2>&1 && echo 'Site restored to %s'`, sitePath, backupPath, sitePath, sitePath)
	return provisioner.RunScript(server, cmd)
}

// ListBackupFiles lists backup files on a server
func (s *BackupManager) ListBackupFiles(ctx context.Context, serverID string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}

	cmd := `ls -lhR /var/backups/novapanel/ 2>/dev/null || echo 'No backups found'`
	return provisioner.RunScript(server, cmd)
}

// DeleteBackupFile deletes a backup file from a server
func (s *BackupManager) DeleteBackupFile(ctx context.Context, serverID, filePath string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}

	// Only allow deleting from /var/backups/novapanel
	if !startsWith(filePath, "/var/backups/novapanel") {
		return "", fmt.Errorf("can only delete files in /var/backups/novapanel")
	}

	cmd := fmt.Sprintf("rm -rf '%s' 2>&1 && echo 'Deleted'", filePath)
	return provisioner.RunScript(server, cmd)
}

func isGzipped(path string) bool {
	return len(path) > 3 && path[len(path)-3:] == ".gz"
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
