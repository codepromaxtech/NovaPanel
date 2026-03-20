package services

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
)

type BackupService struct {
	pool *pgxpool.Pool
}

func NewBackupService(pool *pgxpool.Pool) *BackupService {
	return &BackupService{pool: pool}
}

func (s *BackupService) Create(ctx context.Context, userID uuid.UUID, req models.CreateBackupRequest) (*models.Backup, error) {
	backupType := req.Type
	if backupType == "" {
		backupType = "full"
	}
	storage := req.Storage
	if storage == "" {
		storage = "local"
	}

	var serverID *uuid.UUID
	if req.ServerID != "" {
		parsed, err := uuid.Parse(req.ServerID)
		if err != nil {
			return nil, fmt.Errorf("invalid server_id")
		}
		serverID = &parsed
	}

	backup := &models.Backup{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO backups (user_id, server_id, type, storage, status)
		 VALUES ($1, $2, $3, $4, 'pending')
		 RETURNING id, user_id, server_id, type, storage, path, size_mb, status, started_at, completed_at, expires_at, created_at`,
		userID, serverID, backupType, storage,
	).Scan(&backup.ID, &backup.UserID, &backup.ServerID, &backup.Type, &backup.Storage, &backup.Path, &backup.SizeMB, &backup.Status, &backup.StartedAt, &backup.CompletedAt, &backup.ExpiresAt, &backup.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}
	return backup, nil
}

func (s *BackupService) List(ctx context.Context, userID uuid.UUID, role string, page, perPage int) (*models.PaginatedResponse, error) {
	offset := (page - 1) * perPage
	var total int64

	query := `SELECT id, user_id, server_id, type, storage, path, size_mb, status, started_at, completed_at, expires_at, created_at FROM backups`
	countQuery := `SELECT count(*) FROM backups`

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

	var backups []models.Backup
	for rows.Next() {
		var b models.Backup
		if err := rows.Scan(&b.ID, &b.UserID, &b.ServerID, &b.Type, &b.Storage, &b.Path, &b.SizeMB, &b.Status, &b.StartedAt, &b.CompletedAt, &b.ExpiresAt, &b.CreatedAt); err != nil {
			continue
		}
		backups = append(backups, b)
	}

	return &models.PaginatedResponse{
		Data:       backups,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}, nil
}

func (s *BackupService) Delete(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM backups WHERE id = $1`, id)
	return err
}
