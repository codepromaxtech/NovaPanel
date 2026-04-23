package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type ResellerService struct {
	pool *pgxpool.Pool
}

func NewResellerService(pool *pgxpool.Pool) *ResellerService {
	return &ResellerService{pool: pool}
}

type AllocateClientRequest struct {
	Email        string `json:"email"         binding:"required,email"`
	Password     string `json:"password"      binding:"required,min=8"`
	FirstName    string `json:"first_name"    binding:"required"`
	LastName     string `json:"last_name"`
	MaxDomains   int    `json:"max_domains"`
	MaxDatabases int    `json:"max_databases"`
	MaxEmail     int    `json:"max_email"`
	DiskGB       int    `json:"disk_gb"`
}

type ClientWithQuota struct {
	models.User
	Quota models.ResellerQuota `json:"quota"`
}

func (s *ResellerService) AllocateClient(ctx context.Context, resellerID uuid.UUID, req AllocateClientRequest) (*ClientWithQuota, error) {
	// Check email not taken
	var exists bool
	s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)`, req.Email).Scan(&exists)
	if exists {
		return nil, errors.New("user with this email already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Defaults
	if req.MaxDomains == 0 {
		req.MaxDomains = 5
	}
	if req.MaxDatabases == 0 {
		req.MaxDatabases = 2
	}
	if req.MaxEmail == 0 {
		req.MaxEmail = 10
	}
	if req.DiskGB == 0 {
		req.DiskGB = 5
	}

	user := models.User{}
	err = s.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, first_name, last_name, role, status)
		 VALUES ($1, $2, $3, $4, 'client', 'active')
		 RETURNING id, email, first_name, last_name, role, status, created_at, updated_at`,
		req.Email, string(hash), req.FirstName, req.LastName,
	).Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	quota := models.ResellerQuota{}
	err = s.pool.QueryRow(ctx,
		`INSERT INTO reseller_quotas (reseller_id, client_id, max_domains, max_databases, max_email, disk_gb)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, reseller_id, client_id, max_domains, max_databases, max_email, disk_gb, created_at`,
		resellerID, user.ID, req.MaxDomains, req.MaxDatabases, req.MaxEmail, req.DiskGB,
	).Scan(&quota.ID, &quota.ResellerID, &quota.ClientID, &quota.MaxDomains, &quota.MaxDatabases, &quota.MaxEmail, &quota.DiskGB, &quota.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to set quota: %w", err)
	}

	return &ClientWithQuota{User: user, Quota: quota}, nil
}

func (s *ResellerService) ListClients(ctx context.Context, resellerID uuid.UUID) ([]ClientWithQuota, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT u.id, u.email, u.first_name, u.last_name, u.role, u.status, u.created_at, u.updated_at,
		        rq.id, rq.max_domains, rq.max_databases, rq.max_email, rq.disk_gb, rq.created_at
		 FROM reseller_quotas rq
		 JOIN users u ON u.id = rq.client_id
		 WHERE rq.reseller_id = $1
		 ORDER BY u.created_at DESC`, resellerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []ClientWithQuota
	for rows.Next() {
		var c ClientWithQuota
		if err := rows.Scan(
			&c.User.ID, &c.User.Email, &c.User.FirstName, &c.User.LastName,
			&c.User.Role, &c.User.Status, &c.User.CreatedAt, &c.User.UpdatedAt,
			&c.Quota.ID, &c.Quota.MaxDomains, &c.Quota.MaxDatabases, &c.Quota.MaxEmail,
			&c.Quota.DiskGB, &c.Quota.CreatedAt,
		); err != nil {
			continue
		}
		clients = append(clients, c)
	}
	return clients, nil
}

func (s *ResellerService) UpdateClientQuota(ctx context.Context, resellerID, clientID uuid.UUID, req AllocateClientRequest) (*models.ResellerQuota, error) {
	quota := &models.ResellerQuota{}
	err := s.pool.QueryRow(ctx,
		`UPDATE reseller_quotas SET max_domains=$3, max_databases=$4, max_email=$5, disk_gb=$6
		 WHERE reseller_id=$1 AND client_id=$2
		 RETURNING id, reseller_id, client_id, max_domains, max_databases, max_email, disk_gb, created_at`,
		resellerID, clientID, req.MaxDomains, req.MaxDatabases, req.MaxEmail, req.DiskGB,
	).Scan(&quota.ID, &quota.ResellerID, &quota.ClientID, &quota.MaxDomains, &quota.MaxDatabases, &quota.MaxEmail, &quota.DiskGB, &quota.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("client not found or not your client")
	}
	return quota, nil
}

func (s *ResellerService) SuspendClient(ctx context.Context, resellerID, clientID uuid.UUID) error {
	// Verify ownership
	var count int
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM reseller_quotas WHERE reseller_id=$1 AND client_id=$2`,
		resellerID, clientID).Scan(&count)
	if count == 0 {
		return errors.New("client not found or not your client")
	}
	_, err := s.pool.Exec(ctx, `UPDATE users SET status='suspended' WHERE id=$1`, clientID)
	return err
}

func (s *ResellerService) DeleteClient(ctx context.Context, resellerID, clientID uuid.UUID) error {
	var count int
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM reseller_quotas WHERE reseller_id=$1 AND client_id=$2`,
		resellerID, clientID).Scan(&count)
	if count == 0 {
		return errors.New("client not found or not your client")
	}
	_, err := s.pool.Exec(ctx, `UPDATE users SET status='deleted' WHERE id=$1`, clientID)
	return err
}

func (s *ResellerService) GetClientUsage(ctx context.Context, resellerID, clientID uuid.UUID) (map[string]interface{}, error) {
	var count int
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM reseller_quotas WHERE reseller_id=$1 AND client_id=$2`,
		resellerID, clientID).Scan(&count)
	if count == 0 {
		return nil, errors.New("client not found or not your client")
	}

	usage := map[string]interface{}{}
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM domains WHERE user_id=$1`, clientID).Scan(&count)
	usage["domains"] = count
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM databases WHERE user_id=$1`, clientID).Scan(&count)
	usage["databases"] = count
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM email_accounts WHERE user_id=$1`, clientID).Scan(&count)
	usage["email_accounts"] = count
	return usage, nil
}
