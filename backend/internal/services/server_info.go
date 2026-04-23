package services

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	novacrypto "github.com/novapanel/novapanel/internal/crypto"
	"github.com/novapanel/novapanel/internal/provisioner"
)

// GetServerInfo loads full connection details for a server, including SSH,
// local-execution, and Cloudflare Access fields. All encrypted values are
// decrypted before being returned in the ServerInfo struct.
//
// This is the single source of truth for server connectivity; all service
// getServer() helpers delegate here so CF support is automatic everywhere.
func GetServerInfo(ctx context.Context, pool *pgxpool.Pool, serverID string) (provisioner.ServerInfo, error) {
	var s provisioner.ServerInfo
	var port int
	var encSSHKey, encSSHPassword, encCFClientID, encCFClientSecret string

	err := pool.QueryRow(ctx, `
		SELECT host(ip_address), port, ssh_user,
		       COALESCE(ssh_key, ''), COALESCE(ssh_password, ''), COALESCE(auth_method, 'password'),
		       COALESCE(is_local, FALSE),
		       COALESCE(connect_type, 'ssh'), COALESCE(cf_hostname, ''),
		       COALESCE(cf_client_id, ''), COALESCE(cf_client_secret, '')
		FROM servers WHERE id = $1`, serverID,
	).Scan(
		&s.IPAddress, &port, &s.SSHUser,
		&encSSHKey, &encSSHPassword, &s.AuthMethod,
		&s.IsLocal,
		&s.ConnectType, &s.CFHostname,
		&encCFClientID, &encCFClientSecret,
	)
	if err != nil {
		return s, err
	}
	s.Port = port

	if cryptoKey, kerr := novacrypto.GetEncryptionKey(); kerr == nil {
		dec := func(enc string) string {
			if enc == "" {
				return ""
			}
			if plain, derr := novacrypto.Decrypt(enc, cryptoKey); derr == nil {
				return plain
			}
			return enc
		}
		s.SSHKey = dec(encSSHKey)
		s.SSHPassword = dec(encSSHPassword)
		s.CFClientID = dec(encCFClientID)
		s.CFClientSecret = dec(encCFClientSecret)
	}
	return s, nil
}
