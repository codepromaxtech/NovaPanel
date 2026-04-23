package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	novacrypto "github.com/novapanel/novapanel/internal/crypto"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/provisioner"
)

type BackupService struct {
	pool      *pgxpool.Pool
	cryptoKey []byte
}

func NewBackupService(pool *pgxpool.Pool, encryptionKey string) *BackupService {
	return &BackupService{pool: pool, cryptoKey: novacrypto.DeriveKey(encryptionKey)}
}

func (s *BackupService) getServerSSH(ctx context.Context, serverID string) (provisioner.ServerInfo, error) {
	return GetServerInfo(ctx, s.pool, serverID)
}

// RunDueSchedules executes any backup schedules whose next_run_at has passed.
func (s *BackupService) RunDueSchedules(ctx context.Context) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, server_id, type, storage, frequency, retention_days
		 FROM backup_schedules WHERE is_active = true AND next_run_at <= NOW()`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, freq, btype, storage string
		var userID uuid.UUID
		var serverID *uuid.UUID
		var retention int
		if err := rows.Scan(&id, &userID, &serverID, &btype, &storage, &freq, &retention); err != nil {
			continue
		}
		go s.executeScheduledBackup(id, userID, serverID, btype, storage, freq, retention)
	}
}

func (s *BackupService) executeScheduledBackup(scheduleID string, userID uuid.UUID, serverID *uuid.UUID, btype, storage, freq string, retention int) {
	ctx := context.Background()
	timestamp := time.Now().Format("20060102-150405")

	// Create a backup record
	var backupID string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO backups (user_id, server_id, type, storage, status, started_at)
		 VALUES ($1, $2, $3, $4, 'running', NOW())
		 RETURNING id`, userID, serverID, btype, storage,
	).Scan(&backupID)
	if err != nil {
		log.Printf("backup schedule %s: failed to create record: %v", scheduleID, err)
		return
	}

	backupPath := fmt.Sprintf("/var/backups/novapanel/%s-%s.tar.gz", scheduleID, timestamp)

	if serverID != nil {
		srv, err := s.getServerSSH(ctx, serverID.String())
		if err == nil {
			script := fmt.Sprintf(`
mkdir -p /var/backups/novapanel
tar -czf %s /var/www 2>/dev/null
SIZE=$(stat -c%%s %s 2>/dev/null || echo 0)
echo "BACKUP_SIZE=${SIZE}"
`, backupPath, backupPath)
			out, err := provisioner.RunScript(srv, script)
			status := "completed"
			if err != nil {
				status = "failed"
				log.Printf("backup schedule %s execution error: %v — %s", scheduleID, err, out)
			}
			s.pool.Exec(ctx,
				`UPDATE backups SET status = $1, path = $2, completed_at = NOW() WHERE id = $3`,
				status, backupPath, backupID)
		}
	}

	// Compute next run based on frequency
	nextRun := nextRunTime(freq)
	s.pool.Exec(ctx,
		`UPDATE backup_schedules SET last_run_at = NOW(), next_run_at = $1 WHERE id = $2`,
		nextRun, scheduleID)

	// Prune old backups
	s.pool.Exec(ctx,
		`DELETE FROM backups WHERE user_id = $1 AND server_id = $2 AND created_at < NOW() - ($3 || ' days')::INTERVAL`,
		userID, serverID, retention)
}

func nextRunTime(freq string) time.Time {
	now := time.Now()
	switch freq {
	case "hourly":
		return now.Add(time.Hour)
	case "weekly":
		return now.Add(7 * 24 * time.Hour)
	case "monthly":
		return now.AddDate(0, 1, 0)
	default: // daily
		return now.Add(24 * time.Hour)
	}
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
	var args []interface{}

	if role != "admin" {
		args = append(args, userID)
		query += ` WHERE user_id = $1`
		countQuery += ` WHERE user_id = $1`
	}

	s.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT %d OFFSET %d`, perPage, offset)

	rows, err := s.pool.Query(ctx, query, args...)
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

func (s *BackupService) Delete(ctx context.Context, id string, userID uuid.UUID, role string) error {
	var ownerID uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT user_id FROM backups WHERE id = $1`, id).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("backup not found")
	}
	if role != "admin" && ownerID != userID {
		return fmt.Errorf("backup not found")
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM backups WHERE id = $1`, id)
	return err
}
