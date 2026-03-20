# NovaPanel

A modern, full-featured server management panel built with **Go** (backend) and **React + TypeScript** (frontend).

## Features

| Module | Capabilities |
|--------|--------------|
| **Servers** | Add/manage servers, SSH-based auto-provisioning with 12 module install scripts |
| **Domains** | Domain management with DNS configuration |
| **Databases** | MySQL, PostgreSQL, MongoDB, Redis — query runner, web tools (phpMyAdmin, Adminer, Mongo Express, Redis Commander), user management, export/import |
| **Email** | Full email server with Postfix/Dovecot, webmail (Roundcube), DKIM/SPF/DMARC |
| **File Manager** | Browse, edit, upload, download files on remote servers |
| **File Transfers** | Rsync-based transfers with scheduling, bandwidth limits, dry-run, exclude patterns |
| **Backups** | Database, site, and full server backups with restore. Supports MySQL, PostgreSQL, MongoDB |
| **Cron Jobs** | View, add, edit, delete cron entries on any server |
| **System Services** | Manage systemd services — start, stop, restart, enable, disable, view logs |
| **Docker** | Container management — run, stop, remove, view logs, images, networks, volumes |
| **Kubernetes** | Cluster management — pods, deployments, services, namespaces, cron jobs |
| **Monitoring** | Server metrics, resource usage, alerts |
| **Security** | Firewall rules (UFW/iptables), fail2ban, SSH hardening |
| **WAF** | ModSecurity + OWASP CRS for Nginx/Apache |
| **Deployments** | Git-based deployments with rollback |
| **Billing** | Usage tracking and billing |

## Tech Stack

- **Backend:** Go, Gin, PostgreSQL, Redis, JWT auth, SSH provisioner
- **Frontend:** React 18, TypeScript, Vite, Lucide icons
- **Infrastructure:** Docker, Docker Compose, Nginx reverse proxy

## Quick Start

```bash
# Clone
git clone git@github.com:codepromaxtech/NovaPanel.git
cd NovaPanel

# Configure
cp .env.example .env
# Edit .env with your database and Redis credentials

# Start with Docker
docker compose up -d

# Or develop locally
cd backend && go run cmd/api/main.go
cd frontend && npm install && npm run dev
```

## Project Structure

```
NovaPanel/
├── backend/
│   ├── cmd/api/main.go          # Entry point
│   ├── internal/
│   │   ├── handlers/            # HTTP handlers (API endpoints)
│   │   ├── services/            # Business logic
│   │   ├── models/              # Data models & DTOs
│   │   ├── middleware/           # Auth, CORS, rate limiting
│   │   └── provisioner/         # SSH provisioner & install scripts
│   └── migrations/              # PostgreSQL migrations
├── frontend/
│   └── src/
│       ├── pages/               # Page components
│       ├── services/            # API service layer
│       ├── components/          # Reusable UI components
│       └── store/               # State management
├── docker-compose.yml
└── Dockerfile
```

## License

MIT
