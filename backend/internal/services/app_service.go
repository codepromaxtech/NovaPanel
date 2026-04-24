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
	novacrypto "github.com/novapanel/novapanel/internal/crypto"
	"github.com/novapanel/novapanel/internal/models"
)

type AppService struct {
	pool      *pgxpool.Pool
	cryptoKey []byte
}

func NewAppService(pool *pgxpool.Pool, encryptionKey string) *AppService {
	return &AppService{pool: pool, cryptoKey: novacrypto.DeriveKey(encryptionKey)}
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
	var args []interface{}
	if role != "admin" {
		args = append(args, userID)
		baseWhere = " WHERE user_id = $1"
	}

	s.pool.QueryRow(ctx, "SELECT count(*) FROM applications"+baseWhere, args...).Scan(&total)

	rows, err := s.pool.Query(ctx,
		fmt.Sprintf(`SELECT id, user_id, domain_id, server_id, name, app_type, runtime, deploy_method,
		        COALESCE(git_repo,''), COALESCE(git_branch,'main'), status, created_at, updated_at
		 FROM applications%s ORDER BY created_at DESC LIMIT %d OFFSET %d`, baseWhere, perPage, offset), args...)
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
	var envVarsJSON []byte
	var envVarsEnc *string
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, domain_id, server_id, name, app_type, runtime, deploy_method,
		        COALESCE(git_repo,''), COALESCE(git_branch,'main'), status,
		        COALESCE(env_vars::text,'{}'), env_vars_enc, created_at, updated_at
		 FROM applications WHERE id = $1`, id,
	).Scan(&app.ID, &app.UserID, &app.DomainID, &app.ServerID, &app.Name, &app.AppType,
		&app.Runtime, &app.DeployMethod, &app.GitRepo, &app.GitBranch, &app.Status,
		&envVarsJSON, &envVarsEnc, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return nil, err
	}
	// Decrypt env vars if encrypted, otherwise use plain JSONB; values are masked
	if envVarsEnc != nil && *envVarsEnc != "" {
		if plain, err := novacrypto.Decrypt(*envVarsEnc, s.cryptoKey); err == nil {
			json.Unmarshal([]byte(plain), &app.EnvVars)
		}
	} else {
		json.Unmarshal(envVarsJSON, &app.EnvVars)
	}
	// Mask values — keys stay visible, values shown as ***
	masked := make(map[string]string, len(app.EnvVars))
	for k := range app.EnvVars {
		masked[k] = "***"
	}
	app.EnvVars = masked
	return app, nil
}

// GetByIDWithPlainEnv returns decrypted env vars — for internal deploy use only.
func (s *AppService) GetByIDWithPlainEnv(ctx context.Context, id string) (*models.Application, error) {
	app, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	var envVarsJSON []byte
	var envVarsEnc *string
	s.pool.QueryRow(ctx,
		`SELECT COALESCE(env_vars::text,'{}'), env_vars_enc FROM applications WHERE id = $1`, id,
	).Scan(&envVarsJSON, &envVarsEnc)
	if envVarsEnc != nil && *envVarsEnc != "" {
		if plain, err := novacrypto.Decrypt(*envVarsEnc, s.cryptoKey); err == nil {
			json.Unmarshal([]byte(plain), &app.EnvVars)
		}
	} else {
		json.Unmarshal(envVarsJSON, &app.EnvVars)
	}
	return app, nil
}

func (s *AppService) Update(ctx context.Context, id string, req models.UpdateAppRequest, userID uuid.UUID, role string) (*models.Application, error) {
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("application not found")
	}
	if role != "admin" && existing.UserID != userID {
		return nil, fmt.Errorf("application not found")
	}
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
		if enc, err := novacrypto.Encrypt(string(envJSON), s.cryptoKey); err == nil {
			setClauses = append(setClauses, fmt.Sprintf("env_vars_enc = $%d", argIdx))
			args = append(args, enc)
			argIdx++
		} else {
			setClauses = append(setClauses, fmt.Sprintf("env_vars = $%d", argIdx))
			args = append(args, envJSON)
			argIdx++
		}
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
	err = s.pool.QueryRow(ctx, query, args...).Scan(
		&app.ID, &app.UserID, &app.DomainID, &app.ServerID, &app.Name, &app.AppType,
		&app.Runtime, &app.DeployMethod, &app.GitRepo, &app.GitBranch, &app.Status,
		&app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update application: %w", err)
	}
	return app, nil
}

func (s *AppService) Delete(ctx context.Context, id string, userID uuid.UUID, role string) error {
	app, err := s.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("application not found")
	}
	if role != "admin" && app.UserID != userID {
		return fmt.Errorf("application not found")
	}
	result, err := s.pool.Exec(ctx, "DELETE FROM applications WHERE id = $1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("application not found")
	}
	return nil
}
