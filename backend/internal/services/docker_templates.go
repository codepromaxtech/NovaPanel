package services

// AppTemplate represents a one-click deploy template (like Portainer's app templates).
type AppTemplate struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Logo        string            `json:"logo"`
	Image       string            `json:"image"`
	Ports       map[string]string `json:"ports,omitempty"`
	Env         []TemplateEnv     `json:"env,omitempty"`
	Volumes     []string          `json:"volumes,omitempty"`
	Restart     string            `json:"restart"`
	Command     string            `json:"command,omitempty"`
}

type TemplateEnv struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Default string `json:"default"`
}

// GetAppTemplates returns the built-in app template catalog.
func GetAppTemplates() []AppTemplate {
	return []AppTemplate{
		// Web Servers
		{ID: "nginx", Title: "Nginx", Description: "High-performance web server and reverse proxy", Category: "Web Server", Logo: "🌐", Image: "nginx:alpine", Ports: map[string]string{"80/tcp": "8080"}, Restart: "unless-stopped"},
		{ID: "apache", Title: "Apache HTTP", Description: "The most widely used web server", Category: "Web Server", Logo: "🪶", Image: "httpd:alpine", Ports: map[string]string{"80/tcp": "8081"}, Restart: "unless-stopped"},
		{ID: "caddy", Title: "Caddy", Description: "Fast web server with automatic HTTPS", Category: "Web Server", Logo: "🔒", Image: "caddy:alpine", Ports: map[string]string{"80/tcp": "8082", "443/tcp": "8443"}, Restart: "unless-stopped"},
		{ID: "traefik", Title: "Traefik", Description: "Modern reverse proxy and load balancer", Category: "Web Server", Logo: "🔀", Image: "traefik:latest", Ports: map[string]string{"80/tcp": "80", "8080/tcp": "8180"}, Restart: "unless-stopped",
			Env: []TemplateEnv{{Name: "TRAEFIK_API_INSECURE", Label: "Enable Dashboard", Default: "true"}}},

		// Databases
		{ID: "mysql", Title: "MySQL", Description: "Popular open-source relational database", Category: "Database", Logo: "🐬", Image: "mysql:8", Ports: map[string]string{"3306/tcp": "3306"},
			Env:     []TemplateEnv{{Name: "MYSQL_ROOT_PASSWORD", Label: "Root Password", Default: "rootpass"}, {Name: "MYSQL_DATABASE", Label: "Database Name", Default: "mydb"}},
			Volumes: []string{"mysql_data:/var/lib/mysql"}, Restart: "unless-stopped"},
		{ID: "postgres", Title: "PostgreSQL", Description: "Advanced open-source relational database", Category: "Database", Logo: "🐘", Image: "postgres:16-alpine", Ports: map[string]string{"5432/tcp": "5433"},
			Env:     []TemplateEnv{{Name: "POSTGRES_PASSWORD", Label: "Password", Default: "postgres"}, {Name: "POSTGRES_DB", Label: "Database", Default: "mydb"}},
			Volumes: []string{"pg_data:/var/lib/postgresql/data"}, Restart: "unless-stopped"},
		{ID: "mariadb", Title: "MariaDB", Description: "Community fork of MySQL", Category: "Database", Logo: "🦭", Image: "mariadb:latest", Ports: map[string]string{"3306/tcp": "3307"},
			Env:     []TemplateEnv{{Name: "MARIADB_ROOT_PASSWORD", Label: "Root Password", Default: "rootpass"}},
			Volumes: []string{"mariadb_data:/var/lib/mysql"}, Restart: "unless-stopped"},
		{ID: "mongo", Title: "MongoDB", Description: "NoSQL document database", Category: "Database", Logo: "🍃", Image: "mongo:latest", Ports: map[string]string{"27017/tcp": "27017"},
			Volumes: []string{"mongo_data:/data/db"}, Restart: "unless-stopped"},

		// Cache & Queue
		{ID: "redis", Title: "Redis", Description: "In-memory key-value store and cache", Category: "Cache", Logo: "🔴", Image: "redis:7-alpine", Ports: map[string]string{"6379/tcp": "6380"}, Restart: "unless-stopped"},
		{ID: "memcached", Title: "Memcached", Description: "High-performance distributed caching", Category: "Cache", Logo: "⚡", Image: "memcached:alpine", Ports: map[string]string{"11211/tcp": "11211"}, Restart: "unless-stopped"},
		{ID: "rabbitmq", Title: "RabbitMQ", Description: "Message broker with management UI", Category: "Queue", Logo: "🐰", Image: "rabbitmq:3-management-alpine", Ports: map[string]string{"5672/tcp": "5672", "15672/tcp": "15672"}, Restart: "unless-stopped"},

		// Monitoring
		{ID: "grafana", Title: "Grafana", Description: "Analytics and monitoring dashboards", Category: "Monitoring", Logo: "📊", Image: "grafana/grafana:latest", Ports: map[string]string{"3000/tcp": "3001"},
			Volumes: []string{"grafana_data:/var/lib/grafana"}, Restart: "unless-stopped"},
		{ID: "prometheus", Title: "Prometheus", Description: "Monitoring and alerting toolkit", Category: "Monitoring", Logo: "🔥", Image: "prom/prometheus:latest", Ports: map[string]string{"9090/tcp": "9090"}, Restart: "unless-stopped"},
		{ID: "uptime-kuma", Title: "Uptime Kuma", Description: "Self-hosted uptime monitoring tool", Category: "Monitoring", Logo: "📈", Image: "louislam/uptime-kuma:latest", Ports: map[string]string{"3001/tcp": "3002"},
			Volumes: []string{"uptime_kuma_data:/app/data"}, Restart: "unless-stopped"},

		// CMS & Apps
		{ID: "wordpress", Title: "WordPress", Description: "World's most popular CMS", Category: "CMS", Logo: "📝", Image: "wordpress:latest", Ports: map[string]string{"80/tcp": "8083"},
			Env: []TemplateEnv{{Name: "WORDPRESS_DB_HOST", Label: "DB Host", Default: "db"}, {Name: "WORDPRESS_DB_PASSWORD", Label: "DB Password", Default: "rootpass"}}, Restart: "unless-stopped"},
		{ID: "ghost", Title: "Ghost", Description: "Modern publishing platform", Category: "CMS", Logo: "👻", Image: "ghost:latest", Ports: map[string]string{"2368/tcp": "2368"}, Restart: "unless-stopped"},

		// DevOps
		{ID: "gitea", Title: "Gitea", Description: "Lightweight self-hosted Git service", Category: "DevOps", Logo: "🍵", Image: "gitea/gitea:latest", Ports: map[string]string{"3000/tcp": "3003", "22/tcp": "2222"},
			Volumes: []string{"gitea_data:/data"}, Restart: "unless-stopped"},
		{ID: "drone", Title: "Drone CI", Description: "Container-native CI/CD platform", Category: "DevOps", Logo: "🤖", Image: "drone/drone:latest", Ports: map[string]string{"80/tcp": "8084"},
			Env: []TemplateEnv{{Name: "DRONE_SERVER_HOST", Label: "Server Host", Default: "localhost"}}, Restart: "unless-stopped"},
		{ID: "registry", Title: "Docker Registry", Description: "Private container image registry", Category: "DevOps", Logo: "📦", Image: "registry:2", Ports: map[string]string{"5000/tcp": "5000"},
			Volumes: []string{"registry_data:/var/lib/registry"}, Restart: "unless-stopped"},

		// Tools
		{ID: "adminer", Title: "Adminer", Description: "Lightweight database management UI", Category: "Tools", Logo: "🗄️", Image: "adminer:latest", Ports: map[string]string{"8080/tcp": "8085"}, Restart: "unless-stopped"},
		{ID: "phpmyadmin", Title: "phpMyAdmin", Description: "MySQL/MariaDB web administration", Category: "Tools", Logo: "🔧", Image: "phpmyadmin:latest", Ports: map[string]string{"80/tcp": "8086"},
			Env: []TemplateEnv{{Name: "PMA_HOST", Label: "MySQL Host", Default: "mysql"}}, Restart: "unless-stopped"},
		{ID: "filebrowser", Title: "File Browser", Description: "Web-based file manager", Category: "Tools", Logo: "📁", Image: "filebrowser/filebrowser:latest", Ports: map[string]string{"80/tcp": "8087"}, Restart: "unless-stopped"},
		{ID: "watchtower", Title: "Watchtower", Description: "Automatic Docker container updates", Category: "Tools", Logo: "🗼", Image: "containrrr/watchtower:latest",
			Volumes: []string{"/var/run/docker.sock:/var/run/docker.sock"}, Restart: "unless-stopped"},
		{ID: "netdata", Title: "Netdata", Description: "Real-time performance monitoring", Category: "Monitoring", Logo: "📉", Image: "netdata/netdata:latest", Ports: map[string]string{"19999/tcp": "19999"},
			Volumes: []string{"/proc:/host/proc:ro", "/sys:/host/sys:ro", "/var/run/docker.sock:/var/run/docker.sock:ro"}, Restart: "unless-stopped"},
		{ID: "minio", Title: "MinIO", Description: "S3-compatible object storage", Category: "Storage", Logo: "💾", Image: "minio/minio:latest", Ports: map[string]string{"9000/tcp": "9001", "9001/tcp": "9002"},
			Env:     []TemplateEnv{{Name: "MINIO_ROOT_USER", Label: "Root User", Default: "minioadmin"}, {Name: "MINIO_ROOT_PASSWORD", Label: "Root Password", Default: "minioadmin"}},
			Volumes: []string{"minio_data:/data"}, Restart: "unless-stopped", Command: "server /data --console-address :9001"},
	}
}
