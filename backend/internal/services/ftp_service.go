package services

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	novacrypto "github.com/novapanel/novapanel/internal/crypto"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/provisioner"
)

type FTPService struct {
	pool      *pgxpool.Pool
	cryptoKey []byte
}

func NewFTPService(pool *pgxpool.Pool, encryptionKey string) *FTPService {
	return &FTPService{pool: pool, cryptoKey: novacrypto.DeriveKey(encryptionKey)}
}

func (s *FTPService) getServerSSH(ctx context.Context, serverID string) (provisioner.ServerInfo, error) {
	var srv provisioner.ServerInfo
	var port int
	var encKey, encPassword string
	err := s.pool.QueryRow(ctx,
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

type CreateFTPRequest struct {
	ServerID string `json:"server_id" binding:"required"`
	Username string `json:"username"  binding:"required"`
	Password string `json:"password"  binding:"required,min=8"`
	HomeDir  string `json:"home_dir"`
	QuotaMB  int    `json:"quota_mb"`
}

func (s *FTPService) Create(ctx context.Context, userID uuid.UUID, req CreateFTPRequest) (*models.FTPAccount, error) {
	if req.HomeDir == "" {
		req.HomeDir = "/var/www/" + req.Username
	}
	if req.QuotaMB == 0 {
		req.QuotaMB = 1024
	}

	// Encrypt password before storing
	encPassword, err := novacrypto.Encrypt(req.Password, s.cryptoKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt password: %w", err)
	}

	acc := &models.FTPAccount{}
	err = s.pool.QueryRow(ctx,
		`INSERT INTO ftp_accounts (user_id, server_id, username, password_enc, home_dir, quota_mb)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, server_id, username, home_dir, quota_mb, is_active, created_at`,
		userID, req.ServerID, req.Username, encPassword, req.HomeDir, req.QuotaMB,
	).Scan(&acc.ID, &acc.UserID, &acc.ServerID, &acc.Username, &acc.HomeDir, &acc.QuotaMB, &acc.IsActive, &acc.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create FTP account: %w", err)
	}

	go s.provisionFTP(req.ServerID, req.Username, req.Password, req.HomeDir)
	return acc, nil
}

func (s *FTPService) provisionFTP(serverID, username, password, homeDir string) {
	ctx := context.Background()
	srv, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		log.Printf("FTP provision: SSH error: %v", err)
		return
	}

	script := fmt.Sprintf(`
set -e
# Install vsftpd if needed
which vsftpd || DEBIAN_FRONTEND=noninteractive apt-get install -y vsftpd

# Create system user if not exists
id "ftp_%s" &>/dev/null || useradd -m -s /bin/bash -d "%s" "ftp_%s"
echo "ftp_%s:%s" | chpasswd
mkdir -p "%s"
chown "ftp_%s": "%s"

# Add to chroot list
touch /etc/vsftpd.chroot_list
grep -qxF "ftp_%s" /etc/vsftpd.chroot_list || echo "ftp_%s" >> /etc/vsftpd.chroot_list

# Ensure vsftpd is running
systemctl enable vsftpd 2>/dev/null || true
systemctl restart vsftpd 2>/dev/null || true
echo "FTP_CREATED"
`, username, homeDir, username, username, password,
		homeDir, username, homeDir, username, username)

	out, err := provisioner.RunScript(srv, script)
	if err != nil {
		log.Printf("FTP provision error: %v — %s", err, out)
	}
}

func (s *FTPService) List(ctx context.Context, userID uuid.UUID, role string) ([]models.FTPAccount, error) {
	var args []interface{}
	query := `SELECT id, user_id, server_id, username, home_dir, quota_mb, is_active, created_at FROM ftp_accounts`
	if role != "admin" {
		args = append(args, userID)
		query += ` WHERE user_id = $1`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []models.FTPAccount
	for rows.Next() {
		var a models.FTPAccount
		if err := rows.Scan(&a.ID, &a.UserID, &a.ServerID, &a.Username, &a.HomeDir, &a.QuotaMB, &a.IsActive, &a.CreatedAt); err != nil {
			continue
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (s *FTPService) Delete(ctx context.Context, id string, userID uuid.UUID, role string) error {
	var acc models.FTPAccount
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, server_id, username FROM ftp_accounts WHERE id = $1`, id,
	).Scan(&acc.ID, &acc.UserID, &acc.ServerID, &acc.Username)
	if err != nil {
		return fmt.Errorf("FTP account not found")
	}
	if role != "admin" && acc.UserID != userID {
		return fmt.Errorf("FTP account not found")
	}

	go s.deprovisionFTP(acc.ServerID.String(), acc.Username)

	_, err = s.pool.Exec(ctx, `DELETE FROM ftp_accounts WHERE id = $1`, id)
	return err
}

func (s *FTPService) deprovisionFTP(serverID, username string) {
	ctx := context.Background()
	srv, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		return
	}
	script := fmt.Sprintf(`
userdel -r "ftp_%s" 2>/dev/null || true
sed -i '/^ftp_%s$/d' /etc/vsftpd.chroot_list 2>/dev/null || true
systemctl restart vsftpd 2>/dev/null || true
echo "FTP_DELETED"
`, username, username)
	provisioner.RunScript(srv, script)
}
