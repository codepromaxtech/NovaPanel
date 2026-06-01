package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/provisioner"
)

type DeployService struct {
	pool *pgxpool.Pool
}

func NewDeployService(pool *pgxpool.Pool) *DeployService {
	return &DeployService{pool: pool}
}

func (s *DeployService) Create(ctx context.Context, userID uuid.UUID, req models.CreateDeploymentRequest) (*models.Deployment, error) {
	branch := req.Branch
	if branch == "" {
		branch = "main"
	}

	d := &models.Deployment{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO deployments (app_id, user_id, branch, status)
		 VALUES ($1, $2, $3, 'pending')
		 RETURNING id, app_id, user_id, commit_hash, branch, status, build_log, started_at, completed_at, created_at`,
		req.AppID, userID, branch,
	).Scan(&d.ID, &d.AppID, &d.UserID, &d.CommitHash, &d.Branch, &d.Status, &d.BuildLog, &d.StartedAt, &d.CompletedAt, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}
	return d, nil
}

// TriggerDeploy creates a deployment record AND executes it in the background.
// When req.TargetServerIDs has 2+ entries the deployment fans out in parallel.
func (s *DeployService) TriggerDeploy(ctx context.Context, userID uuid.UUID, req models.CreateDeploymentRequest) (*models.Deployment, error) {
	d, err := s.Create(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	if len(req.TargetServerIDs) > 1 {
		// Multi-server: insert deployment_targets rows, then fan-out.
		for _, sid := range req.TargetServerIDs {
			s.pool.Exec(ctx,
				`INSERT INTO deployment_targets (deployment_id, server_id, status)
				 VALUES ($1, $2, 'pending')
				 ON CONFLICT (deployment_id, server_id) DO NOTHING`,
				d.ID, sid)
		}
		go func() {
			s.fanOutDeploy(context.Background(), d, req.TargetServerIDs)
		}()
	} else {
		go func() {
			if err := s.ExecuteDeployment(context.Background(), d.ID.String()); err != nil {
				log.Printf("❌ Deployment %s failed: %v", d.ID, err)
			}
		}()
	}

	return d, nil
}

// fanOutDeploy runs ExecuteDeploymentOnServer for each target in parallel.
func (s *DeployService) fanOutDeploy(ctx context.Context, d *models.Deployment, serverIDs []string) {
	var wg sync.WaitGroup
	anyFailed := false
	var mu sync.Mutex

	for _, sid := range serverIDs {
		wg.Add(1)
		go func(serverID string) {
			defer wg.Done()
			if err := s.ExecuteDeploymentOnServer(ctx, d.ID.String(), serverID); err != nil {
				log.Printf("❌ Multi-deploy %s → server %s failed: %v", d.ID, serverID, err)
				mu.Lock()
				anyFailed = true
				mu.Unlock()
			}
		}(sid)
	}
	wg.Wait()

	// Roll up status to the parent deployment row.
	mu.Lock()
	failed := anyFailed
	mu.Unlock()

	finalStatus := "success"
	if failed {
		finalStatus = "partial" // some targets failed
	}
	s.pool.Exec(ctx,
		`UPDATE deployments SET status = $1, completed_at = NOW() WHERE id = $2`,
		finalStatus, d.ID)
}

// ExecuteDeployment runs the actual git clone/pull, build, and restart pipeline on the target server
func (s *DeployService) ExecuteDeployment(ctx context.Context, deploymentID string) error {
	// 1. Fetch deployment + application details
	var appName, gitRepo, gitBranch, runtime, deployBranch string
	var serverID uuid.UUID
	var deployID uuid.UUID

	err := s.pool.QueryRow(ctx,
		`SELECT d.id, d.branch, a.name, a.git_repo, a.git_branch, a.runtime, a.server_id
		 FROM deployments d
		 JOIN applications a ON d.app_id = a.id
		 WHERE d.id = $1`, deploymentID,
	).Scan(&deployID, &deployBranch, &appName, &gitRepo, &gitBranch, &runtime, &serverID)
	if err != nil {
		return s.failDeployment(ctx, deploymentID, fmt.Sprintf("Failed to fetch deployment: %v", err))
	}

	// Use deployment branch if set, otherwise app default
	branch := deployBranch
	if branch == "" {
		branch = gitBranch
	}
	if branch == "" {
		branch = "main"
	}

	if gitRepo == "" {
		return s.failDeployment(ctx, deploymentID, "No git repository configured for this application")
	}

	// 2. Mark as running
	now := time.Now()
	s.pool.Exec(ctx,
		`UPDATE deployments SET status = 'running', started_at = $1 WHERE id = $2`,
		now, deployID)

	// 3. Fetch server SSH details
	server, err := s.getServerInfo(ctx, serverID.String())
	if err != nil {
		return s.failDeployment(ctx, deploymentID, fmt.Sprintf("Server not available: %v", err))
	}

	var logBuilder strings.Builder
	appendLog := func(msg string) {
		logBuilder.WriteString(fmt.Sprintf("[%s] %s\n", time.Now().Format("15:04:05"), msg))
		// Update build_log in real-time so frontend can poll it
		s.pool.Exec(ctx, `UPDATE deployments SET build_log = $1 WHERE id = $2`, logBuilder.String(), deployID)
	}

	appDir := fmt.Sprintf("/opt/novapanel/apps/%s", appName)

	// 4. Git clone or pull
	appendLog(fmt.Sprintf("📦 Starting deployment for %s (branch: %s)", appName, branch))
	appendLog(fmt.Sprintf("📁 App directory: %s", appDir))

	gitScript := fmt.Sprintf(`
if [ -d "%s/.git" ]; then
    cd "%s"
    echo "Pulling latest changes..."
    git fetch --all 2>&1
    git checkout %s 2>&1
    git reset --hard origin/%s 2>&1
    echo "COMMIT:$(git rev-parse --short HEAD)"
else
    echo "Cloning repository..."
    mkdir -p "%s"
    git clone --branch %s "%s" "%s" 2>&1
    cd "%s"
    echo "COMMIT:$(git rev-parse --short HEAD)"
fi
`, appDir, appDir, branch, branch, appDir, branch, gitRepo, appDir, appDir)

	appendLog("🔄 Running git operations...")
	output, err := provisioner.RunScript(server, gitScript)
	if err != nil {
		appendLog(fmt.Sprintf("❌ Git failed: %v\n%s", err, output))
		return s.failDeployment(ctx, deploymentID, logBuilder.String())
	}
	appendLog(output)

	// Extract commit hash
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "COMMIT:") {
			commitHash := strings.TrimPrefix(line, "COMMIT:")
			s.pool.Exec(ctx, `UPDATE deployments SET commit_hash = $1 WHERE id = $2`, strings.TrimSpace(commitHash), deployID)
		}
	}

	// 5. Build step based on runtime
	buildScript := s.getBuildScript(runtime, appDir)
	if buildScript != "" {
		appendLog(fmt.Sprintf("🔨 Building (%s)...", runtime))
		output, err = provisioner.RunScript(server, buildScript)
		if err != nil {
			appendLog(fmt.Sprintf("❌ Build failed: %v\n%s", err, output))
			return s.failDeployment(ctx, deploymentID, logBuilder.String())
		}
		appendLog(output)
	} else {
		appendLog("ℹ️  No build step needed for this runtime")
	}

	// 6. Restart application service
	restartScript := s.getRestartScript(runtime, appName, appDir)
	if restartScript != "" {
		appendLog("🔄 Restarting application...")
		output, err = provisioner.RunScript(server, restartScript)
		if err != nil {
			appendLog(fmt.Sprintf("⚠️  Restart warning: %v\n%s", err, output))
			// Don't fail deployment on restart warning
		} else {
			appendLog(output)
		}
	}

	// 7. Mark as success
	completedAt := time.Now()
	appendLog(fmt.Sprintf("✅ Deployment completed in %s", completedAt.Sub(now).Round(time.Second)))

	s.pool.Exec(ctx,
		`UPDATE deployments SET status = 'success', build_log = $1, completed_at = $2 WHERE id = $3`,
		logBuilder.String(), completedAt, deployID)

	return nil
}

// ExecuteDeploymentOnServer runs the deploy pipeline on a specific server and
// records per-server progress in the deployment_targets table.
func (s *DeployService) ExecuteDeploymentOnServer(ctx context.Context, deploymentID, serverID string) error {
	// Mark target as running
	now := time.Now()
	s.pool.Exec(ctx,
		`UPDATE deployment_targets SET status = 'running', started_at = $1
		 WHERE deployment_id = $2 AND server_id = $3`,
		now, deploymentID, serverID)

	// Fetch app details from the parent deployment
	var appName, gitRepo, gitBranch, runtime, deployBranch string
	var deployID uuid.UUID
	err := s.pool.QueryRow(ctx,
		`SELECT d.id, d.branch, a.name, a.git_repo, a.git_branch, a.runtime
		 FROM deployments d
		 JOIN applications a ON d.app_id = a.id
		 WHERE d.id = $1`, deploymentID,
	).Scan(&deployID, &deployBranch, &appName, &gitRepo, &gitBranch, &runtime)
	if err != nil {
		return s.failTarget(ctx, deploymentID, serverID, fmt.Sprintf("fetch deployment: %v", err))
	}

	branch := deployBranch
	if branch == "" {
		branch = gitBranch
	}
	if branch == "" {
		branch = "main"
	}

	server, err := s.getServerInfo(ctx, serverID)
	if err != nil {
		return s.failTarget(ctx, deploymentID, serverID, fmt.Sprintf("server unavailable: %v", err))
	}

	var logBuilder strings.Builder
	appendLog := func(msg string) {
		logBuilder.WriteString(fmt.Sprintf("[%s] %s\n", time.Now().Format("15:04:05"), msg))
		s.pool.Exec(ctx,
			`UPDATE deployment_targets SET build_log = $1 WHERE deployment_id = $2 AND server_id = $3`,
			logBuilder.String(), deploymentID, serverID)
	}

	appDir := fmt.Sprintf("/opt/novapanel/apps/%s", appName)
	appendLog(fmt.Sprintf("📦 Deploying %s to server %s (branch: %s)", appName, serverID, branch))

	gitScript := fmt.Sprintf(`
if [ -d "%s/.git" ]; then
    cd "%s"; git fetch --all 2>&1; git checkout %s 2>&1; git reset --hard origin/%s 2>&1
    echo "COMMIT:$(git rev-parse --short HEAD)"
else
    mkdir -p "%s"; git clone --branch %s "%s" "%s" 2>&1
    cd "%s"; echo "COMMIT:$(git rev-parse --short HEAD)"
fi
`, appDir, appDir, branch, branch, appDir, branch, gitRepo, appDir, appDir)

	output, err := provisioner.RunScript(server, gitScript)
	if err != nil {
		appendLog(fmt.Sprintf("❌ Git failed: %v\n%s", err, output))
		return s.failTarget(ctx, deploymentID, serverID, logBuilder.String())
	}
	appendLog(output)

	// Capture commit hash
	var commitHash string
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "COMMIT:") {
			commitHash = strings.TrimSpace(strings.TrimPrefix(line, "COMMIT:"))
		}
	}

	buildScript := s.getBuildScript(runtime, appDir)
	if buildScript != "" {
		appendLog(fmt.Sprintf("🔨 Building (%s)...", runtime))
		output, err = provisioner.RunScript(server, buildScript)
		if err != nil {
			appendLog(fmt.Sprintf("❌ Build failed: %v\n%s", err, output))
			return s.failTarget(ctx, deploymentID, serverID, logBuilder.String())
		}
		appendLog(output)
	}

	restartScript := s.getRestartScript(runtime, appName, appDir)
	if restartScript != "" {
		appendLog("🔄 Restarting application...")
		output, err = provisioner.RunScript(server, restartScript)
		if err != nil {
			appendLog(fmt.Sprintf("⚠️ Restart warning: %v\n%s", err, output))
		} else {
			appendLog(output)
		}
	}

	completedAt := time.Now()
	appendLog(fmt.Sprintf("✅ Done in %s", completedAt.Sub(now).Round(time.Second)))
	s.pool.Exec(ctx,
		`UPDATE deployment_targets
		 SET status = 'success', build_log = $1, commit_hash = $2, completed_at = $3
		 WHERE deployment_id = $4 AND server_id = $5`,
		logBuilder.String(), commitHash, completedAt, deploymentID, serverID)
	return nil
}

func (s *DeployService) failTarget(ctx context.Context, deploymentID, serverID, msg string) error {
	s.pool.Exec(ctx,
		`UPDATE deployment_targets SET status = 'failed', build_log = $1, completed_at = NOW()
		 WHERE deployment_id = $2 AND server_id = $3`,
		msg, deploymentID, serverID)
	return errors.New(msg)
}

// ListTargets returns per-server results for a multi-server deployment.
func (s *DeployService) ListTargets(ctx context.Context, deploymentID string) ([]models.DeploymentTarget, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, deployment_id, server_id, status,
		        COALESCE(build_log,''), COALESCE(commit_hash,''),
		        started_at, completed_at, created_at
		 FROM deployment_targets WHERE deployment_id = $1 ORDER BY created_at`, deploymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []models.DeploymentTarget
	for rows.Next() {
		var t models.DeploymentTarget
		if err := rows.Scan(&t.ID, &t.DeploymentID, &t.ServerID, &t.Status,
			&t.BuildLog, &t.CommitHash, &t.StartedAt, &t.CompletedAt, &t.CreatedAt); err != nil {
			continue
		}
		targets = append(targets, t)
	}
	return targets, nil
}

// Redeploy re-triggers a deployment for an existing app with same or new branch
func (s *DeployService) Redeploy(ctx context.Context, deploymentID string) (*models.Deployment, error) {
	// Get original deployment details
	var appID, userID uuid.UUID
	var branch string
	err := s.pool.QueryRow(ctx,
		`SELECT app_id, user_id, branch FROM deployments WHERE id = $1`, deploymentID,
	).Scan(&appID, &userID, &branch)
	if err != nil {
		return nil, fmt.Errorf("deployment not found")
	}

	return s.TriggerDeploy(ctx, userID, models.CreateDeploymentRequest{
		AppID:  appID.String(),
		Branch: branch,
	})
}

// GetLogs returns build logs for a deployment (for polling)
func (s *DeployService) GetLogs(ctx context.Context, id string) (string, string, error) {
	var buildLog, status string
	err := s.pool.QueryRow(ctx,
		`SELECT COALESCE(build_log, ''), status FROM deployments WHERE id = $1`, id,
	).Scan(&buildLog, &status)
	if err != nil {
		return "", "", fmt.Errorf("deployment not found")
	}
	return buildLog, status, nil
}

func (s *DeployService) List(ctx context.Context, userID uuid.UUID, role string, page, perPage int) (*models.PaginatedResponse, error) {
	offset := (page - 1) * perPage
	var total int64

	query := `SELECT id, app_id, user_id, commit_hash, branch, status, build_log, started_at, completed_at, created_at FROM deployments`
	countQuery := `SELECT count(*) FROM deployments`
	if role != "admin" {
		query += fmt.Sprintf(` WHERE user_id = '%s'`, userID)
		countQuery += fmt.Sprintf(` WHERE user_id = '%s'`, userID)
	}
	s.pool.QueryRow(ctx, countQuery).Scan(&total)
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT %d OFFSET %d`, perPage, offset)

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.Deployment
	for rows.Next() {
		var d models.Deployment
		if err := rows.Scan(&d.ID, &d.AppID, &d.UserID, &d.CommitHash, &d.Branch, &d.Status, &d.BuildLog, &d.StartedAt, &d.CompletedAt, &d.CreatedAt); err != nil {
			continue
		}
		items = append(items, d)
	}

	return &models.PaginatedResponse{
		Data: items, Total: total, Page: page, PerPage: perPage,
		TotalPages: int(math.Ceil(float64(total) / float64(perPage))),
	}, nil
}

func (s *DeployService) GetByID(ctx context.Context, id string) (*models.Deployment, error) {
	d := &models.Deployment{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, app_id, user_id, commit_hash, branch, status, build_log, started_at, completed_at, created_at FROM deployments WHERE id = $1`, id,
	).Scan(&d.ID, &d.AppID, &d.UserID, &d.CommitHash, &d.Branch, &d.Status, &d.BuildLog, &d.StartedAt, &d.CompletedAt, &d.CreatedAt)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// ─── Helper Methods ───

func (s *DeployService) failDeployment(ctx context.Context, deploymentID string, logMessage string) error {
	now := time.Now()
	s.pool.Exec(ctx,
		`UPDATE deployments SET status = 'failed', build_log = $1, completed_at = $2 WHERE id = $3`,
		logMessage, now, deploymentID)
	return errors.New(logMessage)
}

func (s *DeployService) getServerInfo(ctx context.Context, serverID string) (provisioner.ServerInfo, error) {
	return GetServerInfo(ctx, s.pool, serverID)
}

func (s *DeployService) getBuildScript(runtime, appDir string) string {
	switch strings.ToLower(runtime) {
	case "node", "nodejs", "javascript", "typescript":
		return fmt.Sprintf(`cd %s
if [ -f "package-lock.json" ]; then
    npm ci --production 2>&1
elif [ -f "yarn.lock" ]; then
    yarn install --production 2>&1
elif [ -f "pnpm-lock.yaml" ]; then
    pnpm install --prod 2>&1
else
    npm install --production 2>&1
fi
if [ -f "package.json" ] && grep -q '"build"' package.json; then
    npm run build 2>&1
fi
echo "Build completed"`, appDir)

	case "go", "golang":
		return fmt.Sprintf(`cd %s
go build -o app ./... 2>&1
echo "Build completed"`, appDir)

	case "python", "django", "flask":
		return fmt.Sprintf(`cd %s
if [ -f "requirements.txt" ]; then
    pip3 install -r requirements.txt 2>&1
fi
if [ -f "manage.py" ]; then
    python3 manage.py migrate --noinput 2>&1
    python3 manage.py collectstatic --noinput 2>&1 || true
fi
echo "Build completed"`, appDir)

	case "php", "laravel":
		return fmt.Sprintf(`cd %s
if [ -f "composer.json" ]; then
    composer install --no-dev --optimize-autoloader 2>&1
fi
if [ -f "artisan" ]; then
    php artisan migrate --force 2>&1
    php artisan config:cache 2>&1
    php artisan route:cache 2>&1
    php artisan view:cache 2>&1
fi
echo "Build completed"`, appDir)

	case "static", "html":
		return "" // No build needed

	default:
		return "" // Unknown runtime, skip build
	}
}

func (s *DeployService) getRestartScript(runtime, appName, appDir string) string {
	serviceName := fmt.Sprintf("novapanel-%s", appName)

	switch strings.ToLower(runtime) {
	case "node", "nodejs", "javascript", "typescript":
		// Try PM2 first, fall back to systemd
		return fmt.Sprintf(`
if command -v pm2 &> /dev/null; then
    if pm2 describe %s &> /dev/null; then
        pm2 restart %s 2>&1
    else
        cd %s
        pm2 start npm --name %s -- start 2>&1 || pm2 start node --name %s -- index.js 2>&1
    fi
    pm2 save 2>&1
    echo "PM2 restart complete"
elif systemctl is-active --quiet %s.service 2>/dev/null; then
    systemctl restart %s.service 2>&1
    echo "Systemd restart complete"
else
    echo "No process manager found. Please configure PM2 or systemd for %s"
fi`, appName, appName, appDir, appName, appName, serviceName, serviceName, appName)

	case "go", "golang":
		return fmt.Sprintf(`
if systemctl is-active --quiet %s.service 2>/dev/null; then
    systemctl restart %s.service 2>&1
    echo "Service restarted"
else
    echo "No systemd service found for %s. Create one at /etc/systemd/system/%s.service"
fi`, serviceName, serviceName, appName, serviceName)

	case "python", "django", "flask":
		return fmt.Sprintf(`
if systemctl is-active --quiet %s.service 2>/dev/null; then
    systemctl restart %s.service 2>&1
    echo "Service restarted"
elif command -v supervisorctl &> /dev/null; then
    supervisorctl restart %s 2>&1 || true
    echo "Supervisor restart complete"
else
    echo "No process manager found for %s"
fi`, serviceName, serviceName, appName, appName)

	case "php", "laravel":
		return `
if systemctl is-active --quiet php*-fpm.service 2>/dev/null; then
    systemctl restart php*-fpm.service 2>&1
    echo "PHP-FPM restarted"
fi
if systemctl is-active --quiet nginx.service 2>/dev/null; then
    nginx -t 2>&1 && systemctl reload nginx 2>&1
    echo "Nginx reloaded"
elif systemctl is-active --quiet apache2.service 2>/dev/null; then
    systemctl reload apache2 2>&1
    echo "Apache reloaded"
fi`

	default:
		return ""
	}
}
