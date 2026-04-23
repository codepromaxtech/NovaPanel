package services

import (
	"context"
	"fmt"
	"log"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
	novacrypto "github.com/novapanel/novapanel/internal/crypto"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/provisioner"
)

type SecurityService struct {
	pool      *pgxpool.Pool
	cryptoKey []byte
}

func NewSecurityService(pool *pgxpool.Pool, encryptionKey string) *SecurityService {
	return &SecurityService{pool: pool, cryptoKey: novacrypto.DeriveKey(encryptionKey)}
}

func (s *SecurityService) getServerSSH(ctx context.Context, serverID string) (provisioner.ServerInfo, error) {
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

func (s *SecurityService) CreateRule(ctx context.Context, req models.CreateFirewallRuleRequest) (*models.FirewallRule, error) {
	direction := req.Direction
	if direction == "" {
		direction = "in"
	}
	protocol := req.Protocol
	if protocol == "" {
		protocol = "tcp"
	}
	sourceIP := req.SourceIP
	if sourceIP == "" {
		sourceIP = "any"
	}
	action := req.Action
	if action == "" {
		action = "allow"
	}

	r := &models.FirewallRule{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO firewall_rules (server_id, direction, protocol, port, source_ip, action, description)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, server_id, direction, protocol, port, source_ip, action, description, is_active, created_at`,
		req.ServerID, direction, protocol, req.Port, sourceIP, action, req.Description,
	).Scan(&r.ID, &r.ServerID, &r.Direction, &r.Protocol, &r.Port, &r.SourceIP, &r.Action, &r.Description, &r.IsActive, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create firewall rule: %w", err)
	}

	// Apply rule via SSH asynchronously
	go s.applyUFWRule(r)
	return r, nil
}

func (s *SecurityService) applyUFWRule(r *models.FirewallRule) {
	ctx := context.Background()
	srv, err := s.getServerSSH(ctx, r.ServerID.String())
	if err != nil {
		log.Printf("firewall: SSH error for server %s: %v", r.ServerID, err)
		return
	}

	fromClause := ""
	if r.SourceIP != "any" {
		fromClause = fmt.Sprintf("from %s ", r.SourceIP)
	}
	portClause := ""
	if r.Port != "" && r.Port != "any" {
		portClause = fmt.Sprintf("to any port %s proto %s", r.Port, r.Protocol)
	}

	script := fmt.Sprintf(`
set -e
which ufw || apt-get install -y ufw
ufw --force enable 2>/dev/null || true
ufw default deny incoming 2>/dev/null || true
ufw default allow outgoing 2>/dev/null || true
ufw allow 22/tcp 2>/dev/null || true
ufw %s %s%s comment "np-%s" 2>/dev/null || true
echo "UFW_APPLIED"
`, r.Action, fromClause, portClause, r.ID)

	out, err := provisioner.RunScript(srv, script)
	if err != nil {
		log.Printf("firewall apply error: %v — %s", err, out)
	}
}

func (s *SecurityService) ListRules(ctx context.Context, serverID string) ([]models.FirewallRule, error) {
	var args []interface{}
	query := `SELECT id, server_id, direction, protocol, port, source_ip, action, description, is_active, created_at
	          FROM firewall_rules`
	if serverID != "" {
		args = append(args, serverID)
		query += ` WHERE server_id = $1`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []models.FirewallRule
	for rows.Next() {
		var r models.FirewallRule
		if err := rows.Scan(&r.ID, &r.ServerID, &r.Direction, &r.Protocol, &r.Port, &r.SourceIP, &r.Action, &r.Description, &r.IsActive, &r.CreatedAt); err != nil {
			continue
		}
		rules = append(rules, r)
	}
	return rules, nil
}

func (s *SecurityService) DeleteRule(ctx context.Context, id string) error {
	// Fetch rule details before deleting
	var r models.FirewallRule
	err := s.pool.QueryRow(ctx,
		`SELECT id, server_id, direction, protocol, port, source_ip, action FROM firewall_rules WHERE id = $1`, id,
	).Scan(&r.ID, &r.ServerID, &r.Direction, &r.Protocol, &r.Port, &r.SourceIP, &r.Action)
	if err == nil {
		go s.removeUFWRule(&r)
	}

	_, err = s.pool.Exec(ctx, `DELETE FROM firewall_rules WHERE id = $1`, id)
	return err
}

func (s *SecurityService) removeUFWRule(r *models.FirewallRule) {
	ctx := context.Background()
	srv, err := s.getServerSSH(ctx, r.ServerID.String())
	if err != nil {
		return
	}

	fromClause := ""
	if r.SourceIP != "any" {
		fromClause = fmt.Sprintf("from %s ", r.SourceIP)
	}
	portClause := ""
	if r.Port != "" && r.Port != "any" {
		portClause = fmt.Sprintf("to any port %s proto %s", r.Port, r.Protocol)
	}

	script := fmt.Sprintf(`ufw delete %s %s%s 2>/dev/null || true && echo "UFW_DELETED"`,
		r.Action, fromClause, portClause)
	provisioner.RunScript(srv, script)
}

func (s *SecurityService) ListEvents(ctx context.Context, page, perPage int) (*models.PaginatedResponse, error) {
	offset := (page - 1) * perPage
	var total int64
	s.pool.QueryRow(ctx, `SELECT count(*) FROM security_events`).Scan(&total)

	rows, err := s.pool.Query(ctx,
		`SELECT id, server_id, event_type, source_ip, details, severity, created_at
		 FROM security_events ORDER BY created_at DESC LIMIT $1 OFFSET $2`, perPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.SecurityEvent
	for rows.Next() {
		var e models.SecurityEvent
		if err := rows.Scan(&e.ID, &e.ServerID, &e.EventType, &e.SourceIP, &e.Details, &e.Severity, &e.CreatedAt); err != nil {
			continue
		}
		events = append(events, e)
	}

	return &models.PaginatedResponse{
		Data: events, Total: total, Page: page, PerPage: perPage,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}, nil
}
