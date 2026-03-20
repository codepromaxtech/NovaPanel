# NovaPanel

A modern, full-featured server management panel built with **Go** (backend) and **React + TypeScript** (frontend). 
It features a powerful **agentless SSH provisioner**, allowing you to manage unlimited remote servers without installing custom daemons.

## ✨ Core Features

| Feature | Capabilities |
|---------|--------------|
| **Server Management** | Add/manage servers via SSH (Key or Password). Auto-provisions environments, tracks live CPU/RAM/Disk metrics, and features a built-in web terminal. |
| **Cloudflare Integration** | Full Cloudflare API v4 UI. Manage Zones, DNS, SSL/TLS, Caching, Security Settings, and **Cloudflare Tunnels**. Includes 1-click `cloudflared` installation and systemd deployment on remote servers. |
| **IDE-like File Manager** | Browse, edit, upload, download, compress, and extract files on remote servers. Includes a multi-tab syntax-highlighted code editor, search/grep, and permission management. |
| **Domains & Web Servers** | Manage domains, configure Nginx/Apache, handle DNS records, and SSL certificates. |
| **Databases** | Deploy MySQL, PostgreSQL, MongoDB, and Redis. Features built-in query runners, DB sizes, and web tools (phpMyAdmin, Adminer, Mongo Express). |
| **Backups & Transfers** | Rsync-based server-to-server file transfers. Full Database, Site, and Server backups with restore capabilities. |
| **System Control** | Manage `systemd` services (start/stop/restart/enable/disable/logs) and view active Cron jobs directly from the panel. |
| **Docker & Kubernetes** | Full container orchestration. Manage Docker containers, images, volumes, and networks. K8s cluster management (Pods, Deployments, Services, Namespaces). |
| **Email Server** | Full email server management with Postfix/Dovecot, Roundcube webmail, DKIM/SPF/DMARC configuration, aliases, and autoresponders. |
| **Security & WAF** | Manage Firewall rules (UFW/iptables), Fail2Ban, and SSH hardening. ModSecurity + OWASP CRS integration for Nginx/Apache. |
| **Deployments** | Git-based CI/CD deployments with build logs and rollback support. |

## 🛠 Tech Stack

- **Backend:** Go, Gin framework, PostgreSQL, Redis, custom SSH provisioner (`golang.org/x/crypto/ssh`)
- **Frontend:** React 18, TypeScript, Vite, Tailwind CSS, Lucide icons
- **Infrastructure:** Docker, Docker Compose, Nginx reverse proxy

## 🚀 Quick Start

```bash
# Clone the repository
git clone git@github.com:codepromaxtech/NovaPanel.git
cd NovaPanel

# Configure environment
cp .env.example .env
# Edit .env with your PostgreSQL and Redis credentials

# Start with Docker Compose
docker compose up -d

# Or develop locally (requires Postgres & Redis running)
cd backend && go run cmd/api/main.go
cd frontend && npm install && npm run dev
```

## 📂 Project Structure

```
NovaPanel/
├── backend/
│   ├── cmd/api/main.go          # Main entry point & API Router
│   ├── internal/
│   │   ├── handlers/            # HTTP endpoint handlers
│   │   ├── services/            # Core business logic
│   │   ├── models/              # Database models & JSON DTOs
│   │   ├── middleware/          # Auth, CORS, rate limiting
│   │   └── provisioner/         # SSH command execution & install scripts
│   └── migrations/              # PostgreSQL schema migrations
├── frontend/
│   └── src/
│       ├── pages/               # React page components
│       ├── services/            # Axios API wrappers
│       ├── components/          # Reusable UI elements & layouts
│       └── store/               # Zustand state management
├── docker-compose.yml           # Production deployment stack
└── Dockerfile                   # Multi-stage Docker build
```

## 📄 License

MIT
