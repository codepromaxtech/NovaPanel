package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	novacrypto "github.com/novapanel/novapanel/internal/crypto"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/provisioner"
)

type DatabaseService struct {
	pool *pgxpool.Pool
}

func NewDatabaseService(pool *pgxpool.Pool) *DatabaseService {
	return &DatabaseService{pool: pool}
}

// getServerSSH gets SSH connection info for a server
func (s *DatabaseService) getServerSSH(ctx context.Context, serverID string) (provisioner.ServerInfo, error) {
	var server provisioner.ServerInfo
	var port int
	var encKey, encPassword string
	err := s.pool.QueryRow(ctx,
		`SELECT host(ip_address), port, ssh_user, COALESCE(ssh_key, ''), COALESCE(ssh_password, ''), COALESCE(auth_method, 'password'), COALESCE(is_local, FALSE)
		 FROM servers WHERE id = $1`, serverID,
	).Scan(&server.IPAddress, &port, &server.SSHUser, &encKey, &encPassword, &server.AuthMethod, &server.IsLocal)
	if err != nil {
		return server, err
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
	return server, nil
}

// provisionDatabase creates the actual database and user on the remote server via SSH
func (s *DatabaseService) provisionDatabase(server provisioner.ServerInfo, dbName, dbUser, dbPass, engine, charset string) error {
	var script string

	switch engine {
	case "mysql", "mariadb":
		script = fmt.Sprintf(`
echo "Creating MySQL/MariaDB database..."
mysql -u root -e "CREATE DATABASE IF NOT EXISTS %s CHARACTER SET %s;" 2>&1
if [ $? -ne 0 ]; then echo "DB_CREATE_FAILED"; exit 1; fi

echo "Creating database user..."
mysql -u root -e "CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY '%s';" 2>&1
mysql -u root -e "CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s';" 2>&1

echo "Granting privileges..."
mysql -u root -e "GRANT ALL PRIVILEGES ON %s.* TO '%s'@'localhost';" 2>&1
mysql -u root -e "GRANT ALL PRIVILEGES ON %s.* TO '%s'@'%%';" 2>&1
mysql -u root -e "FLUSH PRIVILEGES;" 2>&1

echo "DB_PROVISIONED"
`,
			sanitizeDBIdentifier(dbName), charset,
			dbUser, dbPass,
			dbUser, dbPass,
			sanitizeDBIdentifier(dbName), dbUser,
			sanitizeDBIdentifier(dbName), dbUser)

	case "postgresql", "postgres":
		script = fmt.Sprintf(`
echo "Creating PostgreSQL database..."
sudo -u postgres psql -c "SELECT 1 FROM pg_database WHERE datname = '%s'" | grep -q 1 || \
    sudo -u postgres psql -c "CREATE DATABASE \"%s\" ENCODING 'UTF8';" 2>&1
if [ $? -ne 0 ]; then echo "DB_CREATE_FAILED"; exit 1; fi

echo "Creating database user..."
sudo -u postgres psql -c "DO \$\$ BEGIN IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '%s') THEN CREATE USER \"%s\" WITH PASSWORD '%s'; END IF; END \$\$;" 2>&1

echo "Granting privileges..."
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE \"%s\" TO \"%s\";" 2>&1

echo "DB_PROVISIONED"
`,
			dbName, dbName,
			dbUser, dbUser, dbPass,
			dbName, dbUser)

	default:
		return fmt.Errorf("unsupported database engine: %s", engine)
	}

	output, err := provisioner.RunScript(server, script)
	if err != nil {
		return fmt.Errorf("database provisioning failed: %w\nOutput: %s", err, output)
	}

	if strings.Contains(output, "DB_CREATE_FAILED") {
		return fmt.Errorf("database creation failed on server: %s", output)
	}

	if !strings.Contains(output, "DB_PROVISIONED") {
		return fmt.Errorf("database provisioning incomplete: %s", output)
	}

	return nil
}

// deprovisionDatabase drops the database and user on the remote server via SSH
func (s *DatabaseService) deprovisionDatabase(server provisioner.ServerInfo, dbName, dbUser, engine string) error {
	var script string

	switch engine {
	case "mysql", "mariadb":
		script = fmt.Sprintf(`
echo "Dropping MySQL/MariaDB database..."
mysql -u root -e "DROP DATABASE IF EXISTS %s;" 2>&1
mysql -u root -e "DROP USER IF EXISTS '%s'@'localhost';" 2>&1
mysql -u root -e "DROP USER IF EXISTS '%s'@'%%';" 2>&1
mysql -u root -e "FLUSH PRIVILEGES;" 2>&1
echo "DB_DROPPED"
`, sanitizeDBIdentifier(dbName), dbUser, dbUser)

	case "postgresql", "postgres":
		script = fmt.Sprintf(`
echo "Dropping PostgreSQL database..."
sudo -u postgres psql -c "DROP DATABASE IF EXISTS \"%s\";" 2>&1
sudo -u postgres psql -c "DROP USER IF EXISTS \"%s\";" 2>&1
echo "DB_DROPPED"
`, dbName, dbUser)

	default:
		return fmt.Errorf("unsupported engine for deprovisioning: %s", engine)
	}

	output, err := provisioner.RunScript(server, script)
	if err != nil {
		return fmt.Errorf("database deprovisioning failed: %w\nOutput: %s", err, output)
	}

	return nil
}

// sanitizeDBIdentifier wraps a name with backticks for MySQL to prevent SQL injection
func sanitizeDBIdentifier(name string) string {
	// Remove any backticks from the name itself
	clean := strings.ReplaceAll(name, "`", "")
	return fmt.Sprintf("`%s`", clean)
}

func (s *DatabaseService) Create(ctx context.Context, userID uuid.UUID, req models.CreateDatabaseRequest) (*models.Database, error) {
	engine := req.Engine
	if engine == "" {
		engine = "mysql"
	}
	charset := req.Charset
	if charset == "" {
		charset = "utf8mb4"
	}

	dbUser := req.Name + "_user"
	dbPass := uuid.New().String()[:16]

	// Encrypt the DB password before storage
	dbPassStored := dbPass
	if cryptoKey, kerr := novacrypto.GetEncryptionKey(); kerr == nil {
		if enc, eerr := novacrypto.Encrypt(dbPass, cryptoKey); eerr == nil {
			dbPassStored = enc
		}
	}

	var serverID *uuid.UUID
	if req.ServerID != "" {
		parsed, err := uuid.Parse(req.ServerID)
		if err != nil {
			return nil, fmt.Errorf("invalid server_id")
		}
		serverID = &parsed
	}

	db := &models.Database{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO databases (user_id, server_id, name, engine, db_user, db_password_enc, charset, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'provisioning')
		 RETURNING id, user_id, server_id, name, engine, db_user, charset, size_mb, status, created_at, updated_at`,
		userID, serverID, req.Name, engine, dbUser, dbPassStored, charset,
	).Scan(&db.ID, &db.UserID, &db.ServerID, &db.Name, &db.Engine, &db.DBUser, &db.Charset, &db.SizeMB, &db.Status, &db.CreatedAt, &db.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// ── Provision on the remote server (async) ──
	if serverID != nil {
		go func() {
			bgCtx := context.Background()
			server, err := s.getServerSSH(bgCtx, serverID.String())
			if err != nil {
				log.Printf("❌ Database %s: failed to get server SSH info: %v", req.Name, err)
				s.updateStatus(bgCtx, db.ID.String(), "error")
				return
			}

			if err := s.provisionDatabase(server, req.Name, dbUser, dbPass, engine, charset); err != nil {
				log.Printf("❌ Database %s: provisioning failed: %v", req.Name, err)
				s.updateStatus(bgCtx, db.ID.String(), "error")
				return
			}

			log.Printf("✅ Database %s: provisioned successfully on server", req.Name)
			s.updateStatus(bgCtx, db.ID.String(), "active")
		}()
	} else {
		// No server attached, just mark as active (local management)
		s.updateStatus(ctx, db.ID.String(), "active")
	}

	return db, nil
}

func (s *DatabaseService) updateStatus(ctx context.Context, id, status string) {
	s.pool.Exec(ctx, "UPDATE databases SET status = $1, updated_at = $2 WHERE id = $3", status, time.Now(), id)
}

func (s *DatabaseService) List(ctx context.Context, userID uuid.UUID, role string, page, perPage int) (*models.PaginatedResponse, error) {
	offset := (page - 1) * perPage
	var total int64

	query := `SELECT id, user_id, server_id, name, engine, db_user, charset, size_mb, status, created_at, updated_at FROM databases`
	countQuery := `SELECT count(*) FROM databases`
	var args []interface{}

	if role != "admin" {
		args = append(args, userID)
		query += ` WHERE user_id = $1`
		countQuery += ` WHERE user_id = $1`
	}

	s.pool.QueryRow(ctx, countQuery, args...).Scan(&total)

	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT %d OFFSET %d`, perPage, offset)
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []models.Database
	for rows.Next() {
		var db models.Database
		if err := rows.Scan(&db.ID, &db.UserID, &db.ServerID, &db.Name, &db.Engine, &db.DBUser, &db.Charset, &db.SizeMB, &db.Status, &db.CreatedAt, &db.UpdatedAt); err != nil {
			continue
		}
		databases = append(databases, db)
	}

	return &models.PaginatedResponse{
		Data:       databases,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}, nil
}

func (s *DatabaseService) Delete(ctx context.Context, id string, userID uuid.UUID, role string) error {
	// Fetch database details for deprovisioning and ownership check
	var dbName, dbUser, engine string
	var serverID *uuid.UUID
	var ownerID uuid.UUID
	err := s.pool.QueryRow(ctx,
		`SELECT name, db_user, engine, server_id, user_id FROM databases WHERE id = $1`, id,
	).Scan(&dbName, &dbUser, &engine, &serverID, &ownerID)
	if err != nil {
		return fmt.Errorf("database not found")
	}
	if role != "admin" && ownerID != userID {
		return fmt.Errorf("database not found")
	}

	// Deprovision from server if applicable
	if serverID != nil {
		server, err := s.getServerSSH(ctx, serverID.String())
		if err != nil {
			log.Printf("⚠️  Database %s: could not get server SSH info for deprovisioning: %v", dbName, err)
		} else {
			if err := s.deprovisionDatabase(server, dbName, dbUser, engine); err != nil {
				log.Printf("⚠️  Database %s: deprovisioning warning: %v", dbName, err)
				// Continue with DB record deletion even if SSH deprovisioning fails
			} else {
				log.Printf("✅ Database %s: dropped from server", dbName)
			}
		}
	}

	_, err = s.pool.Exec(ctx, `DELETE FROM databases WHERE id = $1`, id)
	return err
}
