package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	novacrypto "github.com/novapanel/novapanel/internal/crypto"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/pquerna/otp/totp"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/novapanel/novapanel/internal/config"
)

const revokedTokensKey = "novapanel:revoked_tokens"

type AuthService struct {
	db  *pgxpool.Pool
	cfg *config.Config
	rdb *redis.Client
}

func NewAuthService(db *pgxpool.Pool, cfg *config.Config, rdb *redis.Client) *AuthService {
	return &AuthService{db: db, cfg: cfg, rdb: rdb}
}

func (s *AuthService) Register(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error) {
	var exists bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("user with this email already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

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

	token, jti, expiresAt, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}
	go s.recordSession(user.ID, jti, expiresAt, "", "")

	return &models.AuthResponse{Token: token, ExpiresAt: expiresAt.Unix(), User: user}, nil
}

func (s *AuthService) Login(ctx context.Context, req models.LoginRequest, ip, ua string) (*models.AuthResponse, error) {
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

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	if user.TwoFactorEnabled {
		return nil, errors.New("two_factor_required")
	}

	now := time.Now()
	s.db.Exec(ctx, "UPDATE users SET last_login_at = $1 WHERE id = $2", now, user.ID)
	user.LastLoginAt = &now

	token, jti, expiresAt, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}
	go s.recordSession(user.ID, jti, expiresAt, ip, ua)

	s.db.Exec(ctx,
		`INSERT INTO audit_logs (user_id, action, resource, details)
		 VALUES ($1, 'login', 'user', '{}')`, user.ID)

	return &models.AuthResponse{Token: token, ExpiresAt: expiresAt.Unix(), User: user}, nil
}

// RevokeToken adds a token JTI to the Redis revocation set and removes the session row.
func (s *AuthService) RevokeToken(ctx context.Context, tokenID string, _ time.Duration) error {
	if tokenID != "" {
		if s.rdb != nil {
			s.rdb.SAdd(ctx, revokedTokensKey, tokenID)
		}
		s.db.Exec(ctx, `DELETE FROM user_sessions WHERE token_jti = $1`, tokenID)
	}
	return nil
}

// Refresh issues a new JWT for an already-authenticated user.
func (s *AuthService) Refresh(ctx context.Context, userID, ip, ua string) (*models.AuthResponse, error) {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	if user.Status != "active" {
		return nil, errors.New("account is not active")
	}
	token, jti, expiresAt, err := s.generateToken(*user)
	if err != nil {
		return nil, err
	}
	go s.recordSession(user.ID, jti, expiresAt, ip, ua)
	return &models.AuthResponse{Token: token, ExpiresAt: expiresAt.Unix(), User: *user}, nil
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

// ─────────────────────────────────────────────
//  2FA / TOTP
// ─────────────────────────────────────────────

type TOTPSetupResponse struct {
	Secret string `json:"secret"`
	QRURI  string `json:"qr_uri"`
}

func (s *AuthService) TOTPSetup(ctx context.Context, userID string) (*TOTPSetupResponse, error) {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "NovaPanel",
		AccountName: user.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	// Encrypt secret before storing
	cryptoKey := novacrypto.DeriveKey(s.cfg.EncryptionKey)
	encSecret, err := novacrypto.Encrypt(key.Secret(), cryptoKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt TOTP secret: %w", err)
	}

	_, err = s.db.Exec(ctx,
		`UPDATE users SET totp_secret = $1 WHERE id = $2`, encSecret, userID)
	if err != nil {
		return nil, err
	}

	return &TOTPSetupResponse{Secret: key.Secret(), QRURI: key.URL()}, nil
}

func (s *AuthService) TOTPEnable(ctx context.Context, userID, code string) ([]string, error) {
	var encSecret string
	err := s.db.QueryRow(ctx,
		`SELECT COALESCE(totp_secret, '') FROM users WHERE id = $1`, userID,
	).Scan(&encSecret)
	if err != nil || encSecret == "" {
		return nil, errors.New("TOTP not set up — call /auth/2fa/setup first")
	}

	cryptoKey := novacrypto.DeriveKey(s.cfg.EncryptionKey)
	secretStr, err := novacrypto.Decrypt(encSecret, cryptoKey)
	if err != nil {
		return nil, errors.New("failed to decrypt TOTP secret")
	}

	if !totp.Validate(code, secretStr) {
		return nil, errors.New("invalid TOTP code")
	}

	// Generate 10 backup codes
	plainCodes := make([]string, 10)
	hashedCodes := make([]string, 10)
	for i := range plainCodes {
		c, err := randomAlphanumeric(8)
		if err != nil {
			return nil, err
		}
		plainCodes[i] = c
		h, _ := bcrypt.GenerateFromPassword([]byte(c), bcrypt.MinCost)
		hashedCodes[i] = string(h)
	}

	_, err = s.db.Exec(ctx,
		`UPDATE users SET two_factor_enabled = true, totp_backup_codes = $1 WHERE id = $2`,
		hashedCodes, userID)
	if err != nil {
		return nil, err
	}
	return plainCodes, nil
}

// TOTPVerify is called after a login that returned "two_factor_required".
func (s *AuthService) TOTPVerify(ctx context.Context, email, code, ip, ua string) (*models.AuthResponse, error) {
	user := models.User{}
	var encSecret string
	var backupCodes []string
	err := s.db.QueryRow(ctx,
		`SELECT id, email, first_name, last_name, role, status,
		        two_factor_enabled, last_login_at, created_at, updated_at,
		        COALESCE(totp_secret, ''), COALESCE(totp_backup_codes, '{}')
		 FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.Role, &user.Status,
		&user.TwoFactorEnabled, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
		&encSecret, &backupCodes)
	if err != nil {
		return nil, errors.New("user not found")
	}
	if !user.TwoFactorEnabled {
		return nil, errors.New("2FA is not enabled for this account")
	}
	if user.Status != "active" {
		return nil, errors.New("account is not active")
	}

	// Try TOTP code
	cryptoKey := novacrypto.DeriveKey(s.cfg.EncryptionKey)
	secretStr, decErr := novacrypto.Decrypt(encSecret, cryptoKey)
	validTOTP := decErr == nil && totp.Validate(code, secretStr)

	// Try backup codes
	usedBackupIdx := -1
	if !validTOTP {
		for i, hashed := range backupCodes {
			if bcrypt.CompareHashAndPassword([]byte(hashed), []byte(code)) == nil {
				usedBackupIdx = i
				break
			}
		}
	}

	if !validTOTP && usedBackupIdx < 0 {
		return nil, errors.New("invalid verification code")
	}

	// Consume backup code if used
	if usedBackupIdx >= 0 {
		newCodes := append(backupCodes[:usedBackupIdx], backupCodes[usedBackupIdx+1:]...)
		s.db.Exec(ctx, `UPDATE users SET totp_backup_codes = $1 WHERE id = $2`, newCodes, user.ID)
	}

	now := time.Now()
	s.db.Exec(ctx, "UPDATE users SET last_login_at = $1 WHERE id = $2", now, user.ID)
	user.LastLoginAt = &now

	token, jti, expiresAt, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}
	go s.recordSession(user.ID, jti, expiresAt, ip, ua)

	return &models.AuthResponse{Token: token, ExpiresAt: expiresAt.Unix(), User: user}, nil
}

func (s *AuthService) TOTPDisable(ctx context.Context, userID, currentPassword string) error {
	var hash string
	err := s.db.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash)
	if err != nil {
		return errors.New("user not found")
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(currentPassword)) != nil {
		return errors.New("incorrect password")
	}
	_, err = s.db.Exec(ctx,
		`UPDATE users SET two_factor_enabled = false, totp_secret = NULL, totp_backup_codes = NULL
		 WHERE id = $1`, userID)
	return err
}

// ─────────────────────────────────────────────
//  Password Reset
// ─────────────────────────────────────────────

func (s *AuthService) CreatePasswordReset(ctx context.Context, email string) (rawToken string, userEmail string, err error) {
	var userID uuid.UUID
	err = s.db.QueryRow(ctx,
		`SELECT id, email FROM users WHERE email = $1 AND status = 'active'`, email,
	).Scan(&userID, &userEmail)
	if err != nil {
		// Don't reveal whether email exists
		return "", email, nil
	}

	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	rawToken = hex.EncodeToString(b)
	tokenHash := sha256Token(rawToken)

	_, err = s.db.Exec(ctx,
		`INSERT INTO password_reset_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, NOW() + INTERVAL '15 minutes')`,
		userID, tokenHash)
	return rawToken, userEmail, err
}

func (s *AuthService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	tokenHash := sha256Token(rawToken)
	var tokenID, userID string
	err := s.db.QueryRow(ctx,
		`SELECT id, user_id FROM password_reset_tokens
		 WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()`,
		tokenHash,
	).Scan(&tokenID, &userID)
	if err != nil {
		return errors.New("invalid or expired reset token")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update password and mark token used in a transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err = tx.Exec(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, string(hash), userID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1`, tokenID); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}

	// Revoke all existing sessions for this user
	go s.revokeAllSessions(context.Background(), userID)
	return nil
}

// ─────────────────────────────────────────────
//  Session Management
// ─────────────────────────────────────────────

func (s *AuthService) ListSessions(ctx context.Context, userID string) ([]models.UserSession, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, host(ip_address), user_agent, last_seen_at, expires_at, created_at
		 FROM user_sessions
		 WHERE user_id = $1 AND expires_at > NOW()
		 ORDER BY last_seen_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.UserSession
	for rows.Next() {
		var sess models.UserSession
		var ip, ua *string
		if err := rows.Scan(&sess.ID, &sess.UserID, &ip, &ua, &sess.LastSeenAt, &sess.ExpiresAt, &sess.CreatedAt); err != nil {
			continue
		}
		if ip != nil {
			sess.IPAddress = *ip
		}
		if ua != nil {
			sess.UserAgent = *ua
		}
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

func (s *AuthService) RevokeSession(ctx context.Context, sessionID, userID string) error {
	var jti string
	err := s.db.QueryRow(ctx,
		`SELECT token_jti FROM user_sessions WHERE id = $1 AND user_id = $2`,
		sessionID, userID,
	).Scan(&jti)
	if err != nil {
		return errors.New("session not found")
	}

	if s.rdb != nil && jti != "" {
		s.rdb.SAdd(ctx, revokedTokensKey, jti)
	}
	s.db.Exec(ctx, `DELETE FROM user_sessions WHERE id = $1`, sessionID)
	return nil
}

func (s *AuthService) RevokeAllOtherSessions(ctx context.Context, currentJTI, userID string) error {
	rows, err := s.db.Query(ctx,
		`SELECT id, token_jti FROM user_sessions WHERE user_id = $1 AND token_jti != $2`,
		userID, currentJTI)
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []string
	var jtis []string
	for rows.Next() {
		var id, jti string
		if rows.Scan(&id, &jti) == nil {
			ids = append(ids, id)
			if jti != "" {
				jtis = append(jtis, jti)
			}
		}
	}

	if s.rdb != nil && len(jtis) > 0 {
		args := make([]interface{}, len(jtis))
		for i, j := range jtis {
			args[i] = j
		}
		s.rdb.SAdd(ctx, revokedTokensKey, args...)
	}
	s.db.Exec(ctx, `DELETE FROM user_sessions WHERE user_id = $1 AND token_jti != $2`, userID, currentJTI)
	return nil
}

func (s *AuthService) revokeAllSessions(ctx context.Context, userID string) {
	rows, _ := s.db.Query(ctx, `SELECT token_jti FROM user_sessions WHERE user_id = $1`, userID)
	if rows == nil {
		return
	}
	defer rows.Close()
	var jtis []interface{}
	for rows.Next() {
		var jti string
		if rows.Scan(&jti) == nil && jti != "" {
			jtis = append(jtis, jti)
		}
	}
	if s.rdb != nil && len(jtis) > 0 {
		s.rdb.SAdd(ctx, revokedTokensKey, jtis...)
	}
	s.db.Exec(ctx, `DELETE FROM user_sessions WHERE user_id = $1`, userID)
}

func (s *AuthService) recordSession(userID uuid.UUID, jti string, expiresAt time.Time, ip, ua string) {
	ctx := context.Background()
	var ipVal interface{} = nil
	if ip != "" {
		ipVal = ip
	}
	s.db.Exec(ctx,
		`INSERT INTO user_sessions (user_id, token_jti, ip_address, user_agent, expires_at)
		 VALUES ($1, $2, $3::inet, $4, $5)
		 ON CONFLICT (token_jti) DO NOTHING`,
		userID, jti, ipVal, ua, expiresAt)
}

// ─────────────────────────────────────────────
//  Internal helpers
// ─────────────────────────────────────────────

// generateToken returns (tokenString, jti, expiresAt, error)
func (s *AuthService) generateToken(user models.User) (string, string, time.Time, error) {
	expiresAt := time.Now().Add(s.cfg.JWTExpiry)
	jti := uuid.New().String()
	claims := models.Claims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "novapanel",
			Subject:   user.ID.String(),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", "", time.Time{}, err
	}
	return tokenStr, jti, expiresAt, nil
}

func sha256Token(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

const backupCodeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func randomAlphanumeric(n int) (string, error) {
	b := make([]byte, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(backupCodeChars))))
		if err != nil {
			return "", err
		}
		b[i] = backupCodeChars[idx.Int64()]
	}
	return string(b), nil
}
