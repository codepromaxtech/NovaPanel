package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
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
