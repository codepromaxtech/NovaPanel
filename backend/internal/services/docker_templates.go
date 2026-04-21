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

		// ═══════════════════════════════════════
		// Web Servers & Reverse Proxies
		// ═══════════════════════════════════════
		{ID: "nginx", Title: "Nginx", Description: "High-performance web server and reverse proxy", Category: "Web Server", Logo: "🌐", Image: "nginx:alpine", Ports: map[string]string{"80/tcp": "8080"}, Restart: "unless-stopped"},
		{ID: "apache", Title: "Apache HTTP", Description: "The most widely used web server", Category: "Web Server", Logo: "🪶", Image: "httpd:alpine", Ports: map[string]string{"80/tcp": "8081"}, Restart: "unless-stopped"},
		{ID: "caddy", Title: "Caddy", Description: "Fast web server with automatic HTTPS", Category: "Web Server", Logo: "🔒", Image: "caddy:alpine", Ports: map[string]string{"80/tcp": "8082", "443/tcp": "8443"}, Restart: "unless-stopped"},
		{ID: "traefik", Title: "Traefik", Description: "Modern reverse proxy and load balancer", Category: "Web Server", Logo: "🔀", Image: "traefik:latest", Ports: map[string]string{"80/tcp": "80", "8080/tcp": "8180"}, Restart: "unless-stopped",
			Env: []TemplateEnv{{Name: "TRAEFIK_API_INSECURE", Label: "Enable Dashboard", Default: "true"}}},
		{ID: "nginx-proxy-manager", Title: "Nginx Proxy Manager", Description: "Easy-to-use reverse proxy with SSL and beautiful UI", Category: "Web Server", Logo: "🔀", Image: "jc21/nginx-proxy-manager:latest", Ports: map[string]string{"80/tcp": "80", "443/tcp": "443", "81/tcp": "81"},
			Volumes: []string{"npm_data:/data", "npm_letsencrypt:/etc/letsencrypt"}, Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// Databases
		// ═══════════════════════════════════════
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
		{ID: "influxdb", Title: "InfluxDB", Description: "Time-series database for metrics and IoT", Category: "Database", Logo: "📈", Image: "influxdb:latest", Ports: map[string]string{"8086/tcp": "8086"},
			Env:     []TemplateEnv{{Name: "DOCKER_INFLUXDB_INIT_USERNAME", Label: "Admin User", Default: "admin"}, {Name: "DOCKER_INFLUXDB_INIT_PASSWORD", Label: "Admin Password", Default: "adminpass"}, {Name: "DOCKER_INFLUXDB_INIT_ORG", Label: "Organization", Default: "myorg"}, {Name: "DOCKER_INFLUXDB_INIT_BUCKET", Label: "Bucket", Default: "mybucket"}},
			Volumes: []string{"influxdb_data:/var/lib/influxdb2"}, Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// Cache & Queue
		// ═══════════════════════════════════════
		{ID: "redis", Title: "Redis", Description: "In-memory key-value store and cache", Category: "Cache", Logo: "🔴", Image: "redis:7-alpine", Ports: map[string]string{"6379/tcp": "6380"}, Restart: "unless-stopped"},
		{ID: "memcached", Title: "Memcached", Description: "High-performance distributed caching", Category: "Cache", Logo: "⚡", Image: "memcached:alpine", Ports: map[string]string{"11211/tcp": "11211"}, Restart: "unless-stopped"},
		{ID: "rabbitmq", Title: "RabbitMQ", Description: "Message broker with management UI", Category: "Queue", Logo: "🐰", Image: "rabbitmq:3-management-alpine", Ports: map[string]string{"5672/tcp": "5672", "15672/tcp": "15672"}, Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// Cloud Storage & File Sharing
		// ═══════════════════════════════════════
		{ID: "nextcloud", Title: "Nextcloud", Description: "Self-hosted file sync, share, and collaboration platform", Category: "Cloud Storage", Logo: "☁️", Image: "nextcloud:latest", Ports: map[string]string{"80/tcp": "8088"},
			Env:     []TemplateEnv{{Name: "NEXTCLOUD_ADMIN_USER", Label: "Admin Username", Default: "admin"}, {Name: "NEXTCLOUD_ADMIN_PASSWORD", Label: "Admin Password", Default: "admin"}, {Name: "SQLITE_DATABASE", Label: "SQLite DB File", Default: "nextcloud"}},
			Volumes: []string{"nextcloud_data:/var/www/html"}, Restart: "unless-stopped"},
		{ID: "owncloud", Title: "ownCloud", Description: "Open-source file sync and share with enterprise features", Category: "Cloud Storage", Logo: "📂", Image: "owncloud/server:latest", Ports: map[string]string{"8080/tcp": "8089"},
			Env:     []TemplateEnv{{Name: "OWNCLOUD_DOMAIN", Label: "Domain", Default: "localhost:8089"}, {Name: "ADMIN_USERNAME", Label: "Admin Username", Default: "admin"}, {Name: "ADMIN_PASSWORD", Label: "Admin Password", Default: "admin"}},
			Volumes: []string{"owncloud_data:/mnt/data"}, Restart: "unless-stopped"},
		{ID: "seafile", Title: "Seafile", Description: "High-performance file syncing and sharing with privacy", Category: "Cloud Storage", Logo: "🌊", Image: "seafileltd/seafile-mc:latest", Ports: map[string]string{"80/tcp": "8090"},
			Env:     []TemplateEnv{{Name: "SEAFILE_ADMIN_EMAIL", Label: "Admin Email", Default: "admin@example.com"}, {Name: "SEAFILE_ADMIN_PASSWORD", Label: "Admin Password", Default: "admin"}},
			Volumes: []string{"seafile_data:/shared"}, Restart: "unless-stopped"},
		{ID: "syncthing", Title: "Syncthing", Description: "Continuous peer-to-peer file synchronization", Category: "Cloud Storage", Logo: "🔄", Image: "syncthing/syncthing:latest", Ports: map[string]string{"8384/tcp": "8384", "22000/tcp": "22000"},
			Volumes: []string{"syncthing_data:/var/syncthing"}, Restart: "unless-stopped"},
		{ID: "filebrowser", Title: "File Browser", Description: "Web-based file manager with sharing", Category: "Cloud Storage", Logo: "📁", Image: "filebrowser/filebrowser:latest", Ports: map[string]string{"80/tcp": "8087"}, Restart: "unless-stopped"},
		{ID: "minio", Title: "MinIO", Description: "S3-compatible object storage", Category: "Cloud Storage", Logo: "💾", Image: "minio/minio:latest", Ports: map[string]string{"9000/tcp": "9001", "9001/tcp": "9002"},
			Env:     []TemplateEnv{{Name: "MINIO_ROOT_USER", Label: "Root User", Default: "minioadmin"}, {Name: "MINIO_ROOT_PASSWORD", Label: "Root Password", Default: "minioadmin"}},
			Volumes: []string{"minio_data:/data"}, Restart: "unless-stopped", Command: "server /data --console-address :9001"},

		// ═══════════════════════════════════════
		// Media & Entertainment
		// ═══════════════════════════════════════
		{ID: "jellyfin", Title: "Jellyfin", Description: "Free media streaming server (movies, TV, music)", Category: "Media", Logo: "🎬", Image: "jellyfin/jellyfin:latest", Ports: map[string]string{"8096/tcp": "8096"},
			Volumes: []string{"jellyfin_config:/config", "jellyfin_cache:/cache"}, Restart: "unless-stopped"},
		{ID: "plex", Title: "Plex", Description: "Organize and stream your media collection", Category: "Media", Logo: "🎥", Image: "plexinc/pms-docker:latest", Ports: map[string]string{"32400/tcp": "32400"},
			Env:     []TemplateEnv{{Name: "PLEX_CLAIM", Label: "Plex Claim Token", Default: ""}},
			Volumes: []string{"plex_config:/config", "plex_transcode:/transcode"}, Restart: "unless-stopped"},
		{ID: "emby", Title: "Emby", Description: "Personal media server for movies, TV, music, and photos", Category: "Media", Logo: "📺", Image: "emby/embyserver:latest", Ports: map[string]string{"8096/tcp": "8097"},
			Volumes: []string{"emby_config:/config"}, Restart: "unless-stopped"},
		{ID: "navidrome", Title: "Navidrome", Description: "Lightweight self-hosted music server (Subsonic/Airsonic)", Category: "Media", Logo: "🎵", Image: "deluan/navidrome:latest", Ports: map[string]string{"4533/tcp": "4533"},
			Env:     []TemplateEnv{{Name: "ND_SCANSCHEDULE", Label: "Scan Schedule", Default: "1h"}},
			Volumes: []string{"navidrome_data:/data"}, Restart: "unless-stopped"},
		{ID: "audiobookshelf", Title: "Audiobookshelf", Description: "Self-hosted audiobook and podcast server", Category: "Media", Logo: "📚", Image: "ghcr.io/advplyr/audiobookshelf:latest", Ports: map[string]string{"80/tcp": "13378"},
			Volumes: []string{"audiobookshelf_data:/config"}, Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// Photos & Gallery
		// ═══════════════════════════════════════
		{ID: "immich", Title: "Immich", Description: "High-performance self-hosted Google Photos alternative", Category: "Photos", Logo: "📸", Image: "ghcr.io/immich-app/immich-server:release", Ports: map[string]string{"2283/tcp": "2283"},
			Env:     []TemplateEnv{{Name: "DB_HOSTNAME", Label: "DB Host", Default: "immich-postgres"}, {Name: "DB_PASSWORD", Label: "DB Password", Default: "postgres"}, {Name: "REDIS_HOSTNAME", Label: "Redis Host", Default: "immich-redis"}},
			Volumes: []string{"immich_upload:/usr/src/app/upload"}, Restart: "unless-stopped"},
		{ID: "photoprism", Title: "PhotoPrism", Description: "AI-powered photo management and browsing", Category: "Photos", Logo: "🖼️", Image: "photoprism/photoprism:latest", Ports: map[string]string{"2342/tcp": "2342"},
			Env:     []TemplateEnv{{Name: "PHOTOPRISM_ADMIN_PASSWORD", Label: "Admin Password", Default: "admin"}, {Name: "PHOTOPRISM_SITE_URL", Label: "Site URL", Default: "http://localhost:2342/"}},
			Volumes: []string{"photoprism_storage:/photoprism/storage"}, Restart: "unless-stopped"},
		{ID: "lychee", Title: "Lychee", Description: "Beautiful self-hosted photo gallery", Category: "Photos", Logo: "🍒", Image: "lycheeorg/lychee:latest", Ports: map[string]string{"80/tcp": "8091"},
			Env:     []TemplateEnv{{Name: "DB_CONNECTION", Label: "DB Type", Default: "sqlite"}, {Name: "APP_URL", Label: "App URL", Default: "http://localhost:8091"}},
			Volumes: []string{"lychee_conf:/conf", "lychee_uploads:/uploads"}, Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// Monitoring & Dashboard
		// ═══════════════════════════════════════
		{ID: "grafana", Title: "Grafana", Description: "Analytics and monitoring dashboards", Category: "Monitoring", Logo: "📊", Image: "grafana/grafana:latest", Ports: map[string]string{"3000/tcp": "3001"},
			Volumes: []string{"grafana_data:/var/lib/grafana"}, Restart: "unless-stopped"},
		{ID: "prometheus", Title: "Prometheus", Description: "Monitoring and alerting toolkit", Category: "Monitoring", Logo: "🔥", Image: "prom/prometheus:latest", Ports: map[string]string{"9090/tcp": "9090"}, Restart: "unless-stopped"},
		{ID: "uptime-kuma", Title: "Uptime Kuma", Description: "Self-hosted status page and uptime monitoring", Category: "Monitoring", Logo: "📈", Image: "louislam/uptime-kuma:latest", Ports: map[string]string{"3001/tcp": "3002"},
			Volumes: []string{"uptime_kuma_data:/app/data"}, Restart: "unless-stopped"},
		{ID: "netdata", Title: "Netdata", Description: "Real-time performance and health monitoring", Category: "Monitoring", Logo: "📉", Image: "netdata/netdata:latest", Ports: map[string]string{"19999/tcp": "19999"},
			Volumes: []string{"/proc:/host/proc:ro", "/sys:/host/sys:ro", "/var/run/docker.sock:/var/run/docker.sock:ro"}, Restart: "unless-stopped"},
		{ID: "homarr", Title: "Homarr", Description: "Sleek homelab dashboard with Docker integration", Category: "Dashboard", Logo: "🏠", Image: "ghcr.io/ajnart/homarr:latest", Ports: map[string]string{"7575/tcp": "7575"},
			Volumes: []string{"homarr_configs:/app/data/configs", "homarr_icons:/app/public/icons", "/var/run/docker.sock:/var/run/docker.sock:ro"}, Restart: "unless-stopped"},
		{ID: "heimdall", Title: "Heimdall", Description: "Application dashboard and launcher", Category: "Dashboard", Logo: "🌈", Image: "linuxserver/heimdall:latest", Ports: map[string]string{"80/tcp": "8092", "443/tcp": "8444"},
			Volumes: []string{"heimdall_config:/config"}, Restart: "unless-stopped"},
		{ID: "homepage", Title: "Homepage", Description: "Modern application dashboard with service integrations", Category: "Dashboard", Logo: "📋", Image: "ghcr.io/gethomepage/homepage:latest", Ports: map[string]string{"3000/tcp": "3004"},
			Volumes: []string{"homepage_config:/app/config"}, Restart: "unless-stopped"},
		{ID: "dashy", Title: "Dashy", Description: "Feature-rich homelab dashboard with widgets and themes", Category: "Dashboard", Logo: "🎯", Image: "lissy93/dashy:latest", Ports: map[string]string{"8080/tcp": "4000"},
			Volumes: []string{"dashy_config:/app/user-data"}, Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// CMS & Wikis
		// ═══════════════════════════════════════
		{ID: "wordpress", Title: "WordPress", Description: "World's most popular CMS", Category: "CMS", Logo: "📝", Image: "wordpress:latest", Ports: map[string]string{"80/tcp": "8083"},
			Env: []TemplateEnv{{Name: "WORDPRESS_DB_HOST", Label: "DB Host", Default: "db"}, {Name: "WORDPRESS_DB_PASSWORD", Label: "DB Password", Default: "rootpass"}}, Restart: "unless-stopped"},
		{ID: "ghost", Title: "Ghost", Description: "Modern publishing platform for blogs", Category: "CMS", Logo: "👻", Image: "ghost:latest", Ports: map[string]string{"2368/tcp": "2368"}, Restart: "unless-stopped"},
		{ID: "bookstack", Title: "BookStack", Description: "Simple wiki and documentation platform", Category: "Wiki", Logo: "📖", Image: "linuxserver/bookstack:latest", Ports: map[string]string{"80/tcp": "6875"},
			Env:     []TemplateEnv{{Name: "DB_HOST", Label: "DB Host", Default: "bookstack-db"}, {Name: "DB_USER", Label: "DB User", Default: "bookstack"}, {Name: "DB_PASS", Label: "DB Password", Default: "bookstack"}, {Name: "DB_DATABASE", Label: "DB Name", Default: "bookstackapp"}, {Name: "APP_URL", Label: "App URL", Default: "http://localhost:6875"}},
			Volumes: []string{"bookstack_data:/config"}, Restart: "unless-stopped"},
		{ID: "wikijs", Title: "Wiki.js", Description: "Modern wiki engine with powerful features", Category: "Wiki", Logo: "📚", Image: "ghcr.io/requarks/wiki:2", Ports: map[string]string{"3000/tcp": "3005"},
			Env:     []TemplateEnv{{Name: "DB_TYPE", Label: "DB Type", Default: "sqlite"}, {Name: "DB_FILEPATH", Label: "DB Path", Default: "/wiki/data/db.sqlite"}},
			Volumes: []string{"wikijs_data:/wiki/data"}, Restart: "unless-stopped"},
		{ID: "outline", Title: "Outline", Description: "Beautiful team knowledge base and wiki", Category: "Wiki", Logo: "✍️", Image: "outlinewiki/outline:latest", Ports: map[string]string{"3000/tcp": "3006"},
			Env:     []TemplateEnv{{Name: "SECRET_KEY", Label: "Secret Key", Default: "change-me-to-random-string"}, {Name: "DATABASE_URL", Label: "Database URL", Default: "postgres://outline:outline@outline-db/outline"}},
			Volumes: []string{"outline_data:/var/lib/outline/data"}, Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// Security & Authentication
		// ═══════════════════════════════════════
		{ID: "vaultwarden", Title: "Vaultwarden", Description: "Lightweight Bitwarden-compatible password manager", Category: "Security", Logo: "🔐", Image: "vaultwarden/server:latest", Ports: map[string]string{"80/tcp": "8093"},
			Env:     []TemplateEnv{{Name: "ADMIN_TOKEN", Label: "Admin Token", Default: ""}, {Name: "SIGNUPS_ALLOWED", Label: "Allow Signups", Default: "true"}},
			Volumes: []string{"vaultwarden_data:/data"}, Restart: "unless-stopped"},
		{ID: "authelia", Title: "Authelia", Description: "SSO and 2FA authentication server for web apps", Category: "Security", Logo: "🛡️", Image: "authelia/authelia:latest", Ports: map[string]string{"9091/tcp": "9091"},
			Volumes: []string{"authelia_config:/config"}, Restart: "unless-stopped"},
		{ID: "keycloak", Title: "Keycloak", Description: "Enterprise-grade identity and access management (IAM)", Category: "Security", Logo: "🔑", Image: "quay.io/keycloak/keycloak:latest", Ports: map[string]string{"8080/tcp": "8094"},
			Env:     []TemplateEnv{{Name: "KEYCLOAK_ADMIN", Label: "Admin User", Default: "admin"}, {Name: "KEYCLOAK_ADMIN_PASSWORD", Label: "Admin Password", Default: "admin"}},
			Restart: "unless-stopped", Command: "start-dev"},

		// ═══════════════════════════════════════
		// VPN & Networking
		// ═══════════════════════════════════════
		{ID: "wireguard", Title: "WireGuard", Description: "Fast, modern VPN with simple configuration", Category: "VPN", Logo: "🔒", Image: "linuxserver/wireguard:latest", Ports: map[string]string{"51820/udp": "51820"},
			Env:     []TemplateEnv{{Name: "SERVERURL", Label: "Server URL", Default: "auto"}, {Name: "PEERS", Label: "Number of Peers", Default: "3"}},
			Volumes: []string{"wireguard_config:/config"}, Restart: "unless-stopped"},
		{ID: "adguard", Title: "AdGuard Home", Description: "Network-wide ad and tracker blocking DNS", Category: "VPN", Logo: "🛡️", Image: "adguard/adguardhome:latest", Ports: map[string]string{"3000/tcp": "3007", "53/tcp": "53", "53/udp": "53"},
			Volumes: []string{"adguard_work:/opt/adguardhome/work", "adguard_conf:/opt/adguardhome/conf"}, Restart: "unless-stopped"},
		{ID: "pihole", Title: "Pi-hole", Description: "Network-wide ad blocking DNS sinkhole", Category: "VPN", Logo: "🕳️", Image: "pihole/pihole:latest", Ports: map[string]string{"80/tcp": "8095", "53/tcp": "5353", "53/udp": "5353"},
			Env:     []TemplateEnv{{Name: "WEBPASSWORD", Label: "Web Password", Default: "admin"}, {Name: "TZ", Label: "Timezone", Default: "UTC"}},
			Volumes: []string{"pihole_data:/etc/pihole", "pihole_dnsmasq:/etc/dnsmasq.d"}, Restart: "unless-stopped"},
		{ID: "tailscale", Title: "Tailscale", Description: "Zero-config mesh VPN built on WireGuard", Category: "VPN", Logo: "🌐", Image: "tailscale/tailscale:latest",
			Env:     []TemplateEnv{{Name: "TS_AUTHKEY", Label: "Auth Key", Default: ""}, {Name: "TS_HOSTNAME", Label: "Hostname", Default: "novapanel-node"}},
			Volumes: []string{"tailscale_state:/var/lib/tailscale"}, Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// Home Automation
		// ═══════════════════════════════════════
		{ID: "homeassistant", Title: "Home Assistant", Description: "Open-source home automation with 2000+ integrations", Category: "Home Automation", Logo: "🏡", Image: "ghcr.io/home-assistant/home-assistant:stable", Ports: map[string]string{"8123/tcp": "8123"},
			Env:     []TemplateEnv{{Name: "TZ", Label: "Timezone", Default: "UTC"}},
			Volumes: []string{"homeassistant_config:/config"}, Restart: "unless-stopped"},
		{ID: "nodered", Title: "Node-RED", Description: "Low-code programming for IoT and automation", Category: "Home Automation", Logo: "🔴", Image: "nodered/node-red:latest", Ports: map[string]string{"1880/tcp": "1880"},
			Volumes: []string{"nodered_data:/data"}, Restart: "unless-stopped"},
		{ID: "mosquitto", Title: "Eclipse Mosquitto", Description: "Lightweight MQTT message broker for IoT", Category: "Home Automation", Logo: "🦟", Image: "eclipse-mosquitto:latest", Ports: map[string]string{"1883/tcp": "1883", "9001/tcp": "9003"},
			Volumes: []string{"mosquitto_data:/mosquitto/data", "mosquitto_log:/mosquitto/log"}, Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// Productivity & Office
		// ═══════════════════════════════════════
		{ID: "paperless-ngx", Title: "Paperless-ngx", Description: "Document management system with OCR and tagging", Category: "Productivity", Logo: "📄", Image: "ghcr.io/paperless-ngx/paperless-ngx:latest", Ports: map[string]string{"8000/tcp": "8096"},
			Env:     []TemplateEnv{{Name: "PAPERLESS_SECRET_KEY", Label: "Secret Key", Default: "change-me-to-random"}, {Name: "PAPERLESS_ADMIN_USER", Label: "Admin User", Default: "admin"}, {Name: "PAPERLESS_ADMIN_PASSWORD", Label: "Admin Password", Default: "admin"}},
			Volumes: []string{"paperless_data:/usr/src/paperless/data", "paperless_media:/usr/src/paperless/media"}, Restart: "unless-stopped"},
		{ID: "stirlingpdf", Title: "Stirling PDF", Description: "Self-hosted PDF manipulation tool", Category: "Productivity", Logo: "📑", Image: "frooodle/s-pdf:latest", Ports: map[string]string{"8080/tcp": "8098"},
			Restart: "unless-stopped"},
		{ID: "excalidraw", Title: "Excalidraw", Description: "Collaborative whiteboarding and diagramming", Category: "Productivity", Logo: "🎨", Image: "excalidraw/excalidraw:latest", Ports: map[string]string{"80/tcp": "8099"},
			Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// ERP & Business
		// ═══════════════════════════════════════
		{ID: "odoo", Title: "Odoo", Description: "All-in-one ERP: CRM, accounting, inventory, HR, and more", Category: "ERP", Logo: "🏢", Image: "odoo:latest", Ports: map[string]string{"8069/tcp": "8069", "8072/tcp": "8072"},
			Env:     []TemplateEnv{{Name: "HOST", Label: "PostgreSQL Host", Default: "odoo-db"}, {Name: "USER", Label: "DB User", Default: "odoo"}, {Name: "PASSWORD", Label: "DB Password", Default: "odoo"}},
			Volumes: []string{"odoo_web_data:/var/lib/odoo", "odoo_config:/etc/odoo"}, Restart: "unless-stopped"},
		{ID: "erpnext", Title: "ERPNext", Description: "Free open-source ERP with manufacturing, accounting, CRM", Category: "ERP", Logo: "📊", Image: "frappe/erpnext:latest", Ports: map[string]string{"8080/tcp": "8106"},
			Env:     []TemplateEnv{{Name: "FRAPPE_SITE_NAME_HEADER", Label: "Site Name", Default: "localhost"}, {Name: "DB_HOST", Label: "DB Host", Default: "erpnext-db"}, {Name: "DB_PORT", Label: "DB Port", Default: "3306"}, {Name: "REDIS_CACHE", Label: "Redis Cache", Default: "redis://erpnext-redis:6379/0"}},
			Volumes: []string{"erpnext_sites:/home/frappe/frappe-bench/sites"}, Restart: "unless-stopped"},
		{ID: "dolibarr", Title: "Dolibarr", Description: "Modular ERP and CRM for small businesses", Category: "ERP", Logo: "💼", Image: "dolibarr/dolibarr:latest", Ports: map[string]string{"80/tcp": "8107"},
			Env:     []TemplateEnv{{Name: "DOLI_DB_HOST", Label: "DB Host", Default: "dolibarr-db"}, {Name: "DOLI_DB_USER", Label: "DB User", Default: "dolibarr"}, {Name: "DOLI_DB_PASSWORD", Label: "DB Password", Default: "dolibarr"}, {Name: "DOLI_DB_NAME", Label: "DB Name", Default: "dolibarr"}, {Name: "DOLI_ADMIN_LOGIN", Label: "Admin Login", Default: "admin"}, {Name: "DOLI_ADMIN_PASSWORD", Label: "Admin Password", Default: "admin"}},
			Volumes: []string{"dolibarr_docs:/var/www/documents", "dolibarr_html:/var/www/html/custom"}, Restart: "unless-stopped"},
		{ID: "invoice-ninja", Title: "Invoice Ninja", Description: "Professional invoicing, billing, and payment platform", Category: "ERP", Logo: "💰", Image: "invoiceninja/invoiceninja:latest", Ports: map[string]string{"80/tcp": "8108"},
			Env:     []TemplateEnv{{Name: "APP_URL", Label: "App URL", Default: "http://localhost:8108"}, {Name: "APP_KEY", Label: "App Key", Default: "base64:changeMe="}, {Name: "DB_HOST", Label: "DB Host", Default: "ninja-db"}, {Name: "DB_DATABASE", Label: "DB Name", Default: "ninja"}, {Name: "DB_USERNAME", Label: "DB User", Default: "ninja"}, {Name: "DB_PASSWORD", Label: "DB Password", Default: "ninja"}},
			Volumes: []string{"invoiceninja_public:/var/www/app/public"}, Restart: "unless-stopped"},
		{ID: "akaunting", Title: "Akaunting", Description: "Free online accounting software for small business", Category: "ERP", Logo: "🧾", Image: "akaunting/akaunting:latest", Ports: map[string]string{"80/tcp": "8109"},
			Env:     []TemplateEnv{{Name: "APP_URL", Label: "App URL", Default: "http://localhost:8109"}, {Name: "DB_HOST", Label: "DB Host", Default: "akaunting-db"}, {Name: "DB_DATABASE", Label: "DB Name", Default: "akaunting"}, {Name: "DB_USERNAME", Label: "DB User", Default: "akaunting"}, {Name: "DB_PASSWORD", Label: "DB Password", Default: "akaunting"}},
			Volumes: []string{"akaunting_data:/var/www/html/storage"}, Restart: "unless-stopped"},
		{ID: "crateinvoice", Title: "Crater", Description: "Open-source invoicing and expense tracking", Category: "ERP", Logo: "📋", Image: "bytefury/crater:latest", Ports: map[string]string{"80/tcp": "8110"},
			Env:     []TemplateEnv{{Name: "APP_URL", Label: "App URL", Default: "http://localhost:8110"}, {Name: "DB_HOST", Label: "DB Host", Default: "crater-db"}},
			Volumes: []string{"crater_data:/var/www/html/storage"}, Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// DevOps & Development
		// ═══════════════════════════════════════
		{ID: "gitea", Title: "Gitea", Description: "Lightweight self-hosted Git service", Category: "DevOps", Logo: "🍵", Image: "gitea/gitea:latest", Ports: map[string]string{"3000/tcp": "3003", "22/tcp": "2222"},
			Volumes: []string{"gitea_data:/data"}, Restart: "unless-stopped"},
		{ID: "gitlab", Title: "GitLab CE", Description: "Complete DevOps platform with Git, CI/CD, and more", Category: "DevOps", Logo: "🦊", Image: "gitlab/gitlab-ce:latest", Ports: map[string]string{"80/tcp": "8100", "443/tcp": "8445", "22/tcp": "2223"},
			Env:     []TemplateEnv{{Name: "GITLAB_ROOT_PASSWORD", Label: "Root Password", Default: "gitlabadmin"}},
			Volumes: []string{"gitlab_config:/etc/gitlab", "gitlab_logs:/var/log/gitlab", "gitlab_data:/var/opt/gitlab"}, Restart: "unless-stopped"},
		{ID: "drone", Title: "Drone CI", Description: "Container-native CI/CD platform", Category: "DevOps", Logo: "🤖", Image: "drone/drone:latest", Ports: map[string]string{"80/tcp": "8084"},
			Env: []TemplateEnv{{Name: "DRONE_SERVER_HOST", Label: "Server Host", Default: "localhost"}}, Restart: "unless-stopped"},
		{ID: "registry", Title: "Docker Registry", Description: "Private container image registry", Category: "DevOps", Logo: "📦", Image: "registry:2", Ports: map[string]string{"5000/tcp": "5000"},
			Volumes: []string{"registry_data:/var/lib/registry"}, Restart: "unless-stopped"},
		{ID: "portainer", Title: "Portainer CE", Description: "Docker and Kubernetes management GUI", Category: "DevOps", Logo: "🐳", Image: "portainer/portainer-ce:latest", Ports: map[string]string{"9443/tcp": "9443", "8000/tcp": "8001"},
			Volumes: []string{"/var/run/docker.sock:/var/run/docker.sock", "portainer_data:/data"}, Restart: "unless-stopped"},
		{ID: "codeserver", Title: "Code Server", Description: "VS Code in the browser", Category: "DevOps", Logo: "💻", Image: "linuxserver/code-server:latest", Ports: map[string]string{"8443/tcp": "8446"},
			Env:     []TemplateEnv{{Name: "PASSWORD", Label: "Access Password", Default: "codeserver"}},
			Volumes: []string{"codeserver_config:/config"}, Restart: "unless-stopped"},
		{ID: "jenkins", Title: "Jenkins", Description: "Leading open-source CI/CD automation server", Category: "DevOps", Logo: "🏗️", Image: "jenkins/jenkins:lts", Ports: map[string]string{"8080/tcp": "8101", "50000/tcp": "50000"},
			Volumes: []string{"jenkins_data:/var/jenkins_home"}, Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// Communication
		// ═══════════════════════════════════════
		{ID: "matrix-synapse", Title: "Matrix Synapse", Description: "Decentralized end-to-end encrypted messaging server", Category: "Communication", Logo: "💬", Image: "matrixdotorg/synapse:latest", Ports: map[string]string{"8008/tcp": "8008"},
			Env:     []TemplateEnv{{Name: "SYNAPSE_SERVER_NAME", Label: "Server Name", Default: "localhost"}, {Name: "SYNAPSE_REPORT_STATS", Label: "Report Stats", Default: "no"}},
			Volumes: []string{"synapse_data:/data"}, Restart: "unless-stopped"},
		{ID: "rocket-chat", Title: "Rocket.Chat", Description: "Open-source Slack alternative for team communication", Category: "Communication", Logo: "🚀", Image: "rocket.chat:latest", Ports: map[string]string{"3000/tcp": "3008"},
			Env:     []TemplateEnv{{Name: "MONGO_URL", Label: "MongoDB URL", Default: "mongodb://rocketchat-mongo:27017/rocketchat"}},
			Volumes: []string{"rocketchat_uploads:/app/uploads"}, Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// Automation & BI
		// ═══════════════════════════════════════
		{ID: "n8n", Title: "n8n", Description: "Workflow automation tool (Zapier alternative)", Category: "Automation", Logo: "⚡", Image: "n8nio/n8n:latest", Ports: map[string]string{"5678/tcp": "5678"},
			Env:     []TemplateEnv{{Name: "N8N_BASIC_AUTH_USER", Label: "Auth User", Default: "admin"}, {Name: "N8N_BASIC_AUTH_PASSWORD", Label: "Auth Password", Default: "admin"}},
			Volumes: []string{"n8n_data:/home/node/.n8n"}, Restart: "unless-stopped"},
		{ID: "nocodb", Title: "NocoDB", Description: "Open-source Airtable alternative (spreadsheet to DB)", Category: "Automation", Logo: "📊", Image: "nocodb/nocodb:latest", Ports: map[string]string{"8080/tcp": "8102"},
			Volumes: []string{"nocodb_data:/usr/app/data"}, Restart: "unless-stopped"},
		{ID: "appsmith", Title: "Appsmith", Description: "Low-code platform for building internal tools", Category: "Automation", Logo: "🛠️", Image: "appsmith/appsmith-ce:latest", Ports: map[string]string{"80/tcp": "8103"},
			Volumes: []string{"appsmith_data:/appsmith-stacks"}, Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// Backup
		// ═══════════════════════════════════════
		{ID: "duplicati", Title: "Duplicati", Description: "Free backup software with cloud storage support", Category: "Backup", Logo: "💿", Image: "linuxserver/duplicati:latest", Ports: map[string]string{"8200/tcp": "8200"},
			Volumes: []string{"duplicati_config:/config"}, Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// Download & Torrent
		// ═══════════════════════════════════════
		{ID: "qbittorrent", Title: "qBittorrent", Description: "Lightweight BitTorrent client with web UI", Category: "Download", Logo: "📥", Image: "linuxserver/qbittorrent:latest", Ports: map[string]string{"8080/tcp": "8104", "6881/tcp": "6881", "6881/udp": "6881"},
			Volumes: []string{"qbittorrent_config:/config"}, Restart: "unless-stopped"},
		{ID: "transmission", Title: "Transmission", Description: "Simple and efficient BitTorrent client", Category: "Download", Logo: "⬇️", Image: "linuxserver/transmission:latest", Ports: map[string]string{"9091/tcp": "9092", "51413/tcp": "51413"},
			Volumes: []string{"transmission_config:/config"}, Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// Database Management Tools
		// ═══════════════════════════════════════
		{ID: "adminer", Title: "Adminer", Description: "Lightweight database management UI", Category: "Tools", Logo: "🗄️", Image: "adminer:latest", Ports: map[string]string{"8080/tcp": "8085"}, Restart: "unless-stopped"},
		{ID: "phpmyadmin", Title: "phpMyAdmin", Description: "MySQL/MariaDB web administration", Category: "Tools", Logo: "🔧", Image: "phpmyadmin:latest", Ports: map[string]string{"80/tcp": "8086"},
			Env: []TemplateEnv{{Name: "PMA_HOST", Label: "MySQL Host", Default: "mysql"}}, Restart: "unless-stopped"},
		{ID: "pgadmin", Title: "pgAdmin 4", Description: "PostgreSQL web-based admin tool", Category: "Tools", Logo: "🐘", Image: "dpage/pgadmin4:latest", Ports: map[string]string{"80/tcp": "5050"},
			Env:     []TemplateEnv{{Name: "PGADMIN_DEFAULT_EMAIL", Label: "Admin Email", Default: "admin@admin.com"}, {Name: "PGADMIN_DEFAULT_PASSWORD", Label: "Admin Password", Default: "admin"}},
			Volumes: []string{"pgadmin_data:/var/lib/pgadmin"}, Restart: "unless-stopped"},
		{ID: "watchtower", Title: "Watchtower", Description: "Automatic Docker container updates", Category: "Tools", Logo: "🗼", Image: "containrrr/watchtower:latest",
			Volumes: []string{"/var/run/docker.sock:/var/run/docker.sock"}, Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// Analytics
		// ═══════════════════════════════════════
		{ID: "plausible", Title: "Plausible Analytics", Description: "Privacy-friendly Google Analytics alternative", Category: "Analytics", Logo: "📊", Image: "plausible/analytics:latest", Ports: map[string]string{"8000/tcp": "8105"},
			Env:     []TemplateEnv{{Name: "BASE_URL", Label: "Base URL", Default: "http://localhost:8105"}, {Name: "SECRET_KEY_BASE", Label: "Secret Key", Default: "change-me-to-random-64-chars"}},
			Restart: "unless-stopped"},
		{ID: "umami", Title: "Umami", Description: "Simple, fast, privacy-focused web analytics", Category: "Analytics", Logo: "📈", Image: "ghcr.io/umami-software/umami:postgresql-latest", Ports: map[string]string{"3000/tcp": "3009"},
			Env:     []TemplateEnv{{Name: "DATABASE_URL", Label: "Database URL", Default: "postgresql://umami:umami@umami-db:5432/umami"}},
			Restart: "unless-stopped"},

		// ═══════════════════════════════════════
		// Email
		// ═══════════════════════════════════════
		{ID: "mailhog", Title: "MailHog", Description: "Email testing tool for developers", Category: "Email", Logo: "📧", Image: "mailhog/mailhog:latest", Ports: map[string]string{"1025/tcp": "1025", "8025/tcp": "8025"}, Restart: "unless-stopped"},
	}
}
