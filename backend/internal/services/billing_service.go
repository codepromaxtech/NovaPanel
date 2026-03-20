package services

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
)

type BillingService struct {
	pool *pgxpool.Pool
}

func NewBillingService(pool *pgxpool.Pool) *BillingService {
	return &BillingService{pool: pool}
}

func (s *BillingService) CreatePlan(ctx context.Context, req models.CreateBillingPlanRequest) (*models.BillingPlan, error) {
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}
	interval := req.Interval
	if interval == "" {
		interval = "monthly"
	}

	p := &models.BillingPlan{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO billing_plans (name, price_cents, currency, interval, max_domains, max_databases, max_email, disk_gb)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, name, price_cents, currency, interval, max_domains, max_databases, max_email, disk_gb, is_active, created_at`,
		req.Name, req.PriceCents, currency, interval, req.MaxDomains, req.MaxDatabases, req.MaxEmail, req.DiskGB,
	).Scan(&p.ID, &p.Name, &p.PriceCents, &p.Currency, &p.Interval, &p.MaxDomains, &p.MaxDatabases, &p.MaxEmail, &p.DiskGB, &p.IsActive, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}
	return p, nil
}

func (s *BillingService) ListPlans(ctx context.Context) ([]models.BillingPlan, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, price_cents, currency, interval, max_domains, max_databases, max_email, disk_gb, is_active, created_at
		 FROM billing_plans WHERE is_active = true ORDER BY price_cents ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []models.BillingPlan
	for rows.Next() {
		var p models.BillingPlan
		if err := rows.Scan(&p.ID, &p.Name, &p.PriceCents, &p.Currency, &p.Interval, &p.MaxDomains, &p.MaxDatabases, &p.MaxEmail, &p.DiskGB, &p.IsActive, &p.CreatedAt); err != nil {
			continue
		}
		plans = append(plans, p)
	}
	return plans, nil
}

func (s *BillingService) ListInvoices(ctx context.Context, userID string) ([]models.Invoice, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, plan_id, amount_cents, currency, status, stripe_id, paid_at, due_at, created_at
		 FROM invoices WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []models.Invoice
	for rows.Next() {
		var inv models.Invoice
		if err := rows.Scan(&inv.ID, &inv.UserID, &inv.PlanID, &inv.AmountCents, &inv.Currency, &inv.Status, &inv.StripeID, &inv.PaidAt, &inv.DueAt, &inv.CreatedAt); err != nil {
			continue
		}
		invoices = append(invoices, inv)
	}
	return invoices, nil
}
