package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/config"
	novacrypto "github.com/novapanel/novapanel/internal/crypto"
	"github.com/novapanel/novapanel/internal/database"
	"github.com/novapanel/novapanel/internal/handlers"
	"github.com/novapanel/novapanel/internal/middleware"
	"github.com/novapanel/novapanel/internal/provisioner"
	"github.com/novapanel/novapanel/internal/queue"
	"github.com/novapanel/novapanel/internal/services"
	"github.com/novapanel/novapanel/internal/websocket"
)

func main() {
	cfg := config.Load()

	// Determine command
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "migrate":
		runMigrations(cfg)
	case "serve":
		startServer(cfg)
	default:
		fmt.Printf("Unknown command: %s\nUsage: novapanel [serve|migrate]\n", cmd)
		os.Exit(1)
	}
}

func runMigrations(cfg *config.Config) {
	pool := database.NewPostgresPool(cfg)
	defer pool.Close()

	// Find migrations directory
	execPath, _ := os.Executable()
	baseDir := filepath.Dir(filepath.Dir(filepath.Dir(execPath)))
	migrationsDir := filepath.Join(baseDir, "migrations")

	// Fallback to relative path for development
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		migrationsDir = "migrations"
	}
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		migrationsDir = "backend/migrations"
	}

	log.Println("Running database migrations...")
	if err := database.RunMigrations(pool, migrationsDir); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("✓ Migrations completed successfully")
}

func startServer(cfg *config.Config) {
	// Reject weak default secrets in production
	if cfg.Env == "production" {
		if cfg.JWTSecret == "novapanel-dev-secret-change-in-production" {
			log.Fatal("FATAL: JWT_SECRET must be set to a strong secret in production. Refusing to start.")
		}
		if cfg.DBPassword == "novapanel_secret" {
			log.Fatal("FATAL: DB_PASSWORD must be set in production. Refusing to start.")
		}
	}

	// Connect to databases
	pool := database.NewPostgresPool(cfg)
	defer pool.Close()

	rdb := database.NewRedisClient(cfg)
	defer rdb.Close()

	// Run migrations automatically in development
	if cfg.Env == "development" {
		migrationsDir := "migrations"
		if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
			migrationsDir = "backend/migrations"
		}
		if _, err := os.Stat(migrationsDir); !os.IsNotExist(err) {
			log.Println("Running auto-migrations (development mode)...")
			if err := database.RunMigrations(pool, migrationsDir); err != nil {
				log.Printf("Warning: auto-migration failed: %v", err)
			}
		}
	}

	// Auto-register host server as the first node
	autoRegisterHostServer(pool)

	// Initialize services
	authService := services.NewAuthService(pool, cfg, rdb)

	// Login brute-force protection
	loginLimiter := middleware.NewLoginLimiter(rdb)

	smtpSvc := services.NewSMTPService(cfg)
	apiKeySvc := services.NewAPIKeyService(pool)
	domainService := services.NewDomainService(pool)
	serverService := services.NewServerService(pool)
	databaseService := services.NewDatabaseService(pool)
	emailService := services.NewEmailService(pool)
	backupService := services.NewBackupService(pool, cfg.EncryptionKey)
	deployService := services.NewDeployService(pool)
	appService := services.NewAppService(pool, cfg.EncryptionKey)
	securityService := services.NewSecurityService(pool, cfg.EncryptionKey)
	billingService := services.NewBillingService(pool)
	metricsService := services.NewMetricsService(pool)
	alertSvc := services.NewAlertService(pool, smtpSvc)
	ftpSvc := services.NewFTPService(pool, cfg.EncryptionKey)
	resellerSvc := services.NewResellerService(pool)

	// Initialize task queue & workers
	taskQueue := queue.NewTaskQueue(rdb, pool)
	worker := queue.NewWorker(taskQueue)

	// Register server_setup task handler for auto-provisioning
	setupTaskHandler := provisioner.NewSetupHandler(pool)
	worker.RegisterHandler("server_setup", setupTaskHandler.Handle)

	// nginx:configure — writes upstream load-balancer config and reloads nginx
	worker.RegisterHandler("nginx:configure", func(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
		serverID, _ := payload["server_id"].(string)
		domain, _ := payload["domain"].(string)
		rawIPs, _ := payload["target_ips"].([]interface{})
		if serverID == "" || domain == "" {
			return nil, fmt.Errorf("nginx:configure: missing server_id or domain")
		}
		var targetIPs []string
		for _, ip := range rawIPs {
			if s, ok := ip.(string); ok && s != "" {
				targetIPs = append(targetIPs, s)
			}
		}
		if len(targetIPs) == 0 {
			return nil, fmt.Errorf("nginx:configure: no target IPs provided")
		}

		// Build upstream block
		upstream := fmt.Sprintf("upstream lb_%s {\n", strings.ReplaceAll(domain, ".", "_"))
		for _, ip := range targetIPs {
			upstream += fmt.Sprintf("    server %s:80;\n", ip)
		}
		upstream += "}\n"

		vhost := fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    location / {
        proxy_pass http://lb_%s;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}`, domain, strings.ReplaceAll(domain, ".", "_"))

		confContent := upstream + "\n" + vhost
		confPath := fmt.Sprintf("/etc/nginx/sites-available/%s-lb.conf", domain)
		enabledPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s-lb.conf", domain)

		script := fmt.Sprintf(`cat > '%s' << 'NGINXEOF'
%s
NGINXEOF
ln -sf '%s' '%s'
nginx -t 2>&1 && systemctl reload nginx 2>&1
echo "LB_CONFIGURED"`, confPath, confContent, confPath, enabledPath)

		var svrInfo provisioner.ServerInfo
		var port int
		var encKey, encPassword string
		err := pool.QueryRow(ctx,
			`SELECT host(ip_address), port, ssh_user, COALESCE(ssh_key,''), COALESCE(ssh_password,''), COALESCE(auth_method,'password')
			 FROM servers WHERE id = $1`, serverID,
		).Scan(&svrInfo.IPAddress, &port, &svrInfo.SSHUser, &encKey, &encPassword, &svrInfo.AuthMethod)
		if err != nil {
			return nil, fmt.Errorf("nginx:configure: server not found: %w", err)
		}
		svrInfo.Port = port
		if key, kerr := novacrypto.GetEncryptionKey(); kerr == nil {
			if encKey != "" {
				if dec, derr := novacrypto.Decrypt(encKey, key); derr == nil {
					encKey = dec
				}
			}
			if encPassword != "" {
				if dec, derr := novacrypto.Decrypt(encPassword, key); derr == nil {
					encPassword = dec
				}
			}
		}
		svrInfo.SSHKey = encKey
		svrInfo.SSHPassword = encPassword

		out, err := provisioner.RunScript(svrInfo, script)
		if err != nil {
			return nil, fmt.Errorf("nginx configure failed: %w\n%s", err, out)
		}
		return map[string]interface{}{"output": out}, nil
	})

	worker.Start(3) // Start 3 worker goroutines
	defer worker.Stop()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, smtpSvc, cfg.FrontendURL, loginLimiter)

	// Webhook handler for auto-deploy
	webhookHandler := handlers.NewWebhookHandler(pool, deployService)

	apiKeyHandler := handlers.NewAPIKeyHandler(apiKeySvc)
	alertHandler := handlers.NewAlertHandler(alertSvc)
	ftpHandler := handlers.NewFTPHandler(ftpSvc)
	resellerHandler := handlers.NewResellerHandler(resellerSvc)
	stripeHandler := handlers.NewStripeHandler(pool, cfg)

	domainHandler := handlers.NewDomainHandler(domainService, serverService, taskQueue, pool)
	serverHandler := handlers.NewServerHandler(serverService, taskQueue)
	setupHandler := handlers.NewSetupHandler(pool, taskQueue)
	transferService := services.NewTransferService(pool)
	transferHandler := handlers.NewTransferHandler(transferService)
	dbManagerService := services.NewDBManagerService(pool)
	dbManagerHandler := handlers.NewDBManagerHandler(dbManagerService)
	cronSvc := services.NewCronService(pool)
	cronHandler := handlers.NewCronHandler(cronSvc)
	systemctlSvc := services.NewSystemctlService(pool)
	systemctlHandler := handlers.NewSystemctlHandler(systemctlSvc)
	backupMgr := services.NewBackupManager(pool)
	backupMgrHandler := handlers.NewBackupManagerHandler(backupMgr)
	cfService := services.NewCloudflareService(pool)
	cfHandler := handlers.NewCloudflareHandler(cfService)
	databaseHandler := handlers.NewDatabaseHandler(databaseService, pool)
	emailHandler := handlers.NewEmailHandler(emailService)
	backupHandler := handlers.NewBackupHandler(backupService)
	fileManagerSvc := services.NewFileManagerService(pool)
	fileManagerHandler := handlers.NewFileManagerHandler(fileManagerSvc)
	deployHandler := handlers.NewDeployHandler(deployService)
	deployWSHandler := handlers.NewDeployWSHandler(deployService)
	appHandler := handlers.NewAppHandler(appService, pool)
	securityHandler := handlers.NewSecurityHandler(securityService)
	billingHandler := handlers.NewBillingHandler(billingService)
	settingsHandler := handlers.NewSettingsHandler(pool, smtpSvc, cfg.EncryptionKey)
	auditHandler := handlers.NewAuditHandler(pool)
	metricsHandler := handlers.NewMetricsHandler(metricsService)
	sshHandler := handlers.NewSSHHandler(serverService)

	// Team management
	teamService := services.NewTeamService(pool)
	teamHandler := handlers.NewTeamHandler(teamService)

	// PHP Version Manager
	phpService := services.NewPHPService(pool)
	phpHandler := handlers.NewPHPHandler(phpService)

	// WAF
	wafService := services.NewWAFService(pool, cfg.EncryptionKey)
	wafHandler := handlers.NewWAFHandler(wafService)

	// Server Modules
	modulesService := services.NewServerModulesService(pool, cfg.EncryptionKey)
	modulesHandler := handlers.NewServerModulesHandler(modulesService)

	// Phase 8: Docker
	dockerService, dockerErr := services.NewDockerService(pool)
	var dockerHandler *handlers.DockerHandler
	if dockerErr != nil {
		log.Printf("⚠ Docker service unavailable: %v", dockerErr)
	} else {
		dockerHandler = handlers.NewDockerHandler(dockerService)
		log.Println("✓ Docker Engine connected")
	}
	dockerExecHandler := handlers.NewDockerExecHandler(dockerService)

	// Kubernetes
	k8sService := services.NewK8sService(pool)
	k8sHandler := handlers.NewK8sHandler(k8sService)

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Prune server_metrics older than 30 days (runs daily)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			res, err := pool.Exec(context.Background(),
				`DELETE FROM server_metrics WHERE recorded_at < NOW() - INTERVAL '30 days'`)
			if err == nil {
				log.Printf("✓ Pruned %d old server_metrics rows", res.RowsAffected())
			}
		}
	}()

	// Start background metrics collector (every 30s)
	go func() {
		// Get master server ID
		var masterID string
		pool.QueryRow(context.Background(), "SELECT id FROM servers WHERE role = 'master' LIMIT 1").Scan(&masterID)
		if masterID == "" {
			log.Println("Warning: no master server for metrics collection")
			return
		}

		// Initial collection to prime CPU delta
		metricsService.CollectHostMetrics()
		time.Sleep(2 * time.Second)

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			m, err := metricsService.CollectHostMetrics()
			if err != nil {
				continue
			}
			metricsService.SaveMetrics(context.Background(), masterID, m)
			wsHub.BroadcastEvent("metrics:update", m)
		}
	}()

	// Backup scheduler (every minute)
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			backupService.RunDueSchedules(context.Background())
		}
	}()

	// Transfer scheduler (every minute)
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			transferService.RunDueSchedules(context.Background())
		}
	}()

	// Alert rule evaluator (every minute)
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			alertSvc.EvaluateRules(context.Background())
		}
	}()

	// Set up Gin router
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()

	// Global middleware
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.LoggerMiddleware())
	r.Use(gin.Recovery())
	r.Use(middleware.RateLimitMiddleware(rdb, 100, 1*time.Minute))
	if cfg.IPWhitelist != "" {
		r.Use(middleware.IPWhitelist(middleware.ParseCIDRList(cfg.IPWhitelist)))
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "novapanel-api", "version": "0.4.0"})
	})

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Public routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", loginLimiter.Middleware(), authHandler.Login)
			auth.POST("/2fa/verify", authHandler.TOTPVerify)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
		}

		// Stripe webhook (no JWT — Stripe-Signature validated)
		v1.POST("/billing/webhook", stripeHandler.Webhook)

		// Webhook routes (public — authenticated via webhook secrets)
		webhooks := v1.Group("/webhooks")
		{
			webhooks.POST("/github/:app_id", webhookHandler.HandleGitHub)
			webhooks.POST("/gitlab/:app_id", webhookHandler.HandleGitLab)
		}

		// Protected routes — support both JWT and API key auth
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg, rdb))
		protected.Use(middleware.APIKeyAuth(apiKeySvc))
		{
			// Auth
			protected.GET("/auth/me", authHandler.Me)
			protected.POST("/auth/refresh", authHandler.Refresh)
			protected.POST("/auth/logout", authHandler.Logout)
			protected.POST("/auth/2fa/setup", authHandler.TOTPSetup)
			protected.POST("/auth/2fa/enable", authHandler.TOTPEnable)
			protected.DELETE("/auth/2fa", authHandler.TOTPDisable)
			protected.GET("/auth/sessions", authHandler.ListSessions)
			protected.DELETE("/auth/sessions/:id", authHandler.RevokeSession)
			protected.DELETE("/auth/sessions", authHandler.RevokeAllOtherSessions)

			// Domains
			domains := protected.Group("/domains")
			{
				domains.POST("", domainHandler.Create)
				domains.GET("", domainHandler.List)
				domains.GET("/:id", domainHandler.GetByID)
				domains.PUT("/:id", domainHandler.Update)
				domains.DELETE("/:id", domainHandler.Delete)
				domains.PUT("/:id/php", phpHandler.SwitchDomain)
				domains.POST("/:id/ssl/wildcard", domainHandler.WildcardSSL)
			}

			// Servers (admin only for create/delete)
			servers := protected.Group("/servers")
			{
				servers.GET("", serverHandler.List)
				servers.GET("/:id", serverHandler.GetByID)
				servers.POST("", middleware.RequireAdmin(), serverHandler.Create)
				servers.PUT("/:id", middleware.RequireAdmin(), serverHandler.Update)
				servers.DELETE("/:id", middleware.RequireAdmin(), serverHandler.Delete)
				servers.POST("/test-connection", middleware.RequireAdmin(), serverHandler.TestConnection)
				servers.POST("/:id/heartbeat", serverHandler.Heartbeat)
				servers.GET("/:id/metrics/latest", serverHandler.LatestMetrics)

				// Phase 7: Monitoring & Terminal
				servers.GET("/:id/metrics/live", metricsHandler.LiveMetrics)
				servers.GET("/:id/metrics/history", metricsHandler.HistoryMetrics)
				servers.GET("/:id/services", metricsHandler.ServiceStatuses)
				servers.GET("/:id/terminal", sshHandler.HandleTerminal)

				// Server Modules
				servers.GET("/:id/modules", modulesHandler.ListModules)
				servers.POST("/:id/modules", modulesHandler.EnableModule)
				servers.PUT("/:id/modules", modulesHandler.SetModules)
				servers.DELETE("/:id/modules/:module", modulesHandler.DisableModule)

				// Server Setup / Provisioning
				servers.GET("/:id/setup", setupHandler.GetSetupLogs)
				servers.POST("/:id/setup", setupHandler.TriggerSetup)
			}

			// Modules discovery (global)
			protected.GET("/modules/available", modulesHandler.ListAvailable)
			protected.GET("/modules/active", modulesHandler.ListActive)
			protected.GET("/modules/counts", modulesHandler.ModuleCounts)

			// Phase 2: Databases
			databases := protected.Group("/databases")
			{
				databases.POST("", databaseHandler.Create)
				databases.GET("", databaseHandler.List)
				databases.DELETE("/:id", databaseHandler.Delete)
			}

			// Phase 2: Email
			emails := protected.Group("/emails")
			{
				emails.POST("", emailHandler.CreateAccount)
				emails.GET("", emailHandler.ListAccounts)
				emails.DELETE("/:id", emailHandler.DeleteAccount)
				emails.PUT("/:id/toggle", emailHandler.ToggleAccount)
				emails.PUT("/:id/password", emailHandler.ChangePassword)
				emails.PUT("/:id/quota", emailHandler.UpdateQuota)

				emails.POST("/forwarders", emailHandler.CreateForwarder)
				emails.GET("/forwarders", emailHandler.ListForwarders)
				emails.DELETE("/forwarders/:id", emailHandler.DeleteForwarder)

				emails.POST("/aliases", emailHandler.CreateAlias)
				emails.GET("/aliases", emailHandler.ListAliases)
				emails.DELETE("/aliases/:id", emailHandler.DeleteAlias)

				emails.POST("/autoresponders", emailHandler.CreateAutoresponder)
				emails.GET("/autoresponders", emailHandler.ListAutoresponders)
				emails.DELETE("/autoresponders/:id", emailHandler.DeleteAutoresponder)
				emails.PUT("/autoresponders/:id/toggle", emailHandler.ToggleAutoresponder)

				emails.GET("/dns/:domain", emailHandler.GetDNSStatus)

				emails.GET("/catchall/:domain_id", emailHandler.GetCatchAll)
				emails.PUT("/catchall/:domain_id", emailHandler.SetCatchAll)

				emails.POST("/webmail/deploy", emailHandler.DeployWebmail)
				emails.GET("/webmail/status", emailHandler.WebmailStatus)
				emails.POST("/webmail/stop", emailHandler.StopWebmail)
			}

			// WAF (ModSecurity + OWASP CRS)
			waf := protected.Group("/waf")
			{
				waf.GET("/config/:server_id", wafHandler.GetConfig)
				waf.PUT("/config/:server_id", wafHandler.UpdateConfig)
				waf.GET("/rules/:server_id", wafHandler.ListDisabledRules)
				waf.POST("/rules/:server_id", wafHandler.DisableRule)
				waf.DELETE("/rules/:id", wafHandler.EnableRule)
				waf.GET("/whitelist/:server_id", wafHandler.ListWhitelist)
				waf.POST("/whitelist/:server_id", wafHandler.AddWhitelist)
				waf.DELETE("/whitelist/:id", wafHandler.RemoveWhitelist)
				waf.GET("/logs/:server_id", wafHandler.ListLogs)
			}

			// Cloudflare Integration
			cf := protected.Group("/cloudflare")
			{
				cf.POST("/verify", cfHandler.Verify)
				cf.POST("/zones", cfHandler.ListZones)
				cf.POST("/zones/get", cfHandler.GetZone)
				cf.POST("/dns/list", cfHandler.ListDNS)
				cf.POST("/dns/create", cfHandler.CreateDNS)
				cf.POST("/dns/update", cfHandler.UpdateDNS)
				cf.POST("/dns/delete", cfHandler.DeleteDNS)
				cf.POST("/ssl/get", cfHandler.GetSSL)
				cf.POST("/ssl/set", cfHandler.SetSSL)
				cf.POST("/cache/purge-all", cfHandler.PurgeAll)
				cf.POST("/cache/purge-urls", cfHandler.PurgeURLs)
				cf.POST("/cache/ttl", cfHandler.SetCacheTTL)
				cf.POST("/devmode", cfHandler.SetDevMode)
				cf.POST("/security", cfHandler.SetSecurity)
				cf.POST("/firewall/list", cfHandler.ListFirewall)
				cf.POST("/analytics", cfHandler.Analytics)
				cf.POST("/settings/get", cfHandler.GetSettings)
				cf.POST("/settings/update", cfHandler.UpdateSetting)
				// Tunnels
				cf.POST("/tunnels/list", cfHandler.ListTunnels)
				cf.POST("/tunnels/create", cfHandler.CreateTunnel)
				cf.POST("/tunnels/delete", cfHandler.DeleteTunnel)
				cf.POST("/tunnels/get", cfHandler.GetTunnel)
				cf.POST("/tunnels/config", cfHandler.GetTunnelConfig)
				cf.POST("/tunnels/config/update", cfHandler.UpdateTunnelConfig)
				cf.POST("/tunnels/token", cfHandler.GetTunnelToken)
				cf.POST("/tunnels/connections", cfHandler.ListTunnelConnections)
				cf.POST("/tunnels/dns-route", cfHandler.CreateTunnelDNSRoute)
				cf.POST("/tunnels/install", cfHandler.InstallCloudflared)
				cf.POST("/tunnels/run", cfHandler.RunTunnel)
				cf.POST("/tunnels/stop", cfHandler.StopTunnel)
				cf.POST("/tunnels/status", cfHandler.TunnelStatus)
			}

			// Cron Jobs
			cron := protected.Group("/cron")
			{
				cron.POST("/list", cronHandler.List)
				cron.POST("/add", cronHandler.Add)
				cron.POST("/delete", cronHandler.Delete)
				cron.POST("/update", cronHandler.Update)
				cron.POST("/users", cronHandler.ListUsers)
				cron.POST("/logs", cronHandler.Logs)
			}

			// Systemctl Services
			sysctl := protected.Group("/systemctl")
			{
				sysctl.POST("/list", systemctlHandler.List)
				sysctl.POST("/status", systemctlHandler.Status)
				sysctl.POST("/start", systemctlHandler.Start)
				sysctl.POST("/stop", systemctlHandler.Stop)
				sysctl.POST("/restart", systemctlHandler.Restart)
				sysctl.POST("/reload", systemctlHandler.Reload)
				sysctl.POST("/enable", systemctlHandler.Enable)
				sysctl.POST("/disable", systemctlHandler.Disable)
				sysctl.POST("/logs", systemctlHandler.Logs)
				sysctl.POST("/failed", systemctlHandler.Failed)
				sysctl.POST("/timers", systemctlHandler.Timers)
				sysctl.POST("/daemon-reload", systemctlHandler.DaemonReload)
			}

			// Backup Manager (actual backup/restore)
			bkm := protected.Group("/backup-manager")
			{
				bkm.POST("/database", backupMgrHandler.BackupDatabase)
				bkm.POST("/site", backupMgrHandler.BackupSite)
				bkm.POST("/full", backupMgrHandler.BackupFull)
				bkm.POST("/restore/database", backupMgrHandler.RestoreDatabase)
				bkm.POST("/restore/site", backupMgrHandler.RestoreSite)
				bkm.POST("/files", backupMgrHandler.ListFiles)
				bkm.POST("/files/delete", backupMgrHandler.DeleteFile)
			}

			// Database Manager (query runner, web tools)
			dbm := protected.Group("/db-manager")
			{
				dbm.POST("/query", dbManagerHandler.RunQuery)
				dbm.POST("/tables", dbManagerHandler.ListTables)
				dbm.POST("/describe", dbManagerHandler.DescribeTable)
				dbm.POST("/size", dbManagerHandler.GetDBSize)
				dbm.POST("/databases", dbManagerHandler.ListDBsOnServer)
				dbm.POST("/users", dbManagerHandler.ListUsers)
				dbm.POST("/users/create", dbManagerHandler.CreateUser)
				dbm.POST("/export", dbManagerHandler.ExportDB)
				dbm.POST("/import", dbManagerHandler.ImportDB)
				dbm.POST("/tools/deploy", dbManagerHandler.DeployTool)
				dbm.POST("/tools/status", dbManagerHandler.ToolStatus)
				dbm.POST("/tools/stop", dbManagerHandler.StopTool)
			}

			// File Transfers (rsync)
			transfers := protected.Group("/transfers")
			{
				transfers.GET("", transferHandler.List)
				transfers.POST("", transferHandler.Create)
				transfers.GET("/:id", transferHandler.Get)
				transfers.DELETE("/:id", transferHandler.Delete)
				transfers.POST("/:id/cancel", transferHandler.Cancel)
				transfers.POST("/:id/retry", transferHandler.Retry)
				transfers.POST("/preview", transferHandler.Preview)
				transfers.POST("/disk-usage", transferHandler.DiskUsage)
				transfers.GET("/schedules", transferHandler.ListSchedules)
				transfers.POST("/schedules", transferHandler.CreateSchedule)
				transfers.DELETE("/schedules/:id", transferHandler.DeleteSchedule)
				transfers.PUT("/schedules/:id/toggle", transferHandler.ToggleSchedule)
			}

			// Phase 2: Backups
			backups := protected.Group("/backups")
			{
				backups.POST("", backupHandler.Create)
				backups.GET("", backupHandler.List)
				backups.DELETE("/:id", backupHandler.Delete)
			}

			// File Manager (SSH-based)
			fm := protected.Group("/files")
			{
				fm.POST("/list", fileManagerHandler.List)
				fm.POST("/read", fileManagerHandler.Read)
				fm.POST("/write", fileManagerHandler.Write)
				fm.POST("/create", fileManagerHandler.Create)
				fm.POST("/rename", fileManagerHandler.Rename)
				fm.POST("/copy", fileManagerHandler.Copy)
				fm.POST("/move", fileManagerHandler.Move)
				fm.POST("/delete", fileManagerHandler.Delete)
				fm.POST("/chmod", fileManagerHandler.Chmod)
				fm.POST("/chown", fileManagerHandler.Chown)
				fm.POST("/search", fileManagerHandler.Search)
				fm.POST("/info", fileManagerHandler.Info)
				fm.POST("/extract", fileManagerHandler.Extract)
				fm.POST("/compress", fileManagerHandler.Compress)
				fm.POST("/grep", fileManagerHandler.Grep)
			}

			// Phase 3: Applications
			apps := protected.Group("/apps")
			{
				apps.POST("", appHandler.Create)
				apps.GET("", appHandler.List)
				apps.GET("/:id", appHandler.GetByID)
				apps.PUT("/:id", appHandler.Update)
				apps.DELETE("/:id", appHandler.Delete)
			}

			deploys := protected.Group("/deployments")
			{
				deploys.POST("", deployHandler.Create)
				deploys.GET("", deployHandler.List)
				deploys.GET("/:id", deployHandler.GetByID)
				deploys.POST("/:id/redeploy", deployHandler.Redeploy)
				deploys.GET("/:id/logs", deployHandler.GetLogs)
				deploys.GET("/:id/ws", deployWSHandler.HandleDeployLogs)
			}

			// Phase 3: Security
			security := protected.Group("/security")
			{
				security.POST("/rules", middleware.RequireAdmin(), securityHandler.CreateRule)
				security.GET("/rules", securityHandler.ListRules)
				security.DELETE("/rules/:id", middleware.RequireAdmin(), securityHandler.DeleteRule)
				security.GET("/events", securityHandler.ListEvents)
			}

			// Phase 3: Settings
			settings := protected.Group("/settings")
			{
				settings.GET("/profile", settingsHandler.GetProfile)
				settings.PUT("/profile", settingsHandler.UpdateProfile)
				settings.GET("/api-keys", apiKeyHandler.List)
				settings.POST("/api-keys", apiKeyHandler.Create)
				settings.DELETE("/api-keys/:id", apiKeyHandler.Revoke)
				// System settings — admin only
				settings.GET("/system", middleware.RequireAdmin(), settingsHandler.GetSystemSettings)
				settings.PUT("/system", middleware.RequireAdmin(), settingsHandler.UpdateSystemSettings)
				settings.POST("/system/test-smtp", middleware.RequireAdmin(), settingsHandler.TestSMTP)
			}

			// Alert rules and incidents
			alerts := protected.Group("/alerts")
			{
				alerts.POST("/rules", alertHandler.CreateRule)
				alerts.GET("/rules", alertHandler.ListRules)
				alerts.PUT("/rules/:id", alertHandler.UpdateRule)
				alerts.DELETE("/rules/:id", alertHandler.DeleteRule)
				alerts.GET("/incidents", alertHandler.ListIncidents)
			}

			// FTP accounts (per-server)
			protected.GET("/servers/:id/ftp", ftpHandler.List)
			protected.POST("/servers/:id/ftp", ftpHandler.Create)
			protected.DELETE("/servers/:id/ftp/:ftpID", ftpHandler.Delete)

			// Reseller management
			reseller := protected.Group("/reseller")
			{
				reseller.POST("/clients", resellerHandler.AllocateClient)
				reseller.GET("/clients", resellerHandler.ListClients)
				reseller.PUT("/clients/:id", resellerHandler.UpdateClientQuota)
				reseller.DELETE("/clients/:id", resellerHandler.DeleteClient)
				reseller.GET("/clients/:id/usage", resellerHandler.GetClientUsage)
			}

			// Billing / Stripe
			billing := protected.Group("/billing")
			{
				billing.POST("/plans", middleware.RequireAdmin(), billingHandler.CreatePlan)
				billing.GET("/plans", billingHandler.ListPlans)
				billing.GET("/invoices", billingHandler.ListInvoices)
				billing.POST("/checkout", stripeHandler.CreateCheckout)
				billing.GET("/subscription", stripeHandler.GetSubscription)
				billing.DELETE("/subscription", stripeHandler.CancelSubscription)
			}

			// Webhook delivery log
			protected.GET("/apps/:id/webhook-deliveries", webhookHandler.ListDeliveries)

			// Dashboard
			protected.GET("/dashboard/stats", serverHandler.DashboardStats)

			// Audit logs
			protected.GET("/audit-logs", auditHandler.List)

			// Phase 8: Docker Management
			if dockerHandler != nil {
				docker := protected.Group("/docker")
				{
					// Containers
					docker.GET("/containers", dockerHandler.ListContainers)
					docker.GET("/containers/:id", dockerHandler.InspectContainer)
					docker.POST("/containers", dockerHandler.CreateContainer)
					docker.POST("/containers/:id/start", dockerHandler.StartContainer)
					docker.POST("/containers/:id/stop", dockerHandler.StopContainer)
					docker.POST("/containers/:id/restart", dockerHandler.RestartContainer)
					docker.DELETE("/containers/:id", dockerHandler.RemoveContainer)
					docker.GET("/containers/:id/logs", dockerHandler.ContainerLogs)
					docker.GET("/containers/:id/stats", dockerHandler.ContainerStats)
					docker.POST("/containers/:id/rename", dockerHandler.RenameContainer)
					docker.POST("/containers/:id/duplicate", dockerHandler.DuplicateContainer)
					docker.GET("/containers/:id/processes", dockerHandler.ContainerProcesses)
					docker.GET("/containers/:id/exec", dockerExecHandler.HandleExec)

					// Images
					docker.GET("/images", dockerHandler.ListImages)
					docker.POST("/images/pull", dockerHandler.PullImage)
					docker.DELETE("/images/:id", dockerHandler.RemoveImage)
					docker.GET("/images/:id/history", dockerHandler.ImageHistory)
					docker.POST("/images/:id/tag", dockerHandler.TagImage)

					// Volumes
					docker.GET("/volumes", dockerHandler.ListVolumes)
					docker.POST("/volumes", dockerHandler.CreateVolume)
					docker.DELETE("/volumes/:name", dockerHandler.RemoveVolume)

					// Networks
					docker.GET("/networks", dockerHandler.ListNetworks)
					docker.POST("/networks", dockerHandler.CreateNetwork)
					docker.DELETE("/networks/:id", dockerHandler.RemoveNetwork)
					docker.POST("/networks/:id/connect", dockerHandler.NetworkConnect)
					docker.POST("/networks/:id/disconnect", dockerHandler.NetworkDisconnect)

					// Stacks
					docker.GET("/stacks", dockerHandler.ListStacks)
					docker.POST("/stacks", dockerHandler.DeployStack)
					docker.DELETE("/stacks/:name", dockerHandler.RemoveStack)

					// System
					docker.GET("/stats", dockerHandler.Stats)
					docker.GET("/system/info", dockerHandler.SystemInfo)
					docker.POST("/system/prune", dockerHandler.PruneSystem)

					// Templates
					docker.GET("/templates", dockerHandler.ListTemplates)
					docker.POST("/templates/:id/deploy", dockerHandler.DeployTemplate)

					// Pause / Unpause / Kill
					docker.POST("/containers/:id/pause", dockerHandler.PauseContainer)
					docker.POST("/containers/:id/unpause", dockerHandler.UnpauseContainer)
					docker.POST("/containers/:id/kill", dockerHandler.KillContainer)

					// Commit (export container to image)
					docker.POST("/containers/:id/commit", dockerHandler.CommitContainer)

					// Container file browser
					docker.GET("/containers/:id/files", dockerHandler.BrowseFiles)

					// Image build & push
					docker.POST("/images/build", dockerHandler.BuildImage)
					docker.POST("/images/push", dockerHandler.PushImage)

					// Events
					docker.GET("/events", dockerHandler.Events)
				}
			}

			// Kubernetes
			k8s := protected.Group("/k8s")
			{
				// Clusters
				k8s.POST("/clusters", k8sHandler.AddCluster)
				k8s.GET("/clusters", k8sHandler.ListClusters)
				k8s.DELETE("/clusters/:id", k8sHandler.RemoveCluster)
				k8s.GET("/clusters/:id/info", k8sHandler.ClusterInfo)

				// Per-cluster resources
				c := k8s.Group("/:cluster")
				{
					c.GET("/namespaces", k8sHandler.ListNamespaces)
					c.POST("/namespaces", k8sHandler.CreateNamespace)
					c.DELETE("/namespaces/:ns", k8sHandler.DeleteNamespace)

					c.GET("/pods", k8sHandler.ListPods)
					c.GET("/pods/:ns/:name", k8sHandler.GetPod)
					c.DELETE("/pods/:ns/:name", k8sHandler.DeletePod)
					c.GET("/pods/:ns/:name/logs", k8sHandler.PodLogs)

					c.GET("/deployments", k8sHandler.ListDeployments)
					c.PATCH("/deployments/:ns/:name/scale", k8sHandler.ScaleDeployment)
					c.POST("/deployments/:ns/:name/restart", k8sHandler.RestartDeployment)
					c.DELETE("/deployments/:ns/:name", k8sHandler.DeleteDeployment)

					c.GET("/statefulsets", k8sHandler.ListStatefulSets)
					c.PATCH("/statefulsets/:ns/:name/scale", k8sHandler.ScaleStatefulSet)
					c.DELETE("/statefulsets/:ns/:name", k8sHandler.DeleteStatefulSet)

					c.GET("/daemonsets", k8sHandler.ListDaemonSets)
					c.DELETE("/daemonsets/:ns/:name", k8sHandler.DeleteDaemonSet)

					c.GET("/replicasets", k8sHandler.ListReplicaSets)
					c.DELETE("/replicasets/:ns/:name", k8sHandler.DeleteReplicaSet)

					c.GET("/jobs", k8sHandler.ListJobs)
					c.DELETE("/jobs/:ns/:name", k8sHandler.DeleteJob)

					c.GET("/cronjobs", k8sHandler.ListCronJobs)
					c.PATCH("/cronjobs/:ns/:name/suspend", k8sHandler.SuspendCronJob)
					c.POST("/cronjobs/:ns/:name/trigger", k8sHandler.TriggerCronJob)
					c.DELETE("/cronjobs/:ns/:name", k8sHandler.DeleteCronJob)

					c.GET("/services", k8sHandler.ListServices)
					c.DELETE("/services/:ns/:name", k8sHandler.DeleteSvc)

					c.GET("/ingresses", k8sHandler.ListIngresses)
					c.DELETE("/ingresses/:ns/:name", k8sHandler.DeleteIngress)

					c.GET("/configmaps", k8sHandler.ListConfigMaps)
					c.GET("/configmaps/:ns/:name", k8sHandler.GetConfigMap)
					c.DELETE("/configmaps/:ns/:name", k8sHandler.DeleteConfigMap)

					c.GET("/secrets", k8sHandler.ListSecrets)
					c.GET("/secrets/:ns/:name", k8sHandler.GetSecret)
					c.DELETE("/secrets/:ns/:name", k8sHandler.DeleteSecret)

					c.GET("/pvcs", k8sHandler.ListPVCs)
					c.DELETE("/pvcs/:ns/:name", k8sHandler.DeletePVC)
					c.GET("/pvs", k8sHandler.ListPVs)
					c.DELETE("/pvs/:name", k8sHandler.DeletePV)
					c.GET("/storageclasses", k8sHandler.ListStorageClasses)

					c.GET("/nodes", k8sHandler.ListNodes)
					c.POST("/nodes/:name/cordon", k8sHandler.CordonNode)

					c.GET("/events", k8sHandler.ListEvents)

					c.GET("/roles", k8sHandler.ListRoles)
					c.GET("/clusterroles", k8sHandler.ListClusterRoles)
					c.GET("/rolebindings", k8sHandler.ListRoleBindings)
					c.GET("/clusterrolebindings", k8sHandler.ListClusterRoleBindings)
					c.GET("/serviceaccounts", k8sHandler.ListServiceAccounts)
					c.DELETE("/serviceaccounts/:ns/:name", k8sHandler.DeleteServiceAccount)

					c.GET("/hpa", k8sHandler.ListHPAs)
					c.DELETE("/hpa/:ns/:name", k8sHandler.DeleteHPA)

					c.GET("/networkpolicies", k8sHandler.ListNetworkPolicies)
					c.DELETE("/networkpolicies/:ns/:name", k8sHandler.DeleteNetworkPolicy)

					c.GET("/endpoints", k8sHandler.ListEndpoints)
					c.GET("/resourcequotas", k8sHandler.ListResourceQuotas)
					c.GET("/crds", k8sHandler.ListCRDs)
				}
			}



			// Team Management
			team := protected.Group("/team")
			{
				team.POST("/invite", teamHandler.Invite)
				team.POST("/accept/:id", teamHandler.Accept)
				team.GET("/members", teamHandler.ListMembers)
				team.GET("/invites", teamHandler.ListInvites)
				team.DELETE("/members/:id", teamHandler.Remove)
			}

			// PHP Version Manager
			phpRoutes := protected.Group("/servers/:id/php")
			{
				phpRoutes.GET("", phpHandler.ListVersions)
				phpRoutes.POST("", phpHandler.Install)
				phpRoutes.PUT("/default", phpHandler.SetDefault)
				phpRoutes.DELETE("/:version", phpHandler.Uninstall)
			}

			// WebSocket
			protected.GET("/ws", wsHub.HandleWebSocket)
		}
	}

	// Serve frontend SPA (React)
	frontendDir := "./frontend"
	if _, err := os.Stat(frontendDir); !os.IsNotExist(err) {
		r.Static("/assets", filepath.Join(frontendDir, "assets"))
		r.StaticFile("/favicon.ico", filepath.Join(frontendDir, "favicon.ico"))
		r.NoRoute(func(c *gin.Context) {
			// Only serve index.html for non-API requests (SPA fallback)
			if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
				c.JSON(404, gin.H{"error": "endpoint not found"})
				return
			}
			c.File(filepath.Join(frontendDir, "index.html"))
		})
	}

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.APIHost, cfg.APIPort)
	log.Printf("╔══════════════════════════════════════╗")
	log.Printf("║       NovaPanel API Server           ║")
	log.Printf("║   Listening on %s          ║", addr)
	log.Printf("╚══════════════════════════════════════╝")

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// autoRegisterHostServer adds the host machine as the "master" server.
// When SSH credentials are provided it uses SSH; otherwise it marks the server
// as local and routes all commands through nsenter (no SSH daemon required).
func autoRegisterHostServer(pool *pgxpool.Pool) {
	ctx := context.Background()

	// Read SSH credentials from environment
	sshUser := os.Getenv("SSH_USER")
	if sshUser == "" {
		sshUser = os.Getenv("HOST_USER")
	}
	if sshUser == "" {
		sshUser = "root"
	}
	sshPassword := os.Getenv("SSH_PASSWORD")

	// Try to read the host SSH private key from the mounted path
	sshKey := ""
	for _, keyPath := range []string{"/host/ssh/id_rsa", "/host/ssh/id_ed25519", "/host/ssh/id_ecdsa"} {
		if data, err := os.ReadFile(keyPath); err == nil {
			sshKey = string(data)
			break
		}
	}

	// Determine exec mode: local nsenter when no SSH credentials are available
	isLocal := sshKey == "" && sshPassword == ""
	authMethod := "password"
	if sshKey != "" {
		authMethod = "key"
	}
	if isLocal {
		authMethod = "local"
	}

	// Encrypt credentials for storage
	encPassword, encKey := "", ""
	if !isLocal {
		if cryptoKey, err := novacrypto.GetEncryptionKey(); err == nil {
			if sshPassword != "" {
				if enc, err := novacrypto.Encrypt(sshPassword, cryptoKey); err == nil {
					encPassword = enc
				}
			}
			if sshKey != "" {
				if enc, err := novacrypto.Encrypt(sshKey, cryptoKey); err == nil {
					encKey = enc
				}
			}
		}
	}

	// Detect host information
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "novapanel-host"
	}
	ip := getOutboundIP()

	osInfo := runtime.GOOS
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				osInfo = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
				break
			}
		}
	}

	var serverID string
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM servers WHERE role='master'").Scan(&count); err != nil {
		log.Printf("Warning: could not check server count: %v", err)
		return
	}

	if count > 0 {
		// Master already exists — backfill credentials and local flag
		pool.QueryRow(ctx, "SELECT id FROM servers WHERE role = 'master' LIMIT 1").Scan(&serverID)
		if serverID != "" {
			pool.Exec(ctx,
				`UPDATE servers
				 SET ssh_user     = CASE WHEN COALESCE(ssh_user,'') = '' OR ssh_user = 'root' THEN $1 ELSE ssh_user END,
				     ssh_password = CASE WHEN COALESCE(ssh_password,'') = '' THEN $2 ELSE ssh_password END,
				     ssh_key      = CASE WHEN COALESCE(ssh_key,'') = ''      THEN $3 ELSE ssh_key END,
				     auth_method  = $4,
				     is_local     = $5,
				     ip_address   = $6
				 WHERE id = $7`,
				sshUser, encPassword, encKey, authMethod, isLocal, ip, serverID,
			)
			if isLocal {
				log.Printf("✓ Host server set to local execution mode (no SSH required)")
			} else {
				log.Printf("✓ Host server SSH credentials refreshed (user=%s, auth=%s)", sshUser, authMethod)
			}
			autoDiscoverServices(pool, serverID)
		}
		return
	}

	// Insert new master server
	err := pool.QueryRow(ctx,
		`INSERT INTO servers
		   (name, hostname, ip_address, port, os, role, status, agent_status,
		    ssh_user, ssh_password, ssh_key, auth_method, is_local)
		 VALUES ($1, $2, $3, 22, $4, 'master', 'active', 'connected', $5, $6, $7, $8, $9)
		 RETURNING id`,
		"NovaPanel Host", hostname, ip, osInfo,
		sshUser, encPassword, encKey, authMethod, isLocal,
	).Scan(&serverID)
	if err != nil {
		log.Printf("Warning: failed to auto-register host server: %v", err)
		return
	}
	if isLocal {
		log.Printf("✓ Auto-registered host server: %s (%s) — local execution mode (no SSH)",
			hostname, ip)
	} else {
		log.Printf("✓ Auto-registered host server: %s (%s) — %s (user=%s, auth=%s)",
			hostname, ip, osInfo, sshUser, authMethod)
	}

	autoDiscoverServices(pool, serverID)
}

// serviceProbe defines a service to check for.
type serviceProbe struct {
	Name   string
	Ports  []int  // multiple ports to try (default + alternate)
	Type   string // "database", "cache", "webserver", "search", "queue", "app"
	Engine string // e.g. "postgresql", "mysql", "mongodb"
}

// autoDiscoverServices probes common ports on the host/network to detect
// running services, databases, and websites, then registers them in NovaPanel.
func autoDiscoverServices(pool *pgxpool.Pool, serverID string) {
	ctx := context.Background()

	// Get admin user for ownership of auto-discovered entries
	var systemUserID string
	err := pool.QueryRow(ctx, "SELECT id FROM users WHERE role = 'admin' LIMIT 1").Scan(&systemUserID)
	if err != nil {
		log.Printf("Warning: no admin user found, skipping service discovery")
		return
	}

	log.Println("🔍 Starting service discovery...")

	// ── Comprehensive service probes ──
	probes := []serviceProbe{
		// Databases
		{Name: "PostgreSQL", Ports: []int{5432, 5433, 5434}, Type: "database", Engine: "postgresql"},
		{Name: "MySQL/MariaDB", Ports: []int{3306, 3307, 3308}, Type: "database", Engine: "mysql"},
		{Name: "MongoDB", Ports: []int{27017, 27018, 27019}, Type: "database", Engine: "mongodb"},
		{Name: "CouchDB", Ports: []int{5984}, Type: "database", Engine: "couchdb"},
		{Name: "ClickHouse", Ports: []int{9000, 8123}, Type: "database", Engine: "clickhouse"},
		{Name: "CockroachDB", Ports: []int{26257, 26258}, Type: "database", Engine: "cockroachdb"},
		{Name: "InfluxDB", Ports: []int{8086}, Type: "database", Engine: "influxdb"},
		{Name: "Cassandra", Ports: []int{9042}, Type: "database", Engine: "cassandra"},
		{Name: "Neo4j", Ports: []int{7687, 7474}, Type: "database", Engine: "neo4j"},

		// Caches & KV stores
		{Name: "Redis", Ports: []int{6379, 6380}, Type: "cache", Engine: "redis"},
		{Name: "Memcached", Ports: []int{11211}, Type: "cache", Engine: "memcached"},
		{Name: "KeyDB", Ports: []int{6379}, Type: "cache", Engine: "keydb"},

		// Web servers
		{Name: "Nginx", Ports: []int{80, 8080, 8443}, Type: "webserver", Engine: "nginx"},
		{Name: "Apache", Ports: []int{80, 8080, 443}, Type: "webserver", Engine: "apache"},

		// Search engines
		{Name: "Elasticsearch", Ports: []int{9200, 9300}, Type: "search", Engine: "elasticsearch"},
		{Name: "OpenSearch", Ports: []int{9200}, Type: "search", Engine: "opensearch"},
		{Name: "Meilisearch", Ports: []int{7700}, Type: "search", Engine: "meilisearch"},
		{Name: "Solr", Ports: []int{8983}, Type: "search", Engine: "solr"},

		// Message queues
		{Name: "RabbitMQ", Ports: []int{5672, 15672}, Type: "queue", Engine: "rabbitmq"},
		{Name: "Kafka", Ports: []int{9092}, Type: "queue", Engine: "kafka"},
		{Name: "NATS", Ports: []int{4222}, Type: "queue", Engine: "nats"},

		// Application platforms
		{Name: "Node.js App", Ports: []int{3000, 3001, 5000, 5173, 4200}, Type: "app", Engine: "nodejs"},
		{Name: "Tomcat/Java", Ports: []int{8080, 8443, 9090}, Type: "app", Engine: "tomcat"},
		{Name: "PHP-FPM", Ports: []int{9000}, Type: "app", Engine: "php-fpm"},
		{Name: "Python/Gunicorn", Ports: []int{8000, 8001, 5000}, Type: "app", Engine: "python"},
		{Name: "SMTP", Ports: []int{25, 587}, Type: "app", Engine: "smtp"},
		{Name: "FTP", Ports: []int{21}, Type: "app", Engine: "ftp"},
		{Name: "SSH", Ports: []int{22, 2222}, Type: "app", Engine: "ssh"},
		{Name: "Docker API", Ports: []int{2375, 2376}, Type: "app", Engine: "docker"},
		{Name: "Grafana", Ports: []int{3000}, Type: "app", Engine: "grafana"},
		{Name: "Prometheus", Ports: []int{9090}, Type: "app", Engine: "prometheus"},
		{Name: "Jenkins", Ports: []int{8080, 8081}, Type: "app", Engine: "jenkins"},
		{Name: "GitLab", Ports: []int{8929, 80}, Type: "app", Engine: "gitlab"},
		{Name: "MinIO", Ports: []int{9000, 9001}, Type: "app", Engine: "minio"},
	}

	// Hosts to scan
	gatewayIP := getDockerGateway()
	hostsToCheck := []string{gatewayIP, "postgres", "redis", "localhost", "127.0.0.1"}

	discovered := 0
	for _, probe := range probes {
		found := false
		foundHost := ""
		foundPort := 0

		for _, port := range probe.Ports {
			for _, host := range hostsToCheck {
				addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
				conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
				if err == nil {
					conn.Close()
					found = true
					foundHost = host
					foundPort = port
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			continue
		}

		// Register databases
		if probe.Type == "database" || probe.Type == "cache" {
			var dbCount int
			pool.QueryRow(ctx, "SELECT count(*) FROM databases WHERE engine = $1", probe.Engine).Scan(&dbCount)
			if dbCount > 0 {
				continue
			}

			dbName := fmt.Sprintf("%s (auto-discovered)", probe.Name)
			_, insertErr := pool.Exec(ctx,
				`INSERT INTO databases (user_id, server_id, name, engine, db_user, charset, size_mb, status)
				 VALUES ($1, $2::uuid, $3, $4, 'auto_detected', 'utf8', 0, 'active')`,
				systemUserID, serverID, dbName, probe.Engine,
			)
			if insertErr != nil {
				log.Printf("  ⚠ Could not register %s: %v", probe.Name, insertErr)
			} else {
				log.Printf("  ✓ Discovered %s on %s:%d", probe.Name, foundHost, foundPort)
				discovered++
			}
		} else {
			log.Printf("  ✓ Discovered %s on %s:%d", probe.Name, foundHost, foundPort)
			discovered++
		}
	}

	// ── Website discovery from Nginx/Apache configs ──
	websiteCount := discoverWebsites(pool, serverID, systemUserID)
	discovered += websiteCount

	// ── Systemd / init.d service file discovery ──
	systemdCount := discoverSystemServices(pool, serverID, systemUserID)
	discovered += systemdCount

	if discovered > 0 {
		log.Printf("✓ Auto-discovered %d service(s) and website(s) on host", discovered)
	} else {
		log.Println("  No additional services detected")
	}
}

// discoverWebsites parses Nginx and Apache vhost configs mounted from the host
// to find and register existing websites/domains.
func discoverWebsites(pool *pgxpool.Pool, serverID, userID string) int {
	ctx := context.Background()
	count := 0

	// ── Nginx sites ──
	nginxDirs := []string{
		"/host/nginx/sites-enabled",
		"/host/nginx/conf.d",
		"/host/nginx/sites-available",
	}
	for _, dir := range nginxDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}
			domains := parseNginxServerNames(string(data))
			docRoot := parseNginxRoot(string(data))
			for _, domain := range domains {
				if registerDomain(pool, ctx, domain, "nginx", docRoot, serverID, userID) {
					log.Printf("  ✓ Discovered website: %s (Nginx)", domain)
					count++
				}
			}
		}
	}

	// ── Apache sites ──
	apacheDirs := []string{
		"/host/apache2/sites-enabled",
		"/host/apache2/sites-available",
		"/host/httpd/conf.d",
		"/host/httpd/conf/httpd.conf",
	}
	for _, dir := range apacheDirs {
		info, err := os.Stat(dir)
		if err != nil {
			continue
		}

		var files []string
		if info.IsDir() {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() {
					files = append(files, filepath.Join(dir, e.Name()))
				}
			}
		} else {
			files = []string{dir}
		}

		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			domains := parseApacheServerNames(string(data))
			docRoot := parseApacheDocRoot(string(data))
			for _, domain := range domains {
				if registerDomain(pool, ctx, domain, "apache", docRoot, serverID, userID) {
					log.Printf("  ✓ Discovered website: %s (Apache)", domain)
					count++
				}
			}
		}
	}

	return count
}

// parseNginxServerNames extracts server_name values from Nginx config.
func parseNginxServerNames(config string) []string {
	var domains []string
	lines := strings.Split(config, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "server_name") {
			// server_name example.com www.example.com;
			line = strings.TrimPrefix(line, "server_name")
			line = strings.TrimSuffix(strings.TrimSpace(line), ";")
			parts := strings.Fields(line)
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" && p != "_" && p != "localhost" && p != "127.0.0.1" && !strings.HasPrefix(p, "$") {
					domains = append(domains, p)
				}
			}
		}
	}
	return domains
}

// parseNginxRoot extracts the root directive from Nginx config.
func parseNginxRoot(config string) string {
	lines := strings.Split(config, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "root ") {
			root := strings.TrimPrefix(line, "root ")
			return strings.TrimSuffix(strings.TrimSpace(root), ";")
		}
	}
	return "/var/www/html"
}

// parseApacheServerNames extracts ServerName and ServerAlias from Apache config.
func parseApacheServerNames(config string) []string {
	var domains []string
	lines := strings.Split(config, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "servername ") {
			name := strings.TrimSpace(line[11:])
			if name != "" && name != "localhost" {
				domains = append(domains, name)
			}
		}
		if strings.HasPrefix(lower, "serveralias ") {
			aliases := strings.Fields(line[12:])
			for _, a := range aliases {
				a = strings.TrimSpace(a)
				if a != "" && a != "localhost" {
					domains = append(domains, a)
				}
			}
		}
	}
	return domains
}

// parseApacheDocRoot extracts DocumentRoot from Apache config.
func parseApacheDocRoot(config string) string {
	lines := strings.Split(config, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "documentroot ") {
			root := strings.TrimSpace(line[13:])
			return strings.Trim(root, "\"'")
		}
	}
	return "/var/www/html"
}

// registerDomain adds a domain to the database if it doesn't already exist.
func registerDomain(pool *pgxpool.Pool, ctx context.Context, name, webServer, docRoot, serverID, userID string) bool {
	// Skip wildcard, IP-only, or default entries
	if strings.Contains(name, "*") || name == "default" || name == "_" {
		return false
	}

	// Check if domain already exists
	var existing int
	pool.QueryRow(ctx, "SELECT count(*) FROM domains WHERE name = $1", name).Scan(&existing)
	if existing > 0 {
		return false
	}

	_, err := pool.Exec(ctx,
		`INSERT INTO domains (user_id, server_id, name, type, document_root, web_server, status)
		 VALUES ($1, $2::uuid, $3, 'primary', $4, $5, 'active')`,
		userID, serverID, name, docRoot, webServer,
	)
	return err == nil
}

// getDockerGateway returns the Docker host gateway IP.
func getDockerGateway() string {
	// Read the default route to find the gateway
	data, err := os.ReadFile("/proc/net/route")
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 3 && fields[1] == "00000000" {
				// Default route — parse the gateway hex
				gw := fields[2]
				if len(gw) == 8 {
					var a, b, c, d uint64
					fmt.Sscanf(gw[6:8], "%x", &a)
					fmt.Sscanf(gw[4:6], "%x", &b)
					fmt.Sscanf(gw[2:4], "%x", &c)
					fmt.Sscanf(gw[0:2], "%x", &d)
					return fmt.Sprintf("%d.%d.%d.%d", a, b, c, d)
				}
			}
		}
	}
	return "172.17.0.1" // Docker default gateway fallback
}

// getOutboundIP returns the preferred outbound IP of this machine.
func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// knownService maps a systemd/init service name pattern to its engine metadata.
type knownService struct {
	Patterns []string // filename patterns to match (lowercase, without .service)
	Name     string
	Type     string // "database", "cache", "webserver", "search", "queue", "app"
	Engine   string
}

// discoverSystemServices reads systemd unit files and SysV init scripts
// to detect installed services, even those on non-standard ports or stopped.
func discoverSystemServices(pool *pgxpool.Pool, serverID, userID string) int {
	ctx := context.Background()

	known := []knownService{
		// Databases
		{Patterns: []string{"postgresql", "postgres", "pgsql"}, Name: "PostgreSQL", Type: "database", Engine: "postgresql"},
		{Patterns: []string{"mysql", "mysqld", "mariadb"}, Name: "MySQL/MariaDB", Type: "database", Engine: "mysql"},
		{Patterns: []string{"mongod", "mongodb"}, Name: "MongoDB", Type: "database", Engine: "mongodb"},
		{Patterns: []string{"couchdb"}, Name: "CouchDB", Type: "database", Engine: "couchdb"},
		{Patterns: []string{"clickhouse-server", "clickhouse"}, Name: "ClickHouse", Type: "database", Engine: "clickhouse"},
		{Patterns: []string{"cockroach"}, Name: "CockroachDB", Type: "database", Engine: "cockroachdb"},
		{Patterns: []string{"influxdb", "influxd"}, Name: "InfluxDB", Type: "database", Engine: "influxdb"},
		{Patterns: []string{"cassandra"}, Name: "Cassandra", Type: "database", Engine: "cassandra"},
		{Patterns: []string{"neo4j"}, Name: "Neo4j", Type: "database", Engine: "neo4j"},
		{Patterns: []string{"firebird"}, Name: "Firebird", Type: "database", Engine: "firebird"},
		{Patterns: []string{"orientdb"}, Name: "OrientDB", Type: "database", Engine: "orientdb"},
		{Patterns: []string{"arangodb", "arangodb3"}, Name: "ArangoDB", Type: "database", Engine: "arangodb"},
		{Patterns: []string{"rethinkdb"}, Name: "RethinkDB", Type: "database", Engine: "rethinkdb"},

		// Caches
		{Patterns: []string{"redis", "redis-server"}, Name: "Redis", Type: "cache", Engine: "redis"},
		{Patterns: []string{"memcached"}, Name: "Memcached", Type: "cache", Engine: "memcached"},
		{Patterns: []string{"keydb", "keydb-server"}, Name: "KeyDB", Type: "cache", Engine: "keydb"},
		{Patterns: []string{"varnish"}, Name: "Varnish", Type: "cache", Engine: "varnish"},

		// Web servers
		{Patterns: []string{"nginx"}, Name: "Nginx", Type: "webserver", Engine: "nginx"},
		{Patterns: []string{"apache2", "httpd"}, Name: "Apache", Type: "webserver", Engine: "apache"},
		{Patterns: []string{"caddy"}, Name: "Caddy", Type: "webserver", Engine: "caddy"},
		{Patterns: []string{"lighttpd"}, Name: "Lighttpd", Type: "webserver", Engine: "lighttpd"},
		{Patterns: []string{"openlitespeed", "lsws", "litespeed"}, Name: "LiteSpeed", Type: "webserver", Engine: "litespeed"},
		{Patterns: []string{"haproxy"}, Name: "HAProxy", Type: "webserver", Engine: "haproxy"},
		{Patterns: []string{"traefik"}, Name: "Traefik", Type: "webserver", Engine: "traefik"},

		// Search
		{Patterns: []string{"elasticsearch"}, Name: "Elasticsearch", Type: "search", Engine: "elasticsearch"},
		{Patterns: []string{"opensearch"}, Name: "OpenSearch", Type: "search", Engine: "opensearch"},
		{Patterns: []string{"meilisearch"}, Name: "Meilisearch", Type: "search", Engine: "meilisearch"},
		{Patterns: []string{"solr"}, Name: "Solr", Type: "search", Engine: "solr"},

		// Queues
		{Patterns: []string{"rabbitmq", "rabbitmq-server"}, Name: "RabbitMQ", Type: "queue", Engine: "rabbitmq"},
		{Patterns: []string{"kafka"}, Name: "Kafka", Type: "queue", Engine: "kafka"},
		{Patterns: []string{"nats", "nats-server"}, Name: "NATS", Type: "queue", Engine: "nats"},
		{Patterns: []string{"mosquitto"}, Name: "Mosquitto MQTT", Type: "queue", Engine: "mosquitto"},

		// Apps & tools
		{Patterns: []string{"docker", "dockerd", "containerd"}, Name: "Docker", Type: "app", Engine: "docker"},
		{Patterns: []string{"grafana-server", "grafana"}, Name: "Grafana", Type: "app", Engine: "grafana"},
		{Patterns: []string{"prometheus"}, Name: "Prometheus", Type: "app", Engine: "prometheus"},
		{Patterns: []string{"jenkins"}, Name: "Jenkins", Type: "app", Engine: "jenkins"},
		{Patterns: []string{"gitlab", "gitlab-runsvdir"}, Name: "GitLab", Type: "app", Engine: "gitlab"},
		{Patterns: []string{"minio"}, Name: "MinIO", Type: "app", Engine: "minio"},
		{Patterns: []string{"postfix"}, Name: "Postfix (SMTP)", Type: "app", Engine: "postfix"},
		{Patterns: []string{"dovecot"}, Name: "Dovecot (IMAP)", Type: "app", Engine: "dovecot"},
		{Patterns: []string{"fail2ban"}, Name: "Fail2Ban", Type: "app", Engine: "fail2ban"},
		{Patterns: []string{"ufw"}, Name: "UFW Firewall", Type: "app", Engine: "ufw"},
		{Patterns: []string{"certbot"}, Name: "Certbot (SSL)", Type: "app", Engine: "certbot"},
		{Patterns: []string{"php-fpm", "php8.3-fpm", "php8.2-fpm", "php8.1-fpm", "php8.0-fpm", "php7.4-fpm"}, Name: "PHP-FPM", Type: "app", Engine: "php-fpm"},
		{Patterns: []string{"tomcat", "tomcat9", "tomcat10"}, Name: "Tomcat", Type: "app", Engine: "tomcat"},
		{Patterns: []string{"named", "bind9"}, Name: "BIND9 (DNS)", Type: "app", Engine: "bind"},
		{Patterns: []string{"pure-ftpd", "vsftpd", "proftpd"}, Name: "FTP Server", Type: "app", Engine: "ftp"},
		{Patterns: []string{"supervisor", "supervisord"}, Name: "Supervisor", Type: "app", Engine: "supervisor"},
		{Patterns: []string{"pm2"}, Name: "PM2", Type: "app", Engine: "pm2"},
		{Patterns: []string{"cron", "crond"}, Name: "Cron", Type: "app", Engine: "cron"},
		{Patterns: []string{"sshd", "ssh"}, Name: "OpenSSH", Type: "app", Engine: "ssh"},
	}

	// Collect service file basenames from all source directories
	serviceFiles := make(map[string]bool)

	// Systemd directories
	systemdDirs := []string{
		"/host/systemd/system",
		"/host/systemd/lib",
	}
	for _, dir := range systemdDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name := entry.Name()
			// Only look at .service files
			if strings.HasSuffix(name, ".service") {
				base := strings.TrimSuffix(name, ".service")
				// Also handle patterns like postgresql@14-main.service
				if idx := strings.Index(base, "@"); idx > 0 {
					base = base[:idx]
				}
				serviceFiles[strings.ToLower(base)] = true
			}
		}
	}

	// SysV init.d scripts
	initDir := "/host/init.d"
	entries, err := os.ReadDir(initDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				serviceFiles[strings.ToLower(entry.Name())] = true
			}
		}
	}

	if len(serviceFiles) == 0 {
		return 0
	}

	log.Printf("  📋 Scanning %d system service files...", len(serviceFiles))

	count := 0
	for _, svc := range known {
		matched := false
		for _, pattern := range svc.Patterns {
			if serviceFiles[pattern] {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		// For databases/caches, register in the databases table
		if svc.Type == "database" || svc.Type == "cache" {
			var dbCount int
			pool.QueryRow(ctx, "SELECT count(*) FROM databases WHERE engine = $1", svc.Engine).Scan(&dbCount)
			if dbCount > 0 {
				continue // Already registered by port scan
			}

			dbName := fmt.Sprintf("%s (systemd)", svc.Name)
			_, insertErr := pool.Exec(ctx,
				`INSERT INTO databases (user_id, server_id, name, engine, db_user, charset, size_mb, status)
				 VALUES ($1, $2::uuid, $3, $4, 'auto_detected', 'utf8', 0, 'active')`,
				userID, serverID, dbName, svc.Engine,
			)
			if insertErr == nil {
				log.Printf("  ✓ Found %s (from service files)", svc.Name)
				count++
			}
		} else {
			log.Printf("  ✓ Found %s (from service files)", svc.Name)
			count++
		}
	}

	return count
}
