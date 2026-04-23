package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
)

type APIKeyService struct {
	pool *pgxpool.Pool
}

func NewAPIKeyService(pool *pgxpool.Pool) *APIKeyService {
	return &APIKeyService{pool: pool}
}

func generateRawKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "np_" + hex.EncodeToString(b), nil
}

func hashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

type CreateAPIKeyRequest struct {
	Name      string     `json:"name" binding:"required"`
	ExpiresAt *time.Time `json:"expires_at"`
	Scopes    []string   `json:"scopes"`
}

type APIKeyCreated struct {
	models.APIKey
	RawKey string `json:"key"` // returned once only
}

func (s *APIKeyService) Create(ctx context.Context, userID uuid.UUID, req CreateAPIKeyRequest) (*APIKeyCreated, error) {
	raw, err := generateRawKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	hash := hashKey(raw)
	prefix := raw[:12]

	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{}
	}

	k := &models.APIKey{}
	err = s.pool.QueryRow(ctx,
		`INSERT INTO api_keys (user_id, name, key_hash, key_prefix, expires_at, scopes)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, name, key_prefix, is_active, scopes, created_at`,
		userID, req.Name, hash, prefix, req.ExpiresAt, scopes,
	).Scan(&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.IsActive, &k.Scopes, &k.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}
	k.ExpiresAt = req.ExpiresAt
	return &APIKeyCreated{APIKey: *k, RawKey: raw}, nil
}

func (s *APIKeyService) List(ctx context.Context, userID uuid.UUID) ([]models.APIKey, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, name, key_prefix, last_used_at, expires_at, is_active, scopes, created_at
		 FROM api_keys WHERE user_id = $1 AND is_active = true ORDER BY created_at DESC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []models.APIKey
	for rows.Next() {
		var k models.APIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.KeyPrefix, &k.LastUsedAt,
			&k.ExpiresAt, &k.IsActive, &k.Scopes, &k.CreatedAt); err != nil {
			continue
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *APIKeyService) Revoke(ctx context.Context, id string, userID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE api_keys SET is_active = false WHERE id = $1 AND user_id = $2`,
		id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("API key not found")
	}
	return nil
}

// ValidateKey looks up a raw key and returns the owner user. Returns error if key is invalid/expired/revoked.
func (s *APIKeyService) ValidateKey(ctx context.Context, rawKey string) (*models.User, error) {
	hash := hashKey(rawKey)
	var userID uuid.UUID
	var expiresAt *time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, expires_at FROM api_keys
		 WHERE key_hash = $1 AND is_active = true`, hash,
	).Scan(&userID, &expiresAt)
	if err != nil {
		return nil, errors.New("invalid API key")
	}
	if expiresAt != nil && time.Now().After(*expiresAt) {
		return nil, errors.New("API key has expired")
	}

	// Update last_used_at asynchronously
	go s.pool.Exec(context.Background(),
		`UPDATE api_keys SET last_used_at = NOW() WHERE key_hash = $1`, hash)

	user := &models.User{}
	err = s.pool.QueryRow(ctx,
		`SELECT id, email, first_name, last_name, role, status,
		        two_factor_enabled, last_login_at, created_at, updated_at
		 FROM users WHERE id = $1`, userID,
	).Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.Role,
		&user.Status, &user.TwoFactorEnabled, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, errors.New("user not found")
	}
	if user.Status != "active" {
		return nil, errors.New("account is not active")
	}
	return user, nil
}
