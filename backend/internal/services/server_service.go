package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/provisioner"
	"golang.org/x/crypto/ssh"
)

type ServerService struct {
	db *pgxpool.Pool
}

func NewServerService(db *pgxpool.Pool) *ServerService {
	return &ServerService{db: db}
}

func (s *ServerService) Create(ctx context.Context, req models.CreateServerRequest) (*models.Server, error) {
	port := 22
	if req.Port > 0 {
		port = req.Port
	}
	role := "worker"
	if req.Role != "" {
		role = req.Role
	}

	server := &models.Server{}
	sshUser := "root"
	if req.SSHUser != "" {
		sshUser = req.SSHUser
	}
	authMethod := "key"
	if req.AuthMethod != "" {
		authMethod = req.AuthMethod
	}
	err := s.db.QueryRow(ctx,
		`INSERT INTO servers (name, hostname, ip_address, port, os, role, status, agent_status, ssh_user, ssh_key, ssh_password, auth_method)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING id, name, hostname, host(ip_address), port, os, role, status, agent_version, agent_status, ssh_user, ssh_key, ssh_password, auth_method, created_at, updated_at`,
		req.Name, req.Hostname, req.IPAddress, port, req.OS, role, "pending", "disconnected", sshUser, req.SSHKey, req.SSHPassword, authMethod,
	).Scan(&server.ID, &server.Name, &server.Hostname, &server.IPAddress, &server.Port,
		&server.OS, &server.Role, &server.Status, &server.AgentVersion, &server.AgentStatus,
		&server.SSHUser, &server.SSHKey, &server.SSHPassword, &server.AuthMethod,
		&server.CreatedAt, &server.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Auto-setup SSH key pair if created with password auth
	if authMethod == "password" && req.SSHPassword != "" {
		go func() {
			if err := s.SetupSSHKey(context.Background(), server.ID.String()); err != nil {
				log.Printf("⚠ Auto SSH key setup failed for server %s: %v", server.Name, err)
			} else {
				log.Printf("🔑 Auto SSH key pair deployed for server %s", server.Name)
			}
		}()
	}

	return server, nil
}

func (s *ServerService) GetByID(ctx context.Context, id string) (*models.Server, error) {
	server := &models.Server{}
	err := s.db.QueryRow(ctx,
		`SELECT id, name, hostname, host(ip_address), port, os, role, status,
		        COALESCE(agent_version, ''), COALESCE(agent_status, 'disconnected'),
		        COALESCE(ssh_key, ''), last_heartbeat, created_at, updated_at
		 FROM servers WHERE id = $1`, id,
	).Scan(&server.ID, &server.Name, &server.Hostname, &server.IPAddress, &server.Port,
		&server.OS, &server.Role, &server.Status, &server.AgentVersion, &server.AgentStatus,
		&server.SSHKey, &server.LastHeartbeat, &server.CreatedAt, &server.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("server not found")
	}
	return server, nil
}

func (s *ServerService) List(ctx context.Context, page, perPage int) (*models.PaginatedResponse, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var total int64
	s.db.QueryRow(ctx, "SELECT COUNT(*) FROM servers").Scan(&total)

	rows, err := s.db.Query(ctx,
		`SELECT id, name, hostname, host(ip_address), port, os, role, status,
		        COALESCE(agent_version, ''), COALESCE(agent_status, 'disconnected'),
		        last_heartbeat, created_at, updated_at
		 FROM servers ORDER BY created_at DESC LIMIT $1 OFFSET $2`, perPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	servers := []models.Server{}
	for rows.Next() {
		var srv models.Server
		err := rows.Scan(&srv.ID, &srv.Name, &srv.Hostname, &srv.IPAddress, &srv.Port,
			&srv.OS, &srv.Role, &srv.Status, &srv.AgentVersion, &srv.AgentStatus,
			&srv.LastHeartbeat, &srv.CreatedAt, &srv.UpdatedAt)
		if err != nil {
			return nil, err
		}
		servers = append(servers, srv)
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	return &models.PaginatedResponse{
		Data:       servers,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

func (s *ServerService) UpdateHeartbeat(ctx context.Context, serverID string, metrics models.ServerMetrics) error {
	now := time.Now()
	sid, err := uuid.Parse(serverID)
	if err != nil {
		return fmt.Errorf("invalid server ID")
	}

	// Update server status
	_, err = s.db.Exec(ctx,
		`UPDATE servers SET agent_status = 'connected', last_heartbeat = $1, status = 'active', updated_at = $1 WHERE id = $2`,
		now, sid)
	if err != nil {
		return err
	}

	// Insert metrics
	_, err = s.db.Exec(ctx,
		`INSERT INTO server_metrics (server_id, cpu_percent, ram_used_mb, ram_total_mb, disk_used_gb, disk_total_gb,
		 load_avg_1, load_avg_5, load_avg_15, network_rx_bytes, network_tx_bytes, recorded_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		sid, metrics.CPUPercent, metrics.RAMUsedMB, metrics.RAMTotalMB,
		metrics.DiskUsedGB, metrics.DiskTotalGB, metrics.LoadAvg1, metrics.LoadAvg5,
		metrics.LoadAvg15, metrics.NetworkRxBytes, metrics.NetworkTxBytes, now)
	return err
}

func (s *ServerService) GetDashboardStats(ctx context.Context) (*models.DashboardStats, error) {
	stats := &models.DashboardStats{}

	s.db.QueryRow(ctx, "SELECT COUNT(*) FROM servers").Scan(&stats.TotalServers)
	s.db.QueryRow(ctx, "SELECT COUNT(*) FROM servers WHERE status = 'active'").Scan(&stats.ActiveServers)
	s.db.QueryRow(ctx, "SELECT COUNT(*) FROM domains").Scan(&stats.TotalDomains)
	s.db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	s.db.QueryRow(ctx, "SELECT COUNT(*) FROM applications").Scan(&stats.TotalApps)
	s.db.QueryRow(ctx, "SELECT COUNT(*) FROM tasks WHERE status IN ('queued', 'running')").Scan(&stats.PendingTasks)

	return stats, nil
}

func (s *ServerService) Delete(ctx context.Context, id string) error {
	result, err := s.db.Exec(ctx, "DELETE FROM servers WHERE id = $1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("server not found")
	}
	return nil
}

// TestConnection validates SSH credentials by connecting and running hostname
func (s *ServerService) TestConnection(ctx context.Context, req models.TestConnectionRequest) (string, error) {
	port := 22
	if req.Port > 0 {
		port = req.Port
	}
	sshUser := "root"
	if req.SSHUser != "" {
		sshUser = req.SSHUser
	}
	authMethod := "password"
	if req.AuthMethod != "" {
		authMethod = req.AuthMethod
	}

	server := provisioner.ServerInfo{
		IPAddress:   req.IPAddress,
		Port:        port,
		SSHUser:     sshUser,
		SSHKey:      req.SSHKey,
		SSHPassword: req.SSHPassword,
		AuthMethod:  authMethod,
	}

	output, err := provisioner.RunScriptAsUser(server, "hostname && echo '---' && uname -a && echo '---' && uptime")
	if err != nil {
		return "", fmt.Errorf("SSH connection failed: %w", err)
	}
	return strings.TrimSpace(output), nil
}

// Update edits an existing server. Only non-zero/non-empty fields are updated.
func (s *ServerService) Update(ctx context.Context, id string, req models.UpdateServerRequest) (*models.Server, error) {
	sid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid server ID")
	}

	// Build dynamic SET clause
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Name != "" {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, req.Name)
		argIdx++
	}
	if req.Hostname != "" {
		setClauses = append(setClauses, fmt.Sprintf("hostname = $%d", argIdx))
		args = append(args, req.Hostname)
		argIdx++
	}
	if req.IPAddress != "" {
		setClauses = append(setClauses, fmt.Sprintf("ip_address = $%d", argIdx))
		args = append(args, req.IPAddress)
		argIdx++
	}
	if req.Port > 0 {
		setClauses = append(setClauses, fmt.Sprintf("port = $%d", argIdx))
		args = append(args, req.Port)
		argIdx++
	}
	if req.OS != "" {
		setClauses = append(setClauses, fmt.Sprintf("os = $%d", argIdx))
		args = append(args, req.OS)
		argIdx++
	}
	if req.Role != "" {
		setClauses = append(setClauses, fmt.Sprintf("role = $%d", argIdx))
		args = append(args, req.Role)
		argIdx++
	}
	if req.SSHUser != "" {
		setClauses = append(setClauses, fmt.Sprintf("ssh_user = $%d", argIdx))
		args = append(args, req.SSHUser)
		argIdx++
	}
	if req.SSHKey != "" {
		setClauses = append(setClauses, fmt.Sprintf("ssh_key = $%d", argIdx))
		args = append(args, req.SSHKey)
		argIdx++
	}
	if req.SSHPassword != "" {
		setClauses = append(setClauses, fmt.Sprintf("ssh_password = $%d", argIdx))
		args = append(args, req.SSHPassword)
		argIdx++
	}
	if req.AuthMethod != "" {
		setClauses = append(setClauses, fmt.Sprintf("auth_method = $%d", argIdx))
		args = append(args, req.AuthMethod)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now())
	argIdx++

	args = append(args, sid)
	query := fmt.Sprintf(
		`UPDATE servers SET %s WHERE id = $%d
		 RETURNING id, name, hostname, host(ip_address), port, os, role, status,
		           COALESCE(agent_version, ''), COALESCE(agent_status, 'disconnected'),
		           ssh_user, COALESCE(ssh_key, ''), COALESCE(ssh_password, ''), COALESCE(auth_method, 'password'),
		           created_at, updated_at`,
		strings.Join(setClauses, ", "), argIdx,
	)

	server := &models.Server{}
	err = s.db.QueryRow(ctx, query, args...).Scan(
		&server.ID, &server.Name, &server.Hostname, &server.IPAddress, &server.Port,
		&server.OS, &server.Role, &server.Status, &server.AgentVersion, &server.AgentStatus,
		&server.SSHUser, &server.SSHKey, &server.SSHPassword, &server.AuthMethod,
		&server.CreatedAt, &server.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update server: %w", err)
	}
	return server, nil
}

// GetLatestMetrics returns the most recent metrics row for a server
func (s *ServerService) GetLatestMetrics(ctx context.Context, serverID string) (*models.ServerMetrics, error) {
	sid, err := uuid.Parse(serverID)
	if err != nil {
		return nil, fmt.Errorf("invalid server ID")
	}

	m := &models.ServerMetrics{}
	err = s.db.QueryRow(ctx,
		`SELECT id, server_id, cpu_percent, ram_used_mb, ram_total_mb, disk_used_gb, disk_total_gb,
		        load_avg_1, load_avg_5, load_avg_15, network_rx_bytes, network_tx_bytes, recorded_at
		 FROM server_metrics WHERE server_id = $1 ORDER BY recorded_at DESC LIMIT 1`, sid,
	).Scan(&m.ID, &m.ServerID, &m.CPUPercent, &m.RAMUsedMB, &m.RAMTotalMB,
		&m.DiskUsedGB, &m.DiskTotalGB, &m.LoadAvg1, &m.LoadAvg5, &m.LoadAvg15,
		&m.NetworkRxBytes, &m.NetworkTxBytes, &m.RecordedAt)
	if err != nil {
		return nil, fmt.Errorf("no metrics available")
	}
	return m, nil
}

// SetupSSHKey generates an RSA key pair, deploys the public key to the server
// via the existing password connection, then updates the DB to use key-based auth.
func (s *ServerService) SetupSSHKey(ctx context.Context, serverID string) error {
	// 1. Fetch current server details (need password + connection info)
	var ipAddress, sshUser, sshPassword, authMethod string
	var port int
	err := s.db.QueryRow(ctx,
		`SELECT host(ip_address), port, ssh_user, COALESCE(ssh_password, ''), COALESCE(auth_method, 'password')
		 FROM servers WHERE id = $1`, serverID,
	).Scan(&ipAddress, &port, &sshUser, &sshPassword, &authMethod)
	if err != nil {
		return fmt.Errorf("server not found: %w", err)
	}

	// Skip if already using key auth
	if authMethod == "key" {
		return nil
	}
	if sshPassword == "" {
		return fmt.Errorf("no password available for initial connection")
	}

	// 2. Generate RSA-4096 key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Encode private key to PEM
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Generate public key in OpenSSH format
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to create SSH public key: %w", err)
	}
	pubKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub))) + " novapanel-auto"

	// 3. Connect via password and deploy the public key
	server := provisioner.ServerInfo{
		IPAddress:   ipAddress,
		Port:        port,
		SSHUser:     sshUser,
		SSHPassword: sshPassword,
		AuthMethod:  "password",
	}

	deployScript := fmt.Sprintf(`
mkdir -p ~/.ssh && chmod 700 ~/.ssh
touch ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys
if ! grep -q "novapanel-auto" ~/.ssh/authorized_keys 2>/dev/null; then
    echo '%s' >> ~/.ssh/authorized_keys
    echo 'Key deployed successfully'
else
    echo 'NovaPanel key already exists'
fi
`, pubKeyStr)

	output, err := provisioner.RunScriptAsUser(server, deployScript)
	if err != nil {
		return fmt.Errorf("failed to deploy public key: %w (output: %s)", err, output)
	}
	log.Printf("SSH key deploy output for %s: %s", ipAddress, strings.TrimSpace(output))

	// 4. Verify the key works by connecting with it
	keyServer := provisioner.ServerInfo{
		IPAddress:  ipAddress,
		Port:       port,
		SSHUser:    sshUser,
		SSHKey:     string(privPEM),
		AuthMethod: "key",
	}
	verifyOutput, err := provisioner.RunScriptAsUser(keyServer, "echo 'key-auth-ok'")
	if err != nil {
		return fmt.Errorf("key verification failed (key deployed but cannot connect): %w", err)
	}
	if !strings.Contains(verifyOutput, "key-auth-ok") {
		return fmt.Errorf("key verification output unexpected: %s", verifyOutput)
	}

	// 5. Update database: store private key, switch auth method, clear password
	_, err = s.db.Exec(ctx,
		`UPDATE servers SET ssh_key = $1, auth_method = 'key', ssh_password = '', updated_at = $2 WHERE id = $3`,
		string(privPEM), time.Now(), serverID,
	)
	if err != nil {
		return fmt.Errorf("failed to update server auth: %w", err)
	}

	return nil
}
