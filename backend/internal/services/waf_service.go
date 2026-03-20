package services

import (
	"context"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
)

type WAFService struct {
	pool *pgxpool.Pool
}

func NewWAFService(pool *pgxpool.Pool) *WAFService {
	return &WAFService{pool: pool}
}

// ──────────── Config ────────────

func (s *WAFService) GetConfig(ctx context.Context, serverID string) (*models.WAFConfig, error) {
	cfg := &models.WAFConfig{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, server_id, enabled, mode, paranoid_level, allowed_methods, max_request_body,
			crs_enabled, sqli_protection, xss_protection, rfi_protection, lfi_protection, rce_protection,
			scanner_block, created_at, updated_at
		 FROM waf_configs WHERE server_id = $1`, serverID,
	).Scan(&cfg.ID, &cfg.ServerID, &cfg.Enabled, &cfg.Mode, &cfg.ParanoidLevel, &cfg.AllowedMethods,
		&cfg.MaxRequestBody, &cfg.CRSEnabled, &cfg.SQLiProtection, &cfg.XSSProtection,
		&cfg.RFIProtection, &cfg.LFIProtection, &cfg.RCEProtection, &cfg.ScannerBlock,
		&cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		// Create default config if none exists
		cfg = &models.WAFConfig{}
		err = s.pool.QueryRow(ctx,
			`INSERT INTO waf_configs (server_id, enabled, mode, paranoid_level, allowed_methods, max_request_body,
				crs_enabled, sqli_protection, xss_protection, rfi_protection, lfi_protection, rce_protection, scanner_block)
			 VALUES ($1, false, 'detection_only', 1, 'GET HEAD POST PUT DELETE', 13107200,
				true, true, true, true, true, true, true)
			 ON CONFLICT (server_id) DO NOTHING
			 RETURNING id, server_id, enabled, mode, paranoid_level, allowed_methods, max_request_body,
				crs_enabled, sqli_protection, xss_protection, rfi_protection, lfi_protection, rce_protection,
				scanner_block, created_at, updated_at`, serverID,
		).Scan(&cfg.ID, &cfg.ServerID, &cfg.Enabled, &cfg.Mode, &cfg.ParanoidLevel, &cfg.AllowedMethods,
			&cfg.MaxRequestBody, &cfg.CRSEnabled, &cfg.SQLiProtection, &cfg.XSSProtection,
			&cfg.RFIProtection, &cfg.LFIProtection, &cfg.RCEProtection, &cfg.ScannerBlock,
			&cfg.CreatedAt, &cfg.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to get/create WAF config: %w", err)
		}
	}
	return cfg, nil
}

func (s *WAFService) UpdateConfig(ctx context.Context, serverID string, req models.UpdateWAFConfigRequest) (*models.WAFConfig, error) {
	// Ensure config exists first
	_, err := s.GetConfig(ctx, serverID)
	if err != nil {
		return nil, err
	}

	cfg := &models.WAFConfig{}
	err = s.pool.QueryRow(ctx,
		`UPDATE waf_configs SET
			enabled = COALESCE($2, enabled),
			mode = CASE WHEN $3 = '' THEN mode ELSE $3 END,
			paranoid_level = COALESCE($4, paranoid_level),
			allowed_methods = CASE WHEN $5 = '' THEN allowed_methods ELSE $5 END,
			max_request_body = COALESCE($6, max_request_body),
			crs_enabled = COALESCE($7, crs_enabled),
			sqli_protection = COALESCE($8, sqli_protection),
			xss_protection = COALESCE($9, xss_protection),
			rfi_protection = COALESCE($10, rfi_protection),
			lfi_protection = COALESCE($11, lfi_protection),
			rce_protection = COALESCE($12, rce_protection),
			scanner_block = COALESCE($13, scanner_block),
			updated_at = NOW()
		 WHERE server_id = $1
		 RETURNING id, server_id, enabled, mode, paranoid_level, allowed_methods, max_request_body,
			crs_enabled, sqli_protection, xss_protection, rfi_protection, lfi_protection, rce_protection,
			scanner_block, created_at, updated_at`,
		serverID, req.Enabled, req.Mode, req.ParanoidLevel, req.AllowedMethods,
		req.MaxRequestBody, req.CRSEnabled, req.SQLiProtection, req.XSSProtection,
		req.RFIProtection, req.LFIProtection, req.RCEProtection, req.ScannerBlock,
	).Scan(&cfg.ID, &cfg.ServerID, &cfg.Enabled, &cfg.Mode, &cfg.ParanoidLevel, &cfg.AllowedMethods,
		&cfg.MaxRequestBody, &cfg.CRSEnabled, &cfg.SQLiProtection, &cfg.XSSProtection,
		&cfg.RFIProtection, &cfg.LFIProtection, &cfg.RCEProtection, &cfg.ScannerBlock,
		&cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update WAF config: %w", err)
	}
	return cfg, nil
}

// ──────────── Disabled Rules ────────────

func (s *WAFService) ListDisabledRules(ctx context.Context, serverID string) ([]models.WAFRule, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, server_id, rule_id, description, is_disabled, created_at FROM waf_rules WHERE server_id = $1 AND is_disabled = true ORDER BY created_at DESC`,
		serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []models.WAFRule
	for rows.Next() {
		var r models.WAFRule
		if err := rows.Scan(&r.ID, &r.ServerID, &r.RuleID, &r.Description, &r.IsDisabled, &r.CreatedAt); err != nil {
			continue
		}
		rules = append(rules, r)
	}
	return rules, nil
}

func (s *WAFService) DisableRule(ctx context.Context, serverID string, req models.DisableWAFRuleRequest) (*models.WAFRule, error) {
	r := &models.WAFRule{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO waf_rules (server_id, rule_id, description, is_disabled)
		 VALUES ($1, $2, $3, true)
		 RETURNING id, server_id, rule_id, description, is_disabled, created_at`,
		serverID, req.RuleID, req.Description,
	).Scan(&r.ID, &r.ServerID, &r.RuleID, &r.Description, &r.IsDisabled, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to disable rule: %w", err)
	}
	return r, nil
}

func (s *WAFService) EnableRule(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM waf_rules WHERE id = $1`, id)
	return err
}

// ──────────── Whitelist ────────────

func (s *WAFService) ListWhitelist(ctx context.Context, serverID string) ([]models.WAFWhitelist, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, server_id, type, value, reason, created_at FROM waf_whitelist WHERE server_id = $1 ORDER BY created_at DESC`,
		serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.WAFWhitelist
	for rows.Next() {
		var w models.WAFWhitelist
		if err := rows.Scan(&w.ID, &w.ServerID, &w.Type, &w.Value, &w.Reason, &w.CreatedAt); err != nil {
			continue
		}
		items = append(items, w)
	}
	return items, nil
}

func (s *WAFService) AddWhitelist(ctx context.Context, serverID string, req models.CreateWAFWhitelistRequest) (*models.WAFWhitelist, error) {
	w := &models.WAFWhitelist{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO waf_whitelist (server_id, type, value, reason)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, server_id, type, value, reason, created_at`,
		serverID, req.Type, req.Value, req.Reason,
	).Scan(&w.ID, &w.ServerID, &w.Type, &w.Value, &w.Reason, &w.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to add whitelist entry: %w", err)
	}
	return w, nil
}

func (s *WAFService) RemoveWhitelist(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM waf_whitelist WHERE id = $1`, id)
	return err
}

// ──────────── Logs ────────────

func (s *WAFService) ListLogs(ctx context.Context, serverID string, page, perPage int) (*models.PaginatedResponse, error) {
	offset := (page - 1) * perPage
	var total int64
	s.pool.QueryRow(ctx, `SELECT count(*) FROM waf_logs WHERE server_id = $1`, serverID).Scan(&total)

	rows, err := s.pool.Query(ctx,
		`SELECT id, server_id, rule_id, uri, client_ip, message, severity, action, created_at
		 FROM waf_logs WHERE server_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		serverID, perPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.WAFLog
	for rows.Next() {
		var l models.WAFLog
		if err := rows.Scan(&l.ID, &l.ServerID, &l.RuleID, &l.URI, &l.ClientIP, &l.Message, &l.Severity, &l.Action, &l.CreatedAt); err != nil {
			continue
		}
		logs = append(logs, l)
	}

	return &models.PaginatedResponse{
		Data: logs, Total: total, Page: page, PerPage: perPage,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}, nil
}
