package services

import (
	"context"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
)

type SecurityService struct {
	pool *pgxpool.Pool
}

func NewSecurityService(pool *pgxpool.Pool) *SecurityService {
	return &SecurityService{pool: pool}
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
	return r, nil
}

func (s *SecurityService) ListRules(ctx context.Context, serverID string) ([]models.FirewallRule, error) {
	query := `SELECT id, server_id, direction, protocol, port, source_ip, action, description, is_active, created_at FROM firewall_rules`
	if serverID != "" {
		query += fmt.Sprintf(` WHERE server_id = '%s'`, serverID)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, query)
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
	_, err := s.pool.Exec(ctx, `DELETE FROM firewall_rules WHERE id = $1`, id)
	return err
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
