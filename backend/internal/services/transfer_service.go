package services

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	novacrypto "github.com/novapanel/novapanel/internal/crypto"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/provisioner"
)

type TransferService struct {
	pool *pgxpool.Pool
}

func NewTransferService(pool *pgxpool.Pool) *TransferService {
	return &TransferService{pool: pool}
}

// CreateTransfer creates a new rsync transfer job and executes it async
func (s *TransferService) CreateTransfer(ctx context.Context, userID uuid.UUID, req models.CreateTransferRequest) (*models.TransferJob, error) {
	job := &models.TransferJob{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO transfer_jobs (user_id, source_server_id, dest_server_id, source_path, dest_path,
			direction, rsync_options, exclude_patterns, bandwidth_limit, delete_extra, dry_run, status)
		 VALUES ($1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, $10, $11, 'pending')
		 RETURNING id, user_id, source_server_id, dest_server_id, source_path, dest_path,
			direction, rsync_options, exclude_patterns, bandwidth_limit, delete_extra, dry_run,
			status, bytes_transferred, files_transferred, progress, created_at`,
		userID, req.SourceServerID, req.DestServerID, req.SourcePath, req.DestPath,
		req.Direction, req.RsyncOptions, req.ExcludePatterns, req.BandwidthLimit,
		req.DeleteExtra, req.DryRun,
	).Scan(&job.ID, &job.UserID, &job.SourceServerID, &job.DestServerID,
		&job.SourcePath, &job.DestPath, &job.Direction, &job.RsyncOptions,
		&job.ExcludePatterns, &job.BandwidthLimit, &job.DeleteExtra, &job.DryRun,
		&job.Status, &job.BytesTransferred, &job.FilesTransferred, &job.Progress, &job.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create transfer job: %w", err)
	}

	// Execute async
	go s.executeTransfer(job.ID.String())

	return job, nil
}

// executeTransfer runs the rsync command via SSH
func (s *TransferService) executeTransfer(jobID string) {
	ctx := context.Background()

	// Mark as running
	s.pool.Exec(ctx, `UPDATE transfer_jobs SET status = 'running', started_at = NOW() WHERE id = $1`, jobID)

	// Get job details
	var job models.TransferJob
	err := s.pool.QueryRow(ctx,
		`SELECT id, source_server_id, dest_server_id, source_path, dest_path,
			direction, rsync_options, exclude_patterns, bandwidth_limit, delete_extra, dry_run
		 FROM transfer_jobs WHERE id = $1`, jobID,
	).Scan(&job.ID, &job.SourceServerID, &job.DestServerID, &job.SourcePath, &job.DestPath,
		&job.Direction, &job.RsyncOptions, &job.ExcludePatterns, &job.BandwidthLimit,
		&job.DeleteExtra, &job.DryRun)
	if err != nil {
		s.failJob(ctx, jobID, fmt.Sprintf("failed to get job: %v", err))
		return
	}

	// Build rsync command
	cmd := s.buildRsyncCommand(job)

	// Determine which server to run the command on
	var serverID string
	if job.Direction == "push" || job.Direction == "local" {
		if job.SourceServerID != nil {
			serverID = job.SourceServerID.String()
		}
	} else {
		// pull: run on destination
		if job.DestServerID != nil {
			serverID = job.DestServerID.String()
		}
	}

	if serverID == "" {
		s.failJob(ctx, jobID, "no server specified for transfer")
		return
	}

	// Get server SSH info
	var server provisioner.ServerInfo
	var port int
	var encKey, encPassword string
	err = s.pool.QueryRow(ctx,
		`SELECT host(ip_address), port, ssh_user, COALESCE(ssh_key, ''), COALESCE(ssh_password, ''), COALESCE(auth_method, 'password')
		 FROM servers WHERE id = $1`, serverID,
	).Scan(&server.IPAddress, &port, &server.SSHUser, &encKey, &encPassword, &server.AuthMethod)
	if err != nil {
		s.failJob(ctx, jobID, fmt.Sprintf("failed to get server: %v", err))
		return
	}
	server.Port = port
	if cryptoKey, kerr := novacrypto.GetEncryptionKey(); kerr == nil {
		if encKey != "" {
			if dec, derr := novacrypto.Decrypt(encKey, cryptoKey); derr == nil {
				encKey = dec
			}
		}
		if encPassword != "" {
			if dec, derr := novacrypto.Decrypt(encPassword, cryptoKey); derr == nil {
				encPassword = dec
			}
		}
	}
	server.SSHKey = encKey
	server.SSHPassword = encPassword

	// Run rsync
	output, err := provisioner.RunScript(server, cmd)
	if err != nil {
		s.pool.Exec(ctx,
			`UPDATE transfer_jobs SET status = 'failed', output = $2, completed_at = NOW() WHERE id = $1`,
			jobID, output)
		return
	}

	// Parse rsync stats from output
	s.pool.Exec(ctx,
		`UPDATE transfer_jobs SET status = 'completed', output = $2, progress = 100, completed_at = NOW() WHERE id = $1`,
		jobID, output)
}

// sanitizeRsyncOpts whitelists safe rsync flags to prevent command injection.
func sanitizeRsyncOpts(opts string) string {
	allowed := []string{
		"-a", "-v", "-z", "-h", "-r", "-l", "-p", "-t", "-g", "-o", "-D", "-n",
		"--archive", "--verbose", "--compress", "--human-readable", "--recursive",
		"--progress", "--stats", "--checksum", "--backup", "--dry-run",
		"--partial", "--append", "--update", "--links", "--times", "--perms",
		"--owner", "--group", "--devices", "--specials",
	}
	var safe []string
	for _, part := range strings.Fields(opts) {
		for _, a := range allowed {
			if part == a {
				safe = append(safe, part)
				break
			}
		}
	}
	if len(safe) == 0 {
		return "-avzh --progress"
	}
	return strings.Join(safe, " ")
}

func (s *TransferService) buildRsyncCommand(job models.TransferJob) string {
	opts := sanitizeRsyncOpts(job.RsyncOptions)

	cmd := fmt.Sprintf("rsync %s", opts)

	// Add bandwidth limit
	if job.BandwidthLimit > 0 {
		cmd += fmt.Sprintf(" --bwlimit=%d", job.BandwidthLimit)
	}

	// Add delete option
	if job.DeleteExtra {
		cmd += " --delete"
	}

	// Add dry-run
	if job.DryRun {
		cmd += " --dry-run"
	}

	// Add exclude patterns
	if job.ExcludePatterns != "" {
		// Each pattern on a new line
		for _, p := range splitPatterns(job.ExcludePatterns) {
			if p != "" {
				cmd += fmt.Sprintf(" --exclude='%s'", p)
			}
		}
	}

	// Add source and destination
	if job.Direction == "local" {
		// Local transfer on same server
		cmd += fmt.Sprintf(" '%s' '%s'", job.SourcePath, job.DestPath)
	} else if job.Direction == "push" {
		// Push from source server to dest server
		var destUser, destIP string
		var destPort int
		s.pool.QueryRow(context.Background(),
			`SELECT ssh_user, ip_address, port FROM servers WHERE id = $1`,
			job.DestServerID).Scan(&destUser, &destIP, &destPort)
		cmd += fmt.Sprintf(" -e 'ssh -p %d -o StrictHostKeyChecking=no' '%s' '%s@%s:%s'",
			destPort, job.SourcePath, destUser, destIP, job.DestPath)
	} else {
		// Pull from source server to dest server
		var srcUser, srcIP string
		var srcPort int
		s.pool.QueryRow(context.Background(),
			`SELECT ssh_user, ip_address, port FROM servers WHERE id = $1`,
			job.SourceServerID).Scan(&srcUser, &srcIP, &srcPort)
		cmd += fmt.Sprintf(" -e 'ssh -p %d -o StrictHostKeyChecking=no' '%s@%s:%s' '%s'",
			srcPort, srcUser, srcIP, job.SourcePath, job.DestPath)
	}

	return cmd
}

func splitPatterns(s string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if c == ',' || c == '\n' {
			if current != "" {
				result = append(result, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func (s *TransferService) failJob(ctx context.Context, jobID, errMsg string) {
	s.pool.Exec(ctx,
		`UPDATE transfer_jobs SET status = 'failed', output = $2, completed_at = NOW() WHERE id = $1`,
		jobID, errMsg)
}

// ListTransfers gets paginated transfer jobs for a user
func (s *TransferService) ListTransfers(ctx context.Context, userID uuid.UUID, page, perPage int) (*models.PaginatedResponse, error) {
	offset := (page - 1) * perPage
	var total int64
	s.pool.QueryRow(ctx, `SELECT count(*) FROM transfer_jobs WHERE user_id = $1`, userID).Scan(&total)

	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, source_server_id, dest_server_id, source_path, dest_path,
			direction, rsync_options, exclude_patterns, bandwidth_limit, delete_extra, dry_run,
			status, bytes_transferred, files_transferred, progress, output,
			started_at, completed_at, created_at
		 FROM transfer_jobs WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, perPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []models.TransferJob
	for rows.Next() {
		var j models.TransferJob
		rows.Scan(&j.ID, &j.UserID, &j.SourceServerID, &j.DestServerID,
			&j.SourcePath, &j.DestPath, &j.Direction, &j.RsyncOptions,
			&j.ExcludePatterns, &j.BandwidthLimit, &j.DeleteExtra, &j.DryRun,
			&j.Status, &j.BytesTransferred, &j.FilesTransferred, &j.Progress, &j.Output,
			&j.StartedAt, &j.CompletedAt, &j.CreatedAt)
		jobs = append(jobs, j)
	}

	return &models.PaginatedResponse{
		Data: jobs, Total: total, Page: page, PerPage: perPage,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}, nil
}

// GetTransfer gets a single transfer job
func (s *TransferService) GetTransfer(ctx context.Context, id string) (*models.TransferJob, error) {
	j := &models.TransferJob{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, source_server_id, dest_server_id, source_path, dest_path,
			direction, rsync_options, exclude_patterns, bandwidth_limit, delete_extra, dry_run,
			status, bytes_transferred, files_transferred, progress, output,
			started_at, completed_at, created_at
		 FROM transfer_jobs WHERE id = $1`, id,
	).Scan(&j.ID, &j.UserID, &j.SourceServerID, &j.DestServerID,
		&j.SourcePath, &j.DestPath, &j.Direction, &j.RsyncOptions,
		&j.ExcludePatterns, &j.BandwidthLimit, &j.DeleteExtra, &j.DryRun,
		&j.Status, &j.BytesTransferred, &j.FilesTransferred, &j.Progress, &j.Output,
		&j.StartedAt, &j.CompletedAt, &j.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("transfer job not found")
	}
	return j, nil
}

// CancelTransfer cancels a running transfer (marks as cancelled)
func (s *TransferService) CancelTransfer(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE transfer_jobs SET status = 'cancelled', completed_at = NOW() WHERE id = $1 AND status IN ('pending', 'running')`,
		id)
	return err
}

// DeleteTransfer deletes a completed/failed/cancelled transfer job
func (s *TransferService) DeleteTransfer(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM transfer_jobs WHERE id = $1`, id)
	return err
}

// RetryTransfer re-runs a failed transfer
func (s *TransferService) RetryTransfer(ctx context.Context, id string) error {
	s.pool.Exec(ctx,
		`UPDATE transfer_jobs SET status = 'pending', output = '', progress = 0, started_at = NULL, completed_at = NULL WHERE id = $1`,
		id)
	go s.executeTransfer(id)
	return nil
}

// PreviewTransfer runs rsync in dry-run mode to show what would happen
func (s *TransferService) PreviewTransfer(ctx context.Context, userID uuid.UUID, req models.CreateTransferRequest) (*models.TransferJob, error) {
	req.DryRun = true
	return s.CreateTransfer(ctx, userID, req)
}

// GetScheduledTransfers gets any scheduled/recurring transfers
func (s *TransferService) ListSchedules(ctx context.Context, userID uuid.UUID) ([]models.TransferSchedule, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, name, source_server_id, dest_server_id, source_path, dest_path,
			direction, rsync_options, exclude_patterns, bandwidth_limit, delete_extra,
			cron_expression, is_active, last_run, next_run, created_at
		 FROM transfer_schedules WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scheds []models.TransferSchedule
	for rows.Next() {
		var s models.TransferSchedule
		rows.Scan(&s.ID, &s.UserID, &s.Name, &s.SourceServerID, &s.DestServerID,
			&s.SourcePath, &s.DestPath, &s.Direction, &s.RsyncOptions,
			&s.ExcludePatterns, &s.BandwidthLimit, &s.DeleteExtra,
			&s.CronExpression, &s.IsActive, &s.LastRun, &s.NextRun, &s.CreatedAt)
		scheds = append(scheds, s)
	}
	return scheds, nil
}

func (s *TransferService) CreateSchedule(ctx context.Context, userID uuid.UUID, req models.CreateScheduleRequest) (*models.TransferSchedule, error) {
	sched := &models.TransferSchedule{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO transfer_schedules (user_id, name, source_server_id, dest_server_id, source_path, dest_path,
			direction, rsync_options, exclude_patterns, bandwidth_limit, delete_extra, cron_expression, is_active)
		 VALUES ($1, $2, NULLIF($3, '')::uuid, NULLIF($4, '')::uuid, $5, $6, $7, $8, $9, $10, $11, $12, true)
		 RETURNING id, user_id, name, source_server_id, dest_server_id, source_path, dest_path,
			direction, rsync_options, exclude_patterns, bandwidth_limit, delete_extra,
			cron_expression, is_active, created_at`,
		userID, req.Name, req.SourceServerID, req.DestServerID, req.SourcePath, req.DestPath,
		req.Direction, req.RsyncOptions, req.ExcludePatterns, req.BandwidthLimit, req.DeleteExtra,
		req.CronExpression,
	).Scan(&sched.ID, &sched.UserID, &sched.Name, &sched.SourceServerID, &sched.DestServerID,
		&sched.SourcePath, &sched.DestPath, &sched.Direction, &sched.RsyncOptions,
		&sched.ExcludePatterns, &sched.BandwidthLimit, &sched.DeleteExtra,
		&sched.CronExpression, &sched.IsActive, &sched.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}
	return sched, nil
}

func (s *TransferService) DeleteSchedule(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM transfer_schedules WHERE id = $1`, id)
	return err
}

func (s *TransferService) ToggleSchedule(ctx context.Context, id string, active bool) error {
	_, err := s.pool.Exec(ctx, `UPDATE transfer_schedules SET is_active = $2 WHERE id = $1`, id, active)
	return err
}

// DiskUsage returns size info for a path on a server
func (s *TransferService) DiskUsage(ctx context.Context, serverID, path string) (string, error) {
	var server provisioner.ServerInfo
	var port int
	var encKey, encPassword string
	err := s.pool.QueryRow(ctx,
		`SELECT host(ip_address), port, ssh_user, COALESCE(ssh_key, ''), COALESCE(ssh_password, ''), COALESCE(auth_method, 'password')
		 FROM servers WHERE id = $1`, serverID,
	).Scan(&server.IPAddress, &port, &server.SSHUser, &encKey, &encPassword, &server.AuthMethod)
	if err != nil {
		return "", err
	}
	server.Port = port
	if cryptoKey, kerr := novacrypto.GetEncryptionKey(); kerr == nil {
		if encKey != "" {
			if dec, derr := novacrypto.Decrypt(encKey, cryptoKey); derr == nil {
				encKey = dec
			}
		}
		if encPassword != "" {
			if dec, derr := novacrypto.Decrypt(encPassword, cryptoKey); derr == nil {
				encPassword = dec
			}
		}
	}
	server.SSHKey = encKey
	server.SSHPassword = encPassword

	script := fmt.Sprintf("du -sh '%s' 2>/dev/null && echo '---' && ls -la '%s' 2>/dev/null | head -20", path, path)
	output, err := provisioner.RunScript(server, script)
	if err != nil {
		return "", err
	}
	return output, nil
}

// RunDueSchedules executes transfer schedules whose next_run has passed.
func (s *TransferService) RunDueSchedules(ctx context.Context) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, source_server_id, dest_server_id, source_path, dest_path,
		        direction, rsync_options, exclude_patterns, bandwidth_limit, delete_extra
		 FROM transfer_schedules WHERE is_active = true AND next_run <= NOW()`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var sched models.TransferSchedule
		var req models.CreateTransferRequest
		if err := rows.Scan(&sched.ID, &sched.UserID, &sched.SourceServerID, &sched.DestServerID,
			&sched.SourcePath, &sched.DestPath, &sched.Direction, &sched.RsyncOptions,
			&sched.ExcludePatterns, &sched.BandwidthLimit, &sched.DeleteExtra); err != nil {
			continue
		}
		req = models.CreateTransferRequest{
			SourcePath:      sched.SourcePath,
			DestPath:        sched.DestPath,
			Direction:       sched.Direction,
			RsyncOptions:    sched.RsyncOptions,
			ExcludePatterns: sched.ExcludePatterns,
			BandwidthLimit:  sched.BandwidthLimit,
			DeleteExtra:     sched.DeleteExtra,
		}
		if sched.SourceServerID != nil {
			req.SourceServerID = sched.SourceServerID.String()
		}
		if sched.DestServerID != nil {
			req.DestServerID = sched.DestServerID.String()
		}

		go func(schedID string, userID uuid.UUID, r models.CreateTransferRequest) {
			runCtx := context.Background()
			_, _ = s.CreateTransfer(runCtx, userID, r)
			s.pool.Exec(runCtx,
				`UPDATE transfer_schedules SET last_run = NOW(), next_run = NOW() + INTERVAL '1 day' WHERE id = $1`,
				schedID)
		}(sched.ID.String(), sched.UserID, req)
	}
}
