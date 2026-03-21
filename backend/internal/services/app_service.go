package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
)

type AppService struct {
	pool *pgxpool.Pool
}

func NewAppService(pool *pgxpool.Pool) *AppService {
	return &AppService{pool: pool}
}

func (s *AppService) Create(ctx context.Context, userID uuid.UUID, req models.CreateAppRequest) (*models.Application, error) {
	gitBranch := req.GitBranch
	if gitBranch == "" {
		gitBranch = "main"
	}
	appType := req.AppType
	if appType == "" {
		appType = "web"
	}
	runtime := req.Runtime
	if runtime == "" {
		runtime = "node"
	}
	deployMethod := req.DeployMethod
	if deployMethod == "" {
		deployMethod = "git"
	}

	envVarsJSON, _ := json.Marshal(req.EnvVars)
	if req.EnvVars == nil {
		envVarsJSON = []byte("{}")
	}

	var domainID *uuid.UUID
	if req.DomainID != "" {
		did, err := uuid.Parse(req.DomainID)
		if err == nil {
			domainID = &did
		}
	}
	serverID, err := uuid.Parse(req.ServerID)
	if err != nil {
		return nil, fmt.Errorf("invalid server_id")
	}

	app := &models.Application{}
	err = s.pool.QueryRow(ctx,
		`INSERT INTO applications (user_id, server_id, domain_id, name, app_type, runtime, deploy_method, git_repo, git_branch, env_vars, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'active')
		 RETURNING id, user_id, domain_id, server_id, name, app_type, runtime, deploy_method,
		           COALESCE(git_repo, ''), COALESCE(git_branch, 'main'), status, created_at, updated_at`,
		userID, serverID, domainID, req.Name, appType, runtime, deployMethod, req.GitRepo, gitBranch, envVarsJSON,
	).Scan(&app.ID, &app.UserID, &app.DomainID, &app.ServerID, &app.Name, &app.AppType,
		&app.Runtime, &app.DeployMethod, &app.GitRepo, &app.GitBranch, &app.Status,
		&app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create application: %w", err)
	}
	return app, nil
}

func (s *AppService) List(ctx context.Context, userID uuid.UUID, role string, page, perPage int) (*models.PaginatedResponse, error) {
	offset := (page - 1) * perPage
	var total int64

	baseWhere := ""
	if role != "admin" {
		baseWhere = fmt.Sprintf(" WHERE user_id = '%s'", userID)
	}

	s.pool.QueryRow(ctx, "SELECT count(*) FROM applications"+baseWhere).Scan(&total)

	rows, err := s.pool.Query(ctx,
		fmt.Sprintf(`SELECT id, user_id, domain_id, server_id, name, app_type, runtime, deploy_method,
		        COALESCE(git_repo,''), COALESCE(git_branch,'main'), status, created_at, updated_at
		 FROM applications%s ORDER BY created_at DESC LIMIT %d OFFSET %d`, baseWhere, perPage, offset))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.Application
	for rows.Next() {
		var a models.Application
		if err := rows.Scan(&a.ID, &a.UserID, &a.DomainID, &a.ServerID, &a.Name, &a.AppType,
			&a.Runtime, &a.DeployMethod, &a.GitRepo, &a.GitBranch, &a.Status,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			continue
		}
		items = append(items, a)
	}

	return &models.PaginatedResponse{
		Data: items, Total: total, Page: page, PerPage: perPage,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}, nil
}

func (s *AppService) GetByID(ctx context.Context, id string) (*models.Application, error) {
	app := &models.Application{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, domain_id, server_id, name, app_type, runtime, deploy_method,
		        COALESCE(git_repo,''), COALESCE(git_branch,'main'), status, created_at, updated_at
		 FROM applications WHERE id = $1`, id,
	).Scan(&app.ID, &app.UserID, &app.DomainID, &app.ServerID, &app.Name, &app.AppType,
		&app.Runtime, &app.DeployMethod, &app.GitRepo, &app.GitBranch, &app.Status,
		&app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (s *AppService) Update(ctx context.Context, id string, req models.UpdateAppRequest) (*models.Application, error) {
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Name != "" {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, req.Name)
		argIdx++
	}
	if req.ServerID != "" {
		sid, err := uuid.Parse(req.ServerID)
		if err == nil {
			setClauses = append(setClauses, fmt.Sprintf("server_id = $%d", argIdx))
			args = append(args, sid)
			argIdx++
		}
	}
	if req.AppType != "" {
		setClauses = append(setClauses, fmt.Sprintf("app_type = $%d", argIdx))
		args = append(args, req.AppType)
		argIdx++
	}
	if req.Runtime != "" {
		setClauses = append(setClauses, fmt.Sprintf("runtime = $%d", argIdx))
		args = append(args, req.Runtime)
		argIdx++
	}
	if req.DeployMethod != "" {
		setClauses = append(setClauses, fmt.Sprintf("deploy_method = $%d", argIdx))
		args = append(args, req.DeployMethod)
		argIdx++
	}
	if req.GitRepo != "" {
		setClauses = append(setClauses, fmt.Sprintf("git_repo = $%d", argIdx))
		args = append(args, req.GitRepo)
		argIdx++
	}
	if req.GitBranch != "" {
		setClauses = append(setClauses, fmt.Sprintf("git_branch = $%d", argIdx))
		args = append(args, req.GitBranch)
		argIdx++
	}
	if req.DomainID != "" {
		did, err := uuid.Parse(req.DomainID)
		if err == nil {
			setClauses = append(setClauses, fmt.Sprintf("domain_id = $%d", argIdx))
			args = append(args, did)
			argIdx++
		}
	}
	if req.EnvVars != nil {
		envJSON, _ := json.Marshal(req.EnvVars)
		setClauses = append(setClauses, fmt.Sprintf("env_vars = $%d", argIdx))
		args = append(args, envJSON)
		argIdx++
	}
	if req.Status != "" {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, req.Status)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now())
	argIdx++

	args = append(args, id)
	query := fmt.Sprintf(
		`UPDATE applications SET %s WHERE id = $%d
		 RETURNING id, user_id, domain_id, server_id, name, app_type, runtime, deploy_method,
		           COALESCE(git_repo,''), COALESCE(git_branch,'main'), status, created_at, updated_at`,
		strings.Join(setClauses, ", "), argIdx,
	)

	app := &models.Application{}
	err := s.pool.QueryRow(ctx, query, args...).Scan(
		&app.ID, &app.UserID, &app.DomainID, &app.ServerID, &app.Name, &app.AppType,
		&app.Runtime, &app.DeployMethod, &app.GitRepo, &app.GitBranch, &app.Status,
		&app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update application: %w", err)
	}
	return app, nil
}

func (s *AppService) Delete(ctx context.Context, id string) error {
	result, err := s.pool.Exec(ctx, "DELETE FROM applications WHERE id = $1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("application not found")
	}
	return nil
}
