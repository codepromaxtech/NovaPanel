package services

import (
	"context"
	"fmt"
	"log"
	"strings"

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
	return GetServerInfo(ctx, s.pool, serverID)
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

func (s *FTPService) List(ctx context.Context, userID uuid.UUID, role, serverID string) ([]models.FTPAccount, error) {
	var args []interface{}
	conditions := []string{}
	query := `SELECT id, user_id, server_id, username, home_dir, quota_mb, is_active, created_at FROM ftp_accounts`

	if role != "admin" {
		args = append(args, userID)
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", len(args)))
	}
	if serverID != "" && serverID != "-" {
		args = append(args, serverID)
		conditions = append(conditions, fmt.Sprintf("server_id = $%d", len(args)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for _, c := range conditions[1:] {
			query += " AND " + c
		}
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

// ─── SFTP SSH Key Management ──────────────────────────────────────────────────

type SFTPKey struct {
	ID           string `json:"id"`
	FTPAccountID string `json:"ftp_account_id"`
	Label        string `json:"label"`
	Fingerprint  string `json:"fingerprint"`
	CreatedAt    string `json:"created_at"`
}

type AddSFTPKeyRequest struct {
	Label     string `json:"label"`
	PublicKey string `json:"public_key" binding:"required"`
}

func (s *FTPService) ListSFTPKeys(ctx context.Context, ftpAccountID, serverID string, userID uuid.UUID, role string) ([]SFTPKey, error) {
	if err := s.checkFTPOwner(ctx, ftpAccountID, serverID, userID, role); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, ftp_account_id, label, fingerprint, created_at FROM sftp_keys WHERE ftp_account_id = $1 ORDER BY created_at DESC`,
		ftpAccountID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []SFTPKey
	for rows.Next() {
		var k SFTPKey
		rows.Scan(&k.ID, &k.FTPAccountID, &k.Label, &k.Fingerprint, &k.CreatedAt)
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *FTPService) AddSFTPKey(ctx context.Context, ftpAccountID, serverID string, userID uuid.UUID, role string, req AddSFTPKeyRequest) (*SFTPKey, error) {
	if err := s.checkFTPOwner(ctx, ftpAccountID, serverID, userID, role); err != nil {
		return nil, err
	}

	// Derive fingerprint via ssh-keygen
	fingerprint, err := computeSSHFingerprint(req.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid SSH public key: %w", err)
	}

	label := req.Label
	if label == "" {
		label = "key"
	}

	key := &SFTPKey{}
	err = s.pool.QueryRow(ctx,
		`INSERT INTO sftp_keys (ftp_account_id, label, public_key, fingerprint)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, ftp_account_id, label, fingerprint, created_at`,
		ftpAccountID, label, req.PublicKey, fingerprint,
	).Scan(&key.ID, &key.FTPAccountID, &key.Label, &key.Fingerprint, &key.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to store key: %w", err)
	}

	// Provision on server
	go s.provisionSFTPKey(serverID, ftpAccountID, req.PublicKey)
	return key, nil
}

func (s *FTPService) DeleteSFTPKey(ctx context.Context, keyID, ftpAccountID, serverID string, userID uuid.UUID, role string) error {
	if err := s.checkFTPOwner(ctx, ftpAccountID, serverID, userID, role); err != nil {
		return err
	}

	var pubKey string
	err := s.pool.QueryRow(ctx,
		`SELECT public_key FROM sftp_keys WHERE id = $1 AND ftp_account_id = $2`, keyID, ftpAccountID,
	).Scan(&pubKey)
	if err != nil {
		return fmt.Errorf("key not found")
	}

	_, err = s.pool.Exec(ctx, `DELETE FROM sftp_keys WHERE id = $1`, keyID)
	if err != nil {
		return err
	}

	go s.deprovisionSFTPKey(serverID, ftpAccountID, pubKey)
	return nil
}

func (s *FTPService) checkFTPOwner(ctx context.Context, ftpAccountID, serverID string, userID uuid.UUID, role string) error {
	var ownerID uuid.UUID
	err := s.pool.QueryRow(ctx,
		`SELECT user_id FROM ftp_accounts WHERE id = $1 AND server_id = $2`, ftpAccountID, serverID,
	).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("FTP account not found")
	}
	if role != "admin" && ownerID != userID {
		return fmt.Errorf("FTP account not found")
	}
	return nil
}

func (s *FTPService) provisionSFTPKey(serverID, ftpAccountID, pubKey string) {
	ctx := context.Background()
	// Get username from DB
	var username string
	s.pool.QueryRow(ctx, `SELECT username FROM ftp_accounts WHERE id = $1`, ftpAccountID).Scan(&username)
	if username == "" {
		return
	}
	srv, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		log.Printf("SFTP key provision SSH error: %v", err)
		return
	}
	script := fmt.Sprintf(`
set -e
SSHDIR="/home/ftp_%s/.ssh"
mkdir -p "$SSHDIR"
chmod 700 "$SSHDIR"
AK="$SSHDIR/authorized_keys"
touch "$AK"
grep -qxF '%s' "$AK" || echo '%s' >> "$AK"
chmod 600 "$AK"
chown -R "ftp_%s": "$SSHDIR"
echo "SFTP_KEY_ADDED"
`, username, pubKey, pubKey, username)
	out, err := provisioner.RunScript(srv, script)
	if err != nil {
		log.Printf("SFTP key provision error: %v — %s", err, out)
	}
}

func (s *FTPService) deprovisionSFTPKey(serverID, ftpAccountID, pubKey string) {
	ctx := context.Background()
	var username string
	s.pool.QueryRow(ctx, `SELECT username FROM ftp_accounts WHERE id = $1`, ftpAccountID).Scan(&username)
	if username == "" {
		return
	}
	srv, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		return
	}
	escapedKey := strings.ReplaceAll(pubKey, "/", "\\/")
	script := fmt.Sprintf(`
AK="/home/ftp_%s/.ssh/authorized_keys"
[ -f "$AK" ] && sed -i '/%s/d' "$AK" || true
echo "SFTP_KEY_REMOVED"
`, username, escapedKey[:min(len(escapedKey), 30)])
	provisioner.RunScript(srv, script)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func computeSSHFingerprint(pubKey string) (string, error) {
	fields := strings.Fields(strings.TrimSpace(pubKey))
	if len(fields) < 2 {
		return "", fmt.Errorf("invalid public key format")
	}
	// Return truncated key type + first 16 chars of key data as fingerprint
	return fmt.Sprintf("%s:SHA256:%s…", fields[0], fields[1][:min(len(fields[1]), 16)]), nil
}
