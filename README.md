# NovaPanel

<p align="center">
  <img src="https://img.shields.io/docker/v/codepromax24/novapanel?label=Docker%20Hub&logo=docker&color=0db7ed" />
  <img src="https://img.shields.io/docker/pulls/codepromax24/novapanel?color=0db7ed&logo=docker" />
  <img src="https://img.shields.io/github/actions/workflow/status/codepromaxtech/novapanel/docker-publish.yml?label=CI&logo=github" />
  <img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go" />
  <img src="https://img.shields.io/badge/React-18-61DAFB?logo=react" />
  <img src="https://img.shields.io/badge/license-MIT-green" />
</p>

A modern, full-featured **server control panel** built with Go and React. Manage unlimited remote servers agentlessly over SSH — no daemons, no agents, no vendor lock-in.

---

## One-line install

```bash
curl -fsSL https://raw.githubusercontent.com/codepromaxtech/novapanel/main/install.sh | bash
```

This downloads `docker-compose.yml` and `.env` into a `novapanel/` folder. Edit `.env`, then:

```bash
cd novapanel
docker compose up -d
```

Open `http://<your-server-ip>:8080` — the first registered user becomes admin.

---

## Features

| Area | What you can do |
|---|---|
| **Servers** | Add servers via SSH key or password. Live CPU/RAM/disk metrics, web terminal, module installer (Docker, MySQL, Redis, Nginx, K8s, mail, DNS…) |
| **Domains & SSL** | Nginx/Apache virtual hosts, Let's Encrypt, wildcard SSL via Cloudflare DNS challenge, 1-click HTTPS |
| **Databases** | MySQL, PostgreSQL, MongoDB, Redis — create/drop/query, built-in web UIs (phpMyAdmin, Adminer, Mongo Express) |
| **Email** | Postfix/Dovecot setup, accounts, forwarders, aliases, DKIM/SPF/DMARC records, Roundcube webmail |
| **File Manager** | Browser-based file browser with multi-tab code editor, upload/download, grep search, chmod |
| **Deployments** | Git-push CI/CD (GitHub & GitLab webhooks), build logs, rollback, multi-server fan-out |
| **Docker** | Container/image/volume/network management via secure Docker socket proxy |
| **Kubernetes** | Pod, Deployment, Service, Namespace management |
| **Cloudflare** | Full API v4 UI — zones, DNS, SSL/TLS, caching, security, Tunnels, 1-click `cloudflared` |
| **Backups** | Scheduled rsync/tar backups with restore, server-to-server transfers |
| **WAF** | ModSecurity + OWASP CRS for Nginx/Apache, rule management, IP whitelist |
| **Firewall** | UFW rule management, Fail2Ban, SSH hardening |
| **Billing** | Stripe-powered plans (Community / Enterprise / Reseller), invoice history |
| **Team** | Role-based access control, invite members, per-resource permissions |
| **API Keys** | Scoped API keys with `np_` prefix, SHA-256 hashed, shown raw once at creation |
| **2FA** | TOTP (Google Authenticator / Authy), backup codes, post-login verification flow |
| **Alerts** | Metric threshold rules, email + webhook notifications, incident tracking |
| **FTP/SFTP** | Provisioned vsftpd accounts per server, quota management |
| **Reseller** | Sub-account allocation with per-client quotas |
| **Sessions** | View and revoke active login sessions per device/IP |

---

## Auto-discovery

When installed on a server with existing services, NovaPanel automatically detects:

- **Docker containers** — via Docker socket proxy (all running containers visible immediately)
- **Websites** — Nginx and Apache virtual host configs parsed from host mounts
- **Native services** — TCP port probing for 40+ service types (MySQL :3306, Redis :6379, Postgres :5432, Elasticsearch :9200, etc.)
- **Systemd units** — unit files read from host mounts; live state queried via D-Bus socket

No agents required on managed servers.

---

## Architecture

```
internet ──→ :8080 ──→ api (Go + React)
                         │
              ┌──────────┼──────────┐
              │          │          │
           postgres    redis    docker-proxy
           (db-net)  (db-net)  (socket-net)
                                    │
                         /var/run/docker.sock (host, read-only proxy)
```

**Networks are fully isolated** — Postgres and Redis are never reachable from the internet. Only port 8080 is published.

```
NovaPanel/
├── backend/
│   ├── cmd/api/main.go          # Entry point, router, auto-discovery
│   ├── internal/
│   │   ├── handlers/            # HTTP handlers
│   │   ├── services/            # Business logic, SSH provisioning
│   │   ├── models/              # DB models + DTOs
│   │   ├── middleware/          # JWT auth, quota gates, API key auth
│   │   └── provisioner/         # SSH setup scripts
│   └── migrations/              # PostgreSQL migrations (001–041+)
├── frontend/
│   └── src/
│       ├── pages/               # React pages (one per feature)
│       ├── services/            # Typed API clients
│       └── components/          # Layout, UI primitives, ToastProvider
├── automation/                  # Python FastAPI — certbot, nginx config templating
├── docker-compose.yml           # Dev stack (builds from source)
├── docker-compose.hub.yml       # Production stack (pulls from Docker Hub)
├── backend/Dockerfile           # 3-stage build: Node → Go → Alpine
└── install.sh                   # One-line installer
```

---

## Tech stack

| Layer | Technology |
|---|---|
| Backend | Go 1.25, Gin, pgx v5, Redis |
| Frontend | React 18, TypeScript, Vite 8, Tailwind CSS v4, Lucide icons |
| Database | PostgreSQL 16 |
| Cache / pubsub | Redis 7 |
| SSH | `golang.org/x/crypto/ssh` (agentless, no daemon) |
| Auth | JWT (JTI-based revocation), TOTP, bcrypt |
| Payments | Stripe (checkout sessions + webhooks) |
| Containers | Docker SDK, Kubernetes client-go |
| CI/CD | GitHub Actions → Docker Hub (`linux/amd64` + `linux/arm64`) |

---

## Docker Hub

```
codepromax24/novapanel-api:latest          # Go API + compiled React SPA
codepromax24/novapanel-automation:latest   # Python certbot/nginx automation
```

Images are built and pushed automatically on every `v*.*.*` git tag via [.github/workflows/docker-publish.yml](.github/workflows/docker-publish.yml).

---

## Local development

```bash
# Prerequisites: Go 1.25+, Node 20+, PostgreSQL 16, Redis 7

# Backend
cd backend
cp ../.env.example ../.env   # edit DB_ and REDIS_ vars
go run ./cmd/api

# Frontend (separate terminal)
cd frontend
npm install
npm run dev                  # Vite dev server on :5173 with HMR
```

---

## Configuration

Copy `.env.example` to `.env` and fill in the required values:

| Variable | Required | Description |
|---|---|---|
| `DB_PASSWORD` | Yes | PostgreSQL password |
| `REDIS_PASSWORD` | Yes | Redis password |
| `JWT_SECRET` | Yes | JWT signing key (32+ chars) |
| `ENCRYPTION_KEY` | Yes | 64-char hex key for SSH password + env var encryption |
| `PANEL_URL` | Yes | Public URL of the panel (for CORS + password reset emails) |
| `SMTP_*` | Recommended | Password reset and alert emails |
| `STRIPE_SECRET_KEY` | Optional | Stripe billing integration |
| `STRIPE_WEBHOOK_SECRET` | Optional | Stripe webhook validation |

---

## License

MIT
