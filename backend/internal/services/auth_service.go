package services

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/config"
	"github.com/novapanel/novapanel/internal/middleware"
	"github.com/novapanel/novapanel/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db  *pgxpool.Pool
	cfg *config.Config
}

func NewAuthService(db *pgxpool.Pool, cfg *config.Config) *AuthService {
	return &AuthService{db: db, cfg: cfg}
}

func (s *AuthService) Register(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error) {
	// Check if user exists
	var exists bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("user with this email already exists")
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Insert user
	user := models.User{}
	err = s.db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, first_name, last_name, role, status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, email, first_name, last_name, role, status, created_at, updated_at`,
		req.Email, string(hash), req.FirstName, req.LastName, "client", "active",
	).Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.Role, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Generate token
	token, expiresAt, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	user := models.User{}
	err := s.db.QueryRow(ctx,
		`SELECT id, email, password_hash, first_name, last_name, role, status,
		        two_factor_enabled, last_login_at, created_at, updated_at
		 FROM users WHERE email = $1`,
		req.Email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName,
		&user.Role, &user.Status, &user.TwoFactorEnabled, &user.LastLoginAt,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	if user.Status != "active" {
		return nil, errors.New("account is not active")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Update last login
	now := time.Now()
	s.db.Exec(ctx, "UPDATE users SET last_login_at = $1 WHERE id = $2", now, user.ID)
	user.LastLoginAt = &now

	// Generate token
	token, expiresAt, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	}, nil
}

func (s *AuthService) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRow(ctx,
		`SELECT id, email, first_name, last_name, role, status,
		        two_factor_enabled, last_login_at, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName,
		&user.Role, &user.Status, &user.TwoFactorEnabled, &user.LastLoginAt,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (s *AuthService) generateToken(user models.User) (string, int64, error) {
	expiresAt := time.Now().Add(s.cfg.JWTExpiry)
	claims := middleware.Claims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "novapanel",
			Subject:   user.ID.String(),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", 0, err
	}

	return tokenStr, expiresAt.Unix(), nil
}
