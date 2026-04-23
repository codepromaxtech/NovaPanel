package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/provisioner"
)

// CronService manages cron jobs on remote servers via SSH
type CronService struct {
	pool *pgxpool.Pool
}

func NewCronService(pool *pgxpool.Pool) *CronService {
	return &CronService{pool: pool}
}

func (s *CronService) getServer(ctx context.Context, serverID string) (provisioner.ServerInfo, error) {
	return GetServerInfo(ctx, s.pool, serverID)
}

// ListCronJobs lists all cron jobs for a user on a server
func (s *CronService) ListCronJobs(ctx context.Context, serverID, user string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	if user == "" {
		user = "root"
	}
	cmd := fmt.Sprintf("crontab -u %s -l 2>/dev/null || echo 'no crontab for %s'", user, user)
	return provisioner.RunScript(server, cmd)
}

// AddCronJob adds a cron job for a user on a server
func (s *CronService) AddCronJob(ctx context.Context, serverID, user, schedule, command, comment string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	if user == "" {
		user = "root"
	}

	escapedCmd := strings.ReplaceAll(command, "'", "'\\''")
	escapedComment := strings.ReplaceAll(comment, "'", "'\\''")

	var script string
	if comment != "" {
		script = fmt.Sprintf(`(crontab -u %s -l 2>/dev/null; echo '# %s'; echo '%s %s') | crontab -u %s - 2>&1 && echo 'Cron job added successfully'`,
			user, escapedComment, schedule, escapedCmd, user)
	} else {
		script = fmt.Sprintf(`(crontab -u %s -l 2>/dev/null; echo '%s %s') | crontab -u %s - 2>&1 && echo 'Cron job added successfully'`,
			user, schedule, escapedCmd, user)
	}
	return provisioner.RunScript(server, script)
}

// DeleteCronJob removes a specific line from crontab
func (s *CronService) DeleteCronJob(ctx context.Context, serverID, user string, lineNumber int) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	if user == "" {
		user = "root"
	}
	// Delete specific line number
	script := fmt.Sprintf(`crontab -u %s -l 2>/dev/null | sed '%dd' | crontab -u %s - 2>&1 && echo 'Cron job deleted'`, user, lineNumber, user)
	return provisioner.RunScript(server, script)
}

// UpdateCronJob replaces a specific cron line
func (s *CronService) UpdateCronJob(ctx context.Context, serverID, user string, lineNumber int, newLine string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	if user == "" {
		user = "root"
	}
	escapedLine := strings.ReplaceAll(newLine, "'", "'\\''")
	script := fmt.Sprintf(`crontab -u %s -l 2>/dev/null | sed '%ds/.*/%s/' | crontab -u %s - 2>&1 && echo 'Cron job updated'`,
		user, lineNumber, escapedLine, user)
	return provisioner.RunScript(server, script)
}

// ListCronUsers lists users that have crontab entries
func (s *CronService) ListCronUsers(ctx context.Context, serverID string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	script := `ls /var/spool/cron/crontabs/ 2>/dev/null || ls /var/spool/cron/ 2>/dev/null || echo 'root'`
	return provisioner.RunScript(server, script)
}

// GetCronLog gets recent cron execution logs
func (s *CronService) GetCronLog(ctx context.Context, serverID string, lines int) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	if lines <= 0 {
		lines = 50
	}
	script := fmt.Sprintf(`grep -i cron /var/log/syslog 2>/dev/null | tail -%d || journalctl -u cron --no-pager -n %d 2>/dev/null || echo 'No cron logs found'`, lines, lines)
	return provisioner.RunScript(server, script)
}
