package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
)

type AlertService struct {
	pool     *pgxpool.Pool
	smtpSvc  *SMTPService
}

func NewAlertService(pool *pgxpool.Pool, smtpSvc *SMTPService) *AlertService {
	return &AlertService{pool: pool, smtpSvc: smtpSvc}
}

func (s *AlertService) CreateRule(ctx context.Context, userID uuid.UUID, req models.AlertRule) (*models.AlertRule, error) {
	r := &models.AlertRule{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO alert_rules (user_id, server_id, name, metric, threshold, operator, duration_min, channel, destination)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, user_id, server_id, name, metric, threshold, operator, duration_min, channel, destination, is_active, created_at`,
		userID, req.ServerID, req.Name, req.Metric, req.Threshold, req.Operator,
		req.DurationMin, req.Channel, req.Destination,
	).Scan(&r.ID, &r.UserID, &r.ServerID, &r.Name, &r.Metric, &r.Threshold, &r.Operator,
		&r.DurationMin, &r.Channel, &r.Destination, &r.IsActive, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create alert rule: %w", err)
	}
	return r, nil
}

func (s *AlertService) ListRules(ctx context.Context, userID uuid.UUID, role string) ([]models.AlertRule, error) {
	var args []interface{}
	query := `SELECT id, user_id, server_id, name, metric, threshold, operator, duration_min, channel, destination, is_active, created_at
	          FROM alert_rules`
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

	var rules []models.AlertRule
	for rows.Next() {
		var r models.AlertRule
		if err := rows.Scan(&r.ID, &r.UserID, &r.ServerID, &r.Name, &r.Metric, &r.Threshold,
			&r.Operator, &r.DurationMin, &r.Channel, &r.Destination, &r.IsActive, &r.CreatedAt); err != nil {
			continue
		}
		rules = append(rules, r)
	}
	return rules, nil
}

func (s *AlertService) UpdateRule(ctx context.Context, id string, userID uuid.UUID, role string, req models.AlertRule) (*models.AlertRule, error) {
	var ownerID uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT user_id FROM alert_rules WHERE id = $1`, id).Scan(&ownerID)
	if err != nil {
		return nil, fmt.Errorf("rule not found")
	}
	if role != "admin" && ownerID != userID {
		return nil, fmt.Errorf("rule not found")
	}

	r := &models.AlertRule{}
	err = s.pool.QueryRow(ctx,
		`UPDATE alert_rules SET name=$2, metric=$3, threshold=$4, operator=$5,
		        duration_min=$6, channel=$7, destination=$8, is_active=$9
		 WHERE id=$1
		 RETURNING id, user_id, server_id, name, metric, threshold, operator, duration_min, channel, destination, is_active, created_at`,
		id, req.Name, req.Metric, req.Threshold, req.Operator,
		req.DurationMin, req.Channel, req.Destination, req.IsActive,
	).Scan(&r.ID, &r.UserID, &r.ServerID, &r.Name, &r.Metric, &r.Threshold, &r.Operator,
		&r.DurationMin, &r.Channel, &r.Destination, &r.IsActive, &r.CreatedAt)
	return r, err
}

func (s *AlertService) DeleteRule(ctx context.Context, id string, userID uuid.UUID, role string) error {
	var ownerID uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT user_id FROM alert_rules WHERE id = $1`, id).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("rule not found")
	}
	if role != "admin" && ownerID != userID {
		return fmt.Errorf("rule not found")
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM alert_rules WHERE id = $1`, id)
	return err
}

func (s *AlertService) ListIncidents(ctx context.Context, userID uuid.UUID, role string) ([]models.AlertIncident, error) {
	var args []interface{}
	query := `SELECT ai.id, ai.rule_id, ai.fired_at, ai.resolved_at, ai.value, ai.notified
	          FROM alert_incidents ai
	          JOIN alert_rules ar ON ar.id = ai.rule_id`
	if role != "admin" {
		args = append(args, userID)
		query += ` WHERE ar.user_id = $1`
	}
	query += ` ORDER BY ai.fired_at DESC LIMIT 200`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []models.AlertIncident
	for rows.Next() {
		var i models.AlertIncident
		if err := rows.Scan(&i.ID, &i.RuleID, &i.FiredAt, &i.ResolvedAt, &i.Value, &i.Notified); err != nil {
			continue
		}
		incidents = append(incidents, i)
	}
	return incidents, nil
}

// EvaluateRules checks all active rules against the latest server metrics.
// Called every minute by the background goroutine in main.go.
func (s *AlertService) EvaluateRules(ctx context.Context) {
	rows, err := s.pool.Query(ctx,
		`SELECT ar.id, ar.server_id, ar.name, ar.metric, ar.threshold, ar.operator,
		        ar.channel, ar.destination, ar.user_id,
		        sm.cpu_percent, sm.ram_used_mb, sm.ram_total_mb,
		        sm.disk_used_gb, sm.disk_total_gb, sm.load_avg_1
		 FROM alert_rules ar
		 JOIN LATERAL (
		   SELECT cpu_percent, ram_used_mb, ram_total_mb, disk_used_gb, disk_total_gb, load_avg_1
		   FROM server_metrics
		   WHERE server_id = ar.server_id
		   ORDER BY recorded_at DESC LIMIT 1
		 ) sm ON true
		 WHERE ar.is_active = true AND ar.server_id IS NOT NULL`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var ruleID, serverID, name, metric, operator, channel, dest string
		var threshold, cpu, ramUsed, ramTotal, diskUsed, diskTotal, load float64
		var userID uuid.UUID

		if err := rows.Scan(&ruleID, &serverID, &name, &metric, &threshold, &operator,
			&channel, &dest, &userID,
			&cpu, &ramUsed, &ramTotal, &diskUsed, &diskTotal, &load); err != nil {
			continue
		}

		var value float64
		switch metric {
		case "cpu":
			value = cpu
		case "memory":
			if ramTotal > 0 {
				value = (ramUsed / ramTotal) * 100
			}
		case "disk":
			if diskTotal > 0 {
				value = (diskUsed / diskTotal) * 100
			}
		case "load":
			value = load
		default:
			continue
		}

		breached := compare(value, operator, threshold)

		// Check for open incident
		var openIncidentID string
		s.pool.QueryRow(ctx,
			`SELECT id FROM alert_incidents WHERE rule_id = $1 AND resolved_at IS NULL LIMIT 1`,
			ruleID,
		).Scan(&openIncidentID)

		if breached && openIncidentID == "" {
			// Fire new incident
			var incidentID string
			s.pool.QueryRow(ctx,
				`INSERT INTO alert_incidents (rule_id, value) VALUES ($1, $2) RETURNING id`,
				ruleID, value,
			).Scan(&incidentID)
			go s.notify(channel, dest, name, metric, value, threshold, operator, userID)
		} else if !breached && openIncidentID != "" {
			// Resolve open incident
			s.pool.Exec(ctx,
				`UPDATE alert_incidents SET resolved_at = NOW() WHERE id = $1`, openIncidentID)
		}
	}
}

func compare(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	}
	return false
}

func (s *AlertService) notify(channel, dest, ruleName, metric string, value, threshold float64, operator string, userID uuid.UUID) {
	subject := fmt.Sprintf("NovaPanel Alert: %s", ruleName)
	body := fmt.Sprintf("Alert rule '%s' fired:\nMetric: %s\nValue: %.2f %s %.2f\nTime: %s",
		ruleName, metric, value, operator, threshold, time.Now().Format(time.RFC3339))

	switch channel {
	case "email":
		if dest == "" {
			// Fetch user email as fallback
			s.pool.QueryRow(context.Background(),
				`SELECT email FROM users WHERE id = $1`, userID).Scan(&dest)
		}
		if dest != "" && s.smtpSvc != nil {
			if err := s.smtpSvc.SendAlertNotification(dest, subject, body); err != nil {
				log.Printf("alert notify email error: %v", err)
			}
		}
	case "webhook", "slack":
		if dest != "" {
			payload := fmt.Sprintf(`{"rule":"%s","metric":"%s","value":%.2f,"threshold":%.2f,"operator":"%s"}`,
				ruleName, metric, value, threshold, operator)
			req, err := http.NewRequest(http.MethodPost, dest, strings.NewReader(payload))
			if err == nil {
				req.Header.Set("Content-Type", "application/json")
				http.DefaultClient.Do(req)
			}
		}
	}
}
