package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Available modules
var AvailableModules = []ModuleInfo{
	{ID: "web-nginx", Label: "Nginx Web Server", Category: "web", Icon: "🌐"},
	{ID: "web-apache", Label: "Apache Web Server", Category: "web", Icon: "🌐"},
	{ID: "database-mysql", Label: "MySQL / MariaDB", Category: "database", Icon: "🗄️"},
	{ID: "database-postgres", Label: "PostgreSQL", Category: "database", Icon: "🗄️"},
	{ID: "database-mongo", Label: "MongoDB", Category: "database", Icon: "🗄️"},
	{ID: "database-redis", Label: "Redis", Category: "database", Icon: "🗄️"},
	{ID: "docker", Label: "Docker Engine", Category: "containers", Icon: "🐳"},
	{ID: "kubernetes", Label: "Kubernetes", Category: "containers", Icon: "☸️"},
	{ID: "monitoring", Label: "Monitoring Agent", Category: "system", Icon: "📊"},
	{ID: "mail", Label: "Mail Server", Category: "services", Icon: "📧"},
	{ID: "firewall", Label: "Firewall (UFW)", Category: "system", Icon: "🔥"},
	{ID: "dns", Label: "DNS Server", Category: "services", Icon: "🌍"},
}

type ModuleInfo struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Category string `json:"category"`
	Icon     string `json:"icon"`
}

type ServerModule struct {
	ID          string    `json:"id"`
	ServerID    string    `json:"server_id"`
	Module      string    `json:"module"`
	Enabled     bool      `json:"enabled"`
	Config      string    `json:"config"`
	InstalledAt time.Time `json:"installed_at"`
}

type ServerModulesService struct {
	db *pgxpool.Pool
}

func NewServerModulesService(db *pgxpool.Pool) *ServerModulesService {
	return &ServerModulesService{db: db}
}

func (s *ServerModulesService) EnableModule(ctx context.Context, serverID uuid.UUID, module string) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO server_modules (server_id, module, enabled) VALUES ($1, $2, true)
		 ON CONFLICT (server_id, module) DO UPDATE SET enabled = true`,
		serverID, module)
	return err
}

func (s *ServerModulesService) DisableModule(ctx context.Context, serverID uuid.UUID, module string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE server_modules SET enabled = false WHERE server_id = $1 AND module = $2`,
		serverID, module)
	return err
}

func (s *ServerModulesService) RemoveModule(ctx context.Context, serverID uuid.UUID, module string) error {
	_, err := s.db.Exec(ctx,
		`DELETE FROM server_modules WHERE server_id = $1 AND module = $2`,
		serverID, module)
	return err
}

func (s *ServerModulesService) ListModules(ctx context.Context, serverID uuid.UUID) ([]ServerModule, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, server_id, module, enabled, COALESCE(config::text, '{}'), installed_at
		 FROM server_modules WHERE server_id = $1 AND enabled = true ORDER BY installed_at`,
		serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []ServerModule
	for rows.Next() {
		var m ServerModule
		rows.Scan(&m.ID, &m.ServerID, &m.Module, &m.Enabled, &m.Config, &m.InstalledAt)
		modules = append(modules, m)
	}
	return modules, nil
}

func (s *ServerModulesService) GetEnabledModulesForServer(ctx context.Context, serverID uuid.UUID) ([]string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT module FROM server_modules WHERE server_id = $1 AND enabled = true`,
		serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []string
	for rows.Next() {
		var m string
		rows.Scan(&m)
		modules = append(modules, m)
	}
	return modules, nil
}

func (s *ServerModulesService) GetActiveModulesGlobal(ctx context.Context) ([]string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT DISTINCT module FROM server_modules WHERE enabled = true ORDER BY module`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []string
	for rows.Next() {
		var m string
		rows.Scan(&m)
		modules = append(modules, m)
	}
	return modules, nil
}

func (s *ServerModulesService) GetServersForModule(ctx context.Context, module string) ([]string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT server_id FROM server_modules WHERE module = $1 AND enabled = true`,
		module)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *ServerModulesService) GetModuleCounts(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.Query(ctx,
		`SELECT module, COUNT(*) FROM server_modules WHERE enabled = true GROUP BY module`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var m string
		var c int
		rows.Scan(&m, &c)
		counts[m] = c
	}
	return counts, nil
}

func (s *ServerModulesService) SetModules(ctx context.Context, serverID uuid.UUID, modules []string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Disable all existing
	_, err = tx.Exec(ctx, `UPDATE server_modules SET enabled = false WHERE server_id = $1`, serverID)
	if err != nil {
		return err
	}

	// Enable selected
	for _, m := range modules {
		_, err = tx.Exec(ctx,
			`INSERT INTO server_modules (server_id, module, enabled) VALUES ($1, $2, true)
			 ON CONFLICT (server_id, module) DO UPDATE SET enabled = true`,
			serverID, m)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
