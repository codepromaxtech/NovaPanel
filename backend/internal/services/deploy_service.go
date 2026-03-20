package services

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
)

type DeployService struct {
	pool *pgxpool.Pool
}

func NewDeployService(pool *pgxpool.Pool) *DeployService {
	return &DeployService{pool: pool}
}

func (s *DeployService) Create(ctx context.Context, userID uuid.UUID, req models.CreateDeploymentRequest) (*models.Deployment, error) {
	branch := req.Branch
	if branch == "" {
		branch = "main"
	}

	d := &models.Deployment{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO deployments (app_id, user_id, branch, status)
		 VALUES ($1, $2, $3, 'pending')
		 RETURNING id, app_id, user_id, commit_hash, branch, status, build_log, started_at, completed_at, created_at`,
		req.AppID, userID, branch,
	).Scan(&d.ID, &d.AppID, &d.UserID, &d.CommitHash, &d.Branch, &d.Status, &d.BuildLog, &d.StartedAt, &d.CompletedAt, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}
	return d, nil
}

func (s *DeployService) List(ctx context.Context, userID uuid.UUID, role string, page, perPage int) (*models.PaginatedResponse, error) {
	offset := (page - 1) * perPage
	var total int64

	query := `SELECT id, app_id, user_id, commit_hash, branch, status, build_log, started_at, completed_at, created_at FROM deployments`
	countQuery := `SELECT count(*) FROM deployments`
	if role != "admin" {
		query += fmt.Sprintf(` WHERE user_id = '%s'`, userID)
		countQuery += fmt.Sprintf(` WHERE user_id = '%s'`, userID)
	}
	s.pool.QueryRow(ctx, countQuery).Scan(&total)
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT %d OFFSET %d`, perPage, offset)

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.Deployment
	for rows.Next() {
		var d models.Deployment
		if err := rows.Scan(&d.ID, &d.AppID, &d.UserID, &d.CommitHash, &d.Branch, &d.Status, &d.BuildLog, &d.StartedAt, &d.CompletedAt, &d.CreatedAt); err != nil {
			continue
		}
		items = append(items, d)
	}

	return &models.PaginatedResponse{
		Data: items, Total: total, Page: page, PerPage: perPage,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}, nil
}

func (s *DeployService) GetByID(ctx context.Context, id string) (*models.Deployment, error) {
	d := &models.Deployment{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, app_id, user_id, commit_hash, branch, status, build_log, started_at, completed_at, created_at FROM deployments WHERE id = $1`, id,
	).Scan(&d.ID, &d.AppID, &d.UserID, &d.CommitHash, &d.Branch, &d.Status, &d.BuildLog, &d.StartedAt, &d.CompletedAt, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	return d, nil
}
