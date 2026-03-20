package services

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
)

type DatabaseService struct {
	pool *pgxpool.Pool
}

func NewDatabaseService(pool *pgxpool.Pool) *DatabaseService {
	return &DatabaseService{pool: pool}
}

func (s *DatabaseService) Create(ctx context.Context, userID uuid.UUID, req models.CreateDatabaseRequest) (*models.Database, error) {
	engine := req.Engine
	if engine == "" {
		engine = "mysql"
	}
	charset := req.Charset
	if charset == "" {
		charset = "utf8mb4"
	}

	dbUser := req.Name + "_user"
	dbPass := uuid.New().String()[:16]

	var serverID *uuid.UUID
	if req.ServerID != "" {
		parsed, err := uuid.Parse(req.ServerID)
		if err != nil {
			return nil, fmt.Errorf("invalid server_id")
		}
		serverID = &parsed
	}

	db := &models.Database{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO databases (user_id, server_id, name, engine, db_user, db_password_enc, charset, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'active')
		 RETURNING id, user_id, server_id, name, engine, db_user, charset, size_mb, status, created_at, updated_at`,
		userID, serverID, req.Name, engine, dbUser, dbPass, charset,
	).Scan(&db.ID, &db.UserID, &db.ServerID, &db.Name, &db.Engine, &db.DBUser, &db.Charset, &db.SizeMB, &db.Status, &db.CreatedAt, &db.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}
	return db, nil
}

func (s *DatabaseService) List(ctx context.Context, userID uuid.UUID, role string, page, perPage int) (*models.PaginatedResponse, error) {
	offset := (page - 1) * perPage
	var total int64

	query := `SELECT id, user_id, server_id, name, engine, db_user, charset, size_mb, status, created_at, updated_at FROM databases`
	countQuery := `SELECT count(*) FROM databases`

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

	var databases []models.Database
	for rows.Next() {
		var db models.Database
		if err := rows.Scan(&db.ID, &db.UserID, &db.ServerID, &db.Name, &db.Engine, &db.DBUser, &db.Charset, &db.SizeMB, &db.Status, &db.CreatedAt, &db.UpdatedAt); err != nil {
			continue
		}
		databases = append(databases, db)
	}

	return &models.PaginatedResponse{
		Data:       databases,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}, nil
}

func (s *DatabaseService) Delete(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM databases WHERE id = $1`, id)
	return err
}
