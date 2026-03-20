package services

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/provisioner"
)

// SystemctlService manages systemd services on remote servers via SSH
type SystemctlService struct {
	pool *pgxpool.Pool
}

func NewSystemctlService(pool *pgxpool.Pool) *SystemctlService {
	return &SystemctlService{pool: pool}
}

func (s *SystemctlService) getServer(ctx context.Context, serverID string) (provisioner.ServerInfo, error) {
	var server provisioner.ServerInfo
	var port int
	err := s.pool.QueryRow(ctx,
		`SELECT ip_address, port, ssh_user, COALESCE(ssh_key, ''), COALESCE(ssh_password, ''), COALESCE(auth_method, 'password')
		 FROM servers WHERE id = $1`, serverID,
	).Scan(&server.IPAddress, &port, &server.SSHUser, &server.SSHKey, &server.SSHPassword, &server.AuthMethod)
	server.Port = port
	return server, err
}

// ListServices lists all active/loaded services
func (s *SystemctlService) ListServices(ctx context.Context, serverID, filter string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	cmd := "systemctl list-units --type=service --no-pager --no-legend 2>/dev/null"
	if filter != "" {
		cmd += fmt.Sprintf(" | grep -i '%s'", filter)
	}
	return provisioner.RunScript(server, cmd)
}

// GetServiceStatus gets detailed status for a service
func (s *SystemctlService) GetServiceStatus(ctx context.Context, serverID, service string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	cmd := fmt.Sprintf("systemctl status %s --no-pager 2>&1", service)
	return provisioner.RunScript(server, cmd)
}

// StartService starts a service
func (s *SystemctlService) StartService(ctx context.Context, serverID, service string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	cmd := fmt.Sprintf("systemctl start %s 2>&1 && systemctl status %s --no-pager 2>&1", service, service)
	return provisioner.RunScript(server, cmd)
}

// StopService stops a service
func (s *SystemctlService) StopService(ctx context.Context, serverID, service string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	cmd := fmt.Sprintf("systemctl stop %s 2>&1 && echo 'Service %s stopped'", service, service)
	return provisioner.RunScript(server, cmd)
}

// RestartService restarts a service
func (s *SystemctlService) RestartService(ctx context.Context, serverID, service string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	cmd := fmt.Sprintf("systemctl restart %s 2>&1 && systemctl status %s --no-pager 2>&1", service, service)
	return provisioner.RunScript(server, cmd)
}

// ReloadService reloads a service config
func (s *SystemctlService) ReloadService(ctx context.Context, serverID, service string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	cmd := fmt.Sprintf("systemctl reload %s 2>&1 && echo 'Service %s reloaded'", service, service)
	return provisioner.RunScript(server, cmd)
}

// EnableService enables a service to start on boot
func (s *SystemctlService) EnableService(ctx context.Context, serverID, service string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	cmd := fmt.Sprintf("systemctl enable %s 2>&1 && echo 'Service %s enabled'", service, service)
	return provisioner.RunScript(server, cmd)
}

// DisableService disables a service from starting on boot
func (s *SystemctlService) DisableService(ctx context.Context, serverID, service string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	cmd := fmt.Sprintf("systemctl disable %s 2>&1 && echo 'Service %s disabled'", service, service)
	return provisioner.RunScript(server, cmd)
}

// GetServiceLogs gets journal logs for a service
func (s *SystemctlService) GetServiceLogs(ctx context.Context, serverID, service string, lines int) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	if lines <= 0 {
		lines = 50
	}
	cmd := fmt.Sprintf("journalctl -u %s --no-pager -n %d 2>&1", service, lines)
	return provisioner.RunScript(server, cmd)
}

// ListFailedServices lists all failed services
func (s *SystemctlService) ListFailedServices(ctx context.Context, serverID string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	cmd := "systemctl list-units --type=service --state=failed --no-pager --no-legend 2>/dev/null"
	return provisioner.RunScript(server, cmd)
}

// ListTimers lists all systemd timers (scheduled tasks)
func (s *SystemctlService) ListTimers(ctx context.Context, serverID string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	cmd := "systemctl list-timers --all --no-pager 2>/dev/null"
	return provisioner.RunScript(server, cmd)
}

// DaemonReload reloads systemd daemon
func (s *SystemctlService) DaemonReload(ctx context.Context, serverID string) (string, error) {
	server, err := s.getServer(ctx, serverID)
	if err != nil {
		return "", err
	}
	cmd := "systemctl daemon-reload 2>&1 && echo 'Daemon reloaded'"
	return provisioner.RunScript(server, cmd)
}
