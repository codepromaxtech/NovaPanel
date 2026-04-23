package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	novacrypto "github.com/novapanel/novapanel/internal/crypto"
	"github.com/novapanel/novapanel/internal/provisioner"
)

// DBManagerService handles web-based DB management tools and SQL query execution
type DBManagerService struct {
	pool *pgxpool.Pool
}

func NewDBManagerService(pool *pgxpool.Pool) *DBManagerService {
	return &DBManagerService{pool: pool}
}

// getServerSSH gets decrypted SSH connection info for a server
func (s *DBManagerService) getServerSSH(ctx context.Context, serverID string) (provisioner.ServerInfo, error) {
	var server provisioner.ServerInfo
	var port int
	var encKey, encPassword string
	err := s.pool.QueryRow(ctx,
		`SELECT host(ip_address), port, ssh_user, COALESCE(ssh_key, ''), COALESCE(ssh_password, ''), COALESCE(auth_method, 'password')
		 FROM servers WHERE id = $1`, serverID,
	).Scan(&server.IPAddress, &port, &server.SSHUser, &encKey, &encPassword, &server.AuthMethod)
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

// RunQuery executes a SQL query on a remote server database via SSH
func (s *DBManagerService) RunQuery(ctx context.Context, serverID, engine, dbName, query string) (string, error) {
	server, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		return "", fmt.Errorf("server not found: %w", err)
	}

	// Block destructive DDL operations in the ad-hoc query runner.
	upperQ := strings.ToUpper(strings.TrimSpace(query))
	blocked := []string{
		"DROP DATABASE", "DROP USER", "DROP TABLE", "DROP SCHEMA",
		"TRUNCATE TABLE", "TRUNCATE ",
	}
	for _, b := range blocked {
		if strings.HasPrefix(upperQ, b) {
			return "", fmt.Errorf("dangerous operation blocked — use the panel to manage databases")
		}
	}

	var cmd string
	switch engine {
	case "mysql", "mariadb":
		escapedQuery := strings.ReplaceAll(query, "'", "'\\''")
		cmd = fmt.Sprintf("mysql -u root '%s' -e '%s' 2>&1 | head -200", dbName, escapedQuery)
	case "postgresql", "postgres":
		escapedQuery := strings.ReplaceAll(query, "'", "'\\''")
		cmd = fmt.Sprintf("sudo -u postgres psql -d '%s' -c '%s' 2>&1 | head -200", dbName, escapedQuery)
	case "mongodb", "mongo":
		escapedQuery := strings.ReplaceAll(query, "'", "'\\''")
		cmd = fmt.Sprintf("mongosh '%s' --eval '%s' --quiet 2>&1 | head -200", dbName, escapedQuery)
	case "redis":
		cmd = fmt.Sprintf("redis-cli %s 2>&1 | head -200", query)
	default:
		return "", fmt.Errorf("unsupported engine: %s", engine)
	}

	output, err := provisioner.RunScript(server, cmd)
	if err != nil {
		return output, fmt.Errorf("query failed: %w", err)
	}
	return output, nil
}

// ListTables lists all tables/collections in a database on a remote server
func (s *DBManagerService) ListTables(ctx context.Context, serverID, engine, dbName string) (string, error) {
	server, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		return "", err
	}

	var cmd string
	switch engine {
	case "mysql", "mariadb":
		cmd = fmt.Sprintf("mysql -u root '%s' -e 'SHOW TABLES;' 2>&1", dbName)
	case "postgresql", "postgres":
		cmd = fmt.Sprintf("sudo -u postgres psql -d '%s' -c '\\dt' 2>&1", dbName)
	case "mongodb", "mongo":
		cmd = fmt.Sprintf("mongosh '%s' --eval 'db.getCollectionNames()' --quiet 2>&1", dbName)
	case "redis":
		cmd = "redis-cli DBSIZE && redis-cli --scan --count 100 2>&1 | head -50"
	default:
		return "", fmt.Errorf("unsupported engine")
	}

	return provisioner.RunScript(server, cmd)
}

// DescribeTable gets the structure of a table
func (s *DBManagerService) DescribeTable(ctx context.Context, serverID, engine, dbName, table string) (string, error) {
	server, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		return "", err
	}

	var cmd string
	switch engine {
	case "mysql", "mariadb":
		cmd = fmt.Sprintf("mysql -u root '%s' -e 'DESCRIBE %s;' 2>&1", dbName, table)
	case "postgresql", "postgres":
		cmd = fmt.Sprintf("sudo -u postgres psql -d '%s' -c '\\d %s' 2>&1", dbName, table)
	case "mongodb", "mongo":
		cmd = fmt.Sprintf("mongosh '%s' --eval 'db.%s.findOne()' --quiet 2>&1", dbName, table)
	default:
		return "", fmt.Errorf("unsupported engine")
	}

	return provisioner.RunScript(server, cmd)
}

// GetDBSize gets database size info
func (s *DBManagerService) GetDBSize(ctx context.Context, serverID, engine, dbName string) (string, error) {
	server, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		return "", err
	}

	var cmd string
	switch engine {
	case "mysql", "mariadb":
		cmd = fmt.Sprintf(`mysql -u root -e "SELECT table_schema AS 'Database', ROUND(SUM(data_length + index_length)/1024/1024, 2) AS 'Size (MB)' FROM information_schema.tables WHERE table_schema = '%s' GROUP BY table_schema;" 2>&1`, dbName)
	case "postgresql", "postgres":
		cmd = fmt.Sprintf("sudo -u postgres psql -c \"SELECT pg_database.datname, pg_size_pretty(pg_database_size(pg_database.datname)) FROM pg_database WHERE datname = '%s';\" 2>&1", dbName)
	case "mongodb", "mongo":
		cmd = fmt.Sprintf("mongosh '%s' --eval 'db.stats()' --quiet 2>&1", dbName)
	case "redis":
		cmd = "redis-cli INFO memory 2>&1 | grep -E 'used_memory_human|total_system_memory_human'"
	default:
		return "", fmt.Errorf("unsupported engine")
	}

	return provisioner.RunScript(server, cmd)
}

// ListDatabasesOnServer lists all databases on a server
func (s *DBManagerService) ListDatabasesOnServer(ctx context.Context, serverID, engine string) (string, error) {
	server, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		return "", err
	}

	var cmd string
	switch engine {
	case "mysql", "mariadb":
		cmd = "mysql -u root -e 'SHOW DATABASES;' 2>&1"
	case "postgresql", "postgres":
		cmd = "sudo -u postgres psql -c '\\l' 2>&1"
	case "mongodb", "mongo":
		cmd = "mongosh --eval 'show dbs' --quiet 2>&1"
	case "redis":
		cmd = "redis-cli INFO keyspace 2>&1"
	default:
		return "", fmt.Errorf("unsupported engine")
	}

	return provisioner.RunScript(server, cmd)
}

// ListUsers lists database users on a server
func (s *DBManagerService) ListUsers(ctx context.Context, serverID, engine string) (string, error) {
	server, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		return "", err
	}

	var cmd string
	switch engine {
	case "mysql", "mariadb":
		cmd = "mysql -u root -e \"SELECT User, Host FROM mysql.user;\" 2>&1"
	case "postgresql", "postgres":
		cmd = "sudo -u postgres psql -c '\\du' 2>&1"
	case "mongodb", "mongo":
		cmd = "mongosh admin --eval 'db.getUsers()' --quiet 2>&1"
	default:
		return "", fmt.Errorf("unsupported engine")
	}

	return provisioner.RunScript(server, cmd)
}

// CreateDBUser creates a database user on a server
func (s *DBManagerService) CreateDBUser(ctx context.Context, serverID, engine, username, password string, dbName string) (string, error) {
	server, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		return "", err
	}

	var cmd string
	switch engine {
	case "mysql", "mariadb":
		cmd = fmt.Sprintf(`mysql -u root -e "CREATE USER '%s'@'%%' IDENTIFIED BY '%s'; GRANT ALL PRIVILEGES ON %s.* TO '%s'@'%%'; FLUSH PRIVILEGES;" 2>&1`,
			username, password, dbName, username)
	case "postgresql", "postgres":
		cmd = fmt.Sprintf(`sudo -u postgres psql -c "CREATE USER %s WITH PASSWORD '%s'; GRANT ALL PRIVILEGES ON DATABASE %s TO %s;" 2>&1`,
			username, password, dbName, username)
	case "mongodb", "mongo":
		cmd = fmt.Sprintf(`mongosh '%s' --eval "db.createUser({user: '%s', pwd: '%s', roles: [{role: 'readWrite', db: '%s'}]})" 2>&1`,
			dbName, username, password, dbName)
	default:
		return "", fmt.Errorf("unsupported engine")
	}

	return provisioner.RunScript(server, cmd)
}

// ExportDB exports a database from a server
func (s *DBManagerService) ExportDB(ctx context.Context, serverID, engine, dbName string) (string, error) {
	server, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		return "", err
	}

	exportPath := fmt.Sprintf("/tmp/novapanel_export_%s.sql", dbName)
	var cmd string
	switch engine {
	case "mysql", "mariadb":
		cmd = fmt.Sprintf("mysqldump -u root '%s' > '%s' 2>&1 && ls -lh '%s' && echo 'Export ready at %s'", dbName, exportPath, exportPath, exportPath)
	case "postgresql", "postgres":
		cmd = fmt.Sprintf("sudo -u postgres pg_dump '%s' > '%s' 2>&1 && ls -lh '%s' && echo 'Export ready at %s'", dbName, exportPath, exportPath, exportPath)
	case "mongodb", "mongo":
		exportPath = fmt.Sprintf("/tmp/novapanel_export_%s", dbName)
		cmd = fmt.Sprintf("mongodump --db '%s' --out '%s' 2>&1 && du -sh '%s' && echo 'Export ready at %s'", dbName, exportPath, exportPath, exportPath)
	default:
		return "", fmt.Errorf("unsupported engine")
	}

	return provisioner.RunScript(server, cmd)
}

// ImportDB imports SQL into a database
func (s *DBManagerService) ImportDB(ctx context.Context, serverID, engine, dbName, filePath string) (string, error) {
	server, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		return "", err
	}

	var cmd string
	switch engine {
	case "mysql", "mariadb":
		cmd = fmt.Sprintf("mysql -u root '%s' < '%s' 2>&1 && echo 'Import completed'", dbName, filePath)
	case "postgresql", "postgres":
		cmd = fmt.Sprintf("sudo -u postgres psql -d '%s' < '%s' 2>&1 && echo 'Import completed'", dbName, filePath)
	case "mongodb", "mongo":
		cmd = fmt.Sprintf("mongorestore --db '%s' '%s' 2>&1 && echo 'Import completed'", dbName, filePath)
	default:
		return "", fmt.Errorf("unsupported engine")
	}

	return provisioner.RunScript(server, cmd)
}

// DeployDBTool deploys a web-based DB management tool via Docker on a server
func (s *DBManagerService) DeployDBTool(ctx context.Context, serverID, engine string) (map[string]string, error) {
	server, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		return nil, err
	}

	var cmd string
	info := map[string]string{"engine": engine, "server_ip": server.IPAddress}

	switch engine {
	case "mysql", "mariadb":
		// Deploy phpMyAdmin
		cmd = `docker rm -f novapanel-phpmyadmin 2>/dev/null
docker run -d --name novapanel-phpmyadmin --restart unless-stopped \
  -e PMA_HOST=host.docker.internal \
  -e PMA_ARBITRARY=1 \
  -e UPLOAD_LIMIT=256M \
  -p 8081:80 \
  --add-host=host.docker.internal:host-gateway \
  phpmyadmin/phpmyadmin:latest 2>&1
echo "phpMyAdmin deployed"
docker port novapanel-phpmyadmin 80`
		info["tool"] = "phpMyAdmin"
		info["port"] = "8081"
		info["url"] = fmt.Sprintf("http://%s:8081", server.IPAddress)

	case "postgresql", "postgres":
		// Deploy Adminer (lightweight, supports postgres)
		cmd = `docker rm -f novapanel-adminer 2>/dev/null
docker run -d --name novapanel-adminer --restart unless-stopped \
  -e ADMINER_DEFAULT_SERVER=host.docker.internal \
  -e ADMINER_DESIGN=nette \
  -p 8082:8080 \
  --add-host=host.docker.internal:host-gateway \
  adminer:latest 2>&1
echo "Adminer deployed"
docker port novapanel-adminer 8080`
		info["tool"] = "Adminer"
		info["port"] = "8082"
		info["url"] = fmt.Sprintf("http://%s:8082", server.IPAddress)

	case "mongodb", "mongo":
		// Deploy Mongo Express
		cmd = `docker rm -f novapanel-mongoexpress 2>/dev/null
docker run -d --name novapanel-mongoexpress --restart unless-stopped \
  -e ME_CONFIG_MONGODB_URL=mongodb://host.docker.internal:27017 \
  -e ME_CONFIG_BASICAUTH=false \
  -p 8083:8081 \
  --add-host=host.docker.internal:host-gateway \
  mongo-express:latest 2>&1
echo "Mongo Express deployed"
docker port novapanel-mongoexpress 8081`
		info["tool"] = "Mongo Express"
		info["port"] = "8083"
		info["url"] = fmt.Sprintf("http://%s:8083", server.IPAddress)

	case "redis":
		// Deploy Redis Commander
		cmd = `docker rm -f novapanel-rediscommander 2>/dev/null
docker run -d --name novapanel-rediscommander --restart unless-stopped \
  -e REDIS_HOSTS=local:host.docker.internal:6379 \
  -p 8084:8081 \
  --add-host=host.docker.internal:host-gateway \
  rediscommander/redis-commander:latest 2>&1
echo "Redis Commander deployed"
docker port novapanel-rediscommander 8081`
		info["tool"] = "Redis Commander"
		info["port"] = "8084"
		info["url"] = fmt.Sprintf("http://%s:8084", server.IPAddress)

	default:
		return nil, fmt.Errorf("no management tool available for engine: %s", engine)
	}

	output, err := provisioner.RunScript(server, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy %s: %w\nOutput: %s", info["tool"], err, output)
	}
	info["output"] = output

	return info, nil
}

// GetToolStatus checks if a DB tool container is running
func (s *DBManagerService) GetToolStatus(ctx context.Context, serverID, engine string) (map[string]string, error) {
	server, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		return nil, err
	}

	containerMap := map[string]string{
		"mysql": "novapanel-phpmyadmin", "mariadb": "novapanel-phpmyadmin",
		"postgresql": "novapanel-adminer", "postgres": "novapanel-adminer",
		"mongodb": "novapanel-mongoexpress", "mongo": "novapanel-mongoexpress",
		"redis": "novapanel-rediscommander",
	}
	toolMap := map[string]string{
		"mysql": "phpMyAdmin", "mariadb": "phpMyAdmin",
		"postgresql": "Adminer", "postgres": "Adminer",
		"mongodb": "Mongo Express", "mongo": "Mongo Express",
		"redis": "Redis Commander",
	}
	portMap := map[string]string{
		"mysql": "8081", "mariadb": "8081",
		"postgresql": "8082", "postgres": "8082",
		"mongodb": "8083", "mongo": "8083",
		"redis": "8084",
	}

	container := containerMap[engine]
	cmd := fmt.Sprintf("docker inspect -f '{{.State.Status}}' %s 2>/dev/null || echo 'not_found'", container)
	output, _ := provisioner.RunScript(server, cmd)
	status := strings.TrimSpace(output)

	info := map[string]string{
		"engine":    engine,
		"tool":      toolMap[engine],
		"container": container,
		"status":    status,
		"port":      portMap[engine],
		"server_ip": server.IPAddress,
	}

	if status == "running" {
		info["url"] = fmt.Sprintf("http://%s:%s", server.IPAddress, portMap[engine])
	}

	return info, nil
}

// StopTool stops and removes a DB tool container
func (s *DBManagerService) StopTool(ctx context.Context, serverID, engine string) error {
	server, err := s.getServerSSH(ctx, serverID)
	if err != nil {
		return err
	}

	containerMap := map[string]string{
		"mysql": "novapanel-phpmyadmin", "mariadb": "novapanel-phpmyadmin",
		"postgresql": "novapanel-adminer", "postgres": "novapanel-adminer",
		"mongodb": "novapanel-mongoexpress", "mongo": "novapanel-mongoexpress",
		"redis": "novapanel-rediscommander",
	}

	container := containerMap[engine]
	cmd := fmt.Sprintf("docker rm -f %s 2>/dev/null && echo 'stopped'", container)
	_, err = provisioner.RunScript(server, cmd)
	return err
}
