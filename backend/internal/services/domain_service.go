package services

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
)

type DomainService struct {
	db *pgxpool.Pool
}

func NewDomainService(db *pgxpool.Pool) *DomainService {
	return &DomainService{db: db}
}

func (s *DomainService) Create(ctx context.Context, userID string, req models.CreateDomainRequest) (*models.Domain, error) {
	// Validate domain name
	req.Name = strings.ToLower(strings.TrimSpace(req.Name))
	if req.Name == "" {
		return nil, fmt.Errorf("domain name is required")
	}

	// Check if domain already exists
	var exists bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM domains WHERE name = $1)", req.Name).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("domain %s already exists", req.Name)
	}

	// Defaults
	domainType := "primary"
	if req.Type != "" {
		domainType = req.Type
	}
	webServer := "nginx"
	if req.WebServer != "" {
		webServer = req.WebServer
	}
	phpVersion := "8.2"
	if req.PHPVersion != "" {
		phpVersion = req.PHPVersion
	}
	docRoot := "/var/www/" + req.Name
	if req.DocumentRoot != "" {
		docRoot = req.DocumentRoot
	}

	// Begin transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var serverID *uuid.UUID
	if !req.IsLoadBalancer && req.ServerID != "" {
		id, err := uuid.Parse(req.ServerID)
		if err != nil {
			return nil, fmt.Errorf("invalid server_id")
		}
		serverID = &id
	}

	uid, _ := uuid.Parse(userID)
	domain := &models.Domain{}
	err = tx.QueryRow(ctx,
		`INSERT INTO domains (user_id, server_id, name, type, document_root, web_server, php_version, status, is_load_balancer)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, user_id, server_id, name, type, document_root, web_server, php_version, ssl_enabled, status, is_load_balancer, created_at, updated_at`,
		uid, serverID, req.Name, domainType, docRoot, webServer, phpVersion, "active", req.IsLoadBalancer,
	).Scan(&domain.ID, &domain.UserID, &domain.ServerID, &domain.Name, &domain.Type,
		&domain.DocumentRoot, &domain.WebServer, &domain.PHPVersion, &domain.SSLEnabled,
		&domain.Status, &domain.IsLoadBalancer, &domain.CreatedAt, &domain.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Insert load balancer backend relationships
	if req.IsLoadBalancer && len(req.BackendServerIDs) > 0 {
		for _, bID := range req.BackendServerIDs {
			backendUUID, err := uuid.Parse(bID)
			if err != nil {
				continue
			}
			_, err = tx.Exec(ctx, "INSERT INTO domain_backend_servers (domain_id, server_id) VALUES ($1, $2)", domain.ID, backendUUID)
			if err == nil {
				domain.BackendServerIDs = append(domain.BackendServerIDs, backendUUID)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return domain, nil
}

func (s *DomainService) GetByID(ctx context.Context, id string) (*models.Domain, error) {
	domain := &models.Domain{}
	err := s.db.QueryRow(ctx,
		`SELECT id, user_id, server_id, name, type, document_root, web_server,
		        php_version, ssl_enabled, status, is_load_balancer, created_at, updated_at
		 FROM domains WHERE id = $1`,
		id,
	).Scan(&domain.ID, &domain.UserID, &domain.ServerID, &domain.Name, &domain.Type,
		&domain.DocumentRoot, &domain.WebServer, &domain.PHPVersion, &domain.SSLEnabled,
		&domain.Status, &domain.IsLoadBalancer, &domain.CreatedAt, &domain.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("domain not found")
	}

	if domain.IsLoadBalancer {
		rows, _ := s.db.Query(ctx, "SELECT server_id FROM domain_backend_servers WHERE domain_id = $1", domain.ID)
		defer rows.Close()
		for rows.Next() {
			var svrID uuid.UUID
			if err := rows.Scan(&svrID); err == nil {
				domain.BackendServerIDs = append(domain.BackendServerIDs, svrID)
			}
		}
	}

	return domain, nil
}

func (s *DomainService) List(ctx context.Context, userID, role string, page, perPage int) (*models.PaginatedResponse, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var (
		total   int64
		domains []models.Domain
	)

	// Count query
	countQuery := "SELECT COUNT(*) FROM domains"
	listQuery := `SELECT id, user_id, server_id, name, type, document_root, web_server,
	              php_version, ssl_enabled, status, is_load_balancer, created_at, updated_at FROM domains`

	if role != "admin" {
		countQuery += " WHERE user_id = $1"
		listQuery += " WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3"
		s.db.QueryRow(ctx, countQuery, userID).Scan(&total)
		rows, err := s.db.Query(ctx, listQuery, userID, perPage, offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var d models.Domain
			err := rows.Scan(&d.ID, &d.UserID, &d.ServerID, &d.Name, &d.Type,
				&d.DocumentRoot, &d.WebServer, &d.PHPVersion, &d.SSLEnabled,
				&d.Status, &d.IsLoadBalancer, &d.CreatedAt, &d.UpdatedAt)
			if err != nil {
				return nil, err
			}
			domains = append(domains, d)
		}
	} else {
		listQuery += " ORDER BY created_at DESC LIMIT $1 OFFSET $2"
		s.db.QueryRow(ctx, countQuery).Scan(&total)
		rows, err := s.db.Query(ctx, listQuery, perPage, offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var d models.Domain
			err := rows.Scan(&d.ID, &d.UserID, &d.ServerID, &d.Name, &d.Type,
				&d.DocumentRoot, &d.WebServer, &d.PHPVersion, &d.SSLEnabled,
				&d.Status, &d.IsLoadBalancer, &d.CreatedAt, &d.UpdatedAt)
			if err != nil {
				return nil, err
			}
			domains = append(domains, d)
		}
	}

	// Fetch backend arrays for load balancer domains
	var lbIDs []uuid.UUID
	for _, d := range domains {
		if d.IsLoadBalancer {
			lbIDs = append(lbIDs, d.ID)
		}
	}

	if len(lbIDs) > 0 {
		rows, err := s.db.Query(ctx, "SELECT domain_id, server_id FROM domain_backend_servers WHERE domain_id = ANY($1)", lbIDs)
		if err == nil {
			defer rows.Close()
			lbMap := make(map[uuid.UUID][]uuid.UUID)
			for rows.Next() {
				var dID, sID uuid.UUID
				if err := rows.Scan(&dID, &sID); err == nil {
					lbMap[dID] = append(lbMap[dID], sID)
				}
			}
			for i, d := range domains {
				if d.IsLoadBalancer {
					domains[i].BackendServerIDs = lbMap[d.ID]
				}
			}
		}
	}

	if domains == nil {
		domains = []models.Domain{}
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))

	return &models.PaginatedResponse{
		Data:       domains,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

func (s *DomainService) Update(ctx context.Context, id string, req models.UpdateDomainRequest) (*models.Domain, error) {
	// Build dynamic update
	sets := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.WebServer != "" {
		sets = append(sets, fmt.Sprintf("web_server = $%d", argIdx))
		args = append(args, req.WebServer)
		argIdx++
	}
	if req.PHPVersion != "" {
		sets = append(sets, fmt.Sprintf("php_version = $%d", argIdx))
		args = append(args, req.PHPVersion)
		argIdx++
	}
	if req.DocumentRoot != "" {
		sets = append(sets, fmt.Sprintf("document_root = $%d", argIdx))
		args = append(args, req.DocumentRoot)
		argIdx++
	}
	if req.SSLEnabled != nil {
		sets = append(sets, fmt.Sprintf("ssl_enabled = $%d", argIdx))
		args = append(args, *req.SSLEnabled)
		argIdx++
	}
	if req.Status != "" {
		sets = append(sets, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, req.Status)
		argIdx++
	}

	if len(sets) == 0 {
		return s.GetByID(ctx, id)
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now())
	argIdx++

	args = append(args, id)
	query := fmt.Sprintf(
		`UPDATE domains SET %s WHERE id = $%d
		 RETURNING id, user_id, server_id, name, type, document_root, web_server, php_version, ssl_enabled, status, is_load_balancer, created_at, updated_at`,
		strings.Join(sets, ", "), argIdx,
	)

	domain := &models.Domain{}
	err := s.db.QueryRow(ctx, query, args...).Scan(
		&domain.ID, &domain.UserID, &domain.ServerID, &domain.Name, &domain.Type,
		&domain.DocumentRoot, &domain.WebServer, &domain.PHPVersion, &domain.SSLEnabled,
		&domain.Status, &domain.IsLoadBalancer, &domain.CreatedAt, &domain.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("domain not found")
	}
	return domain, nil
}

func (s *DomainService) Delete(ctx context.Context, id string) error {
	result, err := s.db.Exec(ctx, "DELETE FROM domains WHERE id = $1", id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("domain not found")
	}
	return nil
}
