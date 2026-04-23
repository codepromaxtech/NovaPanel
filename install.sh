#!/usr/bin/env bash
set -euo pipefail

REPO="https://raw.githubusercontent.com/codepromaxtech/novapanel/main"
INSTALL_DIR="novapanel"
PORT="8080"

# ── Colours ────────────────────────────────────────────────────────────────
BOLD="\033[1m"
DIM="\033[2m"
CYAN="\033[36m"
GREEN="\033[32m"
YELLOW="\033[33m"
RED="\033[31m"
RESET="\033[0m"

print_banner() {
  echo ""
  echo -e "${CYAN}${BOLD}"
  echo "  ███╗   ██╗ ██████╗ ██╗   ██╗ █████╗ ██████╗  █████╗ ███╗   ██╗███████╗██╗"
  echo "  ████╗  ██║██╔═══██╗██║   ██║██╔══██╗██╔══██╗██╔══██╗████╗  ██║██╔════╝██║"
  echo "  ██╔██╗ ██║██║   ██║██║   ██║███████║██████╔╝███████║██╔██╗ ██║█████╗  ██║"
  echo "  ██║╚██╗██║██║   ██║╚██╗ ██╔╝██╔══██║██╔═══╝ ██╔══██║██║╚██╗██║██╔══╝  ██║"
  echo "  ██║ ╚████║╚██████╔╝ ╚████╔╝ ██║  ██║██║     ██║  ██║██║ ╚████║███████╗███████╗"
  echo "  ╚═╝  ╚═══╝ ╚═════╝   ╚═══╝  ╚═╝  ╚═╝╚═╝     ╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝╚══════╝"
  echo -e "${RESET}"
  echo -e "  ${DIM}Server Control Panel — https://github.com/codepromaxtech/novapanel${RESET}"
  echo ""
}

step() { echo -e "  ${CYAN}▸${RESET} ${BOLD}$1${RESET}"; }
ok()   { echo -e "  ${GREEN}✔${RESET} $1"; }
warn() { echo -e "  ${YELLOW}⚠${RESET}  $1"; }
die()  { echo -e "\n  ${RED}✖${RESET}  ${BOLD}$1${RESET}\n"; exit 1; }
hr()   { echo -e "  ${DIM}────────────────────────────────────────────${RESET}"; }

# ── Banner ──────────────────────────────────────────────────────────────────
print_banner

hr
echo -e "  ${BOLD}Checking requirements...${RESET}"
hr

# Docker
step "Docker"
command -v docker >/dev/null 2>&1 || die "Docker not found.\n     Install it first: https://docs.docker.com/get-docker/"
DOCKER_VER=$(docker version --format '{{.Server.Version}}' 2>/dev/null || echo "unknown")
ok "Docker $DOCKER_VER"

# Docker Compose
step "Docker Compose"
docker compose version >/dev/null 2>&1 || die "Docker Compose plugin not found.\n     Run: apt install docker-compose-plugin"
COMPOSE_VER=$(docker compose version --short 2>/dev/null || echo "unknown")
ok "Docker Compose $COMPOSE_VER"

# curl
step "curl"
command -v curl >/dev/null 2>&1 || die "curl not found. Run: apt install curl"
ok "curl $(curl --version | head -1 | awk '{print $2}')"

echo ""
hr
echo -e "  ${BOLD}Downloading NovaPanel...${RESET}"
hr

# Create install directory
mkdir -p "$INSTALL_DIR" && cd "$INSTALL_DIR"

# docker-compose.yml
step "docker-compose.yml"
curl -fsSL "$REPO/docker-compose.hub.yml" -o docker-compose.yml
ok "docker-compose.yml"

# .env — only create if it doesn't exist
step ".env configuration"
if [ -f .env ]; then
  warn ".env already exists — skipping (delete it to reset)"
else
  curl -fsSL "$REPO/.env.example" -o .env

  # Auto-generate secrets so the user just needs to set passwords
  JWT=$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32)
  ENC=$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | base64 | tr -dc 'a-f0-9' | head -c 64)
  DB_PASS=$(openssl rand -base64 18 | tr -dc 'a-zA-Z0-9' | head -c 20)
  REDIS_PASS=$(openssl rand -base64 18 | tr -dc 'a-zA-Z0-9' | head -c 20)

  sed -i "s|change_me_jwt_secret_at_least_32_chars|$JWT|g"              .env
  sed -i "s|change_me_64_char_hex_key_here_0000000000000000000000000000000000|${ENC}${ENC}|g" .env
  sed -i "s|change_me_strong_password|$DB_PASS|g"                       .env
  sed -i "s|change_me_redis_password|$REDIS_PASS|g"                     .env

  ok ".env created with auto-generated secrets"
fi

# Detect public IP
step "Detecting server IP"
SERVER_IP=$(curl -fsSL --max-time 5 ifconfig.me 2>/dev/null \
         || curl -fsSL --max-time 5 icanhazip.com 2>/dev/null \
         || hostname -I | awk '{print $1}')
sed -i "s|http://your-server-ip:$PORT|http://$SERVER_IP:$PORT|g" .env 2>/dev/null || true
ok "Server IP: $SERVER_IP"

echo ""
hr
echo ""
echo -e "  ${GREEN}${BOLD}NovaPanel is ready to launch!${RESET}"
echo ""
echo -e "  ${BOLD}Before you start, open .env and set:${RESET}"
echo ""
echo -e "    ${YELLOW}DB_PASSWORD${RESET}      — PostgreSQL password           ${DIM}(auto-generated ✔)${RESET}"
echo -e "    ${YELLOW}REDIS_PASSWORD${RESET}   — Redis password                ${DIM}(auto-generated ✔)${RESET}"
echo -e "    ${YELLOW}JWT_SECRET${RESET}       — JWT signing key               ${DIM}(auto-generated ✔)${RESET}"
echo -e "    ${YELLOW}ENCRYPTION_KEY${RESET}   — SSH/env var encryption key    ${DIM}(auto-generated ✔)${RESET}"
echo -e "    ${YELLOW}SMTP_*${RESET}           — Email (for password reset)    ${DIM}(optional)${RESET}"
echo -e "    ${YELLOW}STRIPE_*${RESET}         — Billing integration           ${DIM}(optional)${RESET}"
echo ""
echo -e "  ${DIM}Edit with:  nano $(pwd)/.env${RESET}"
echo ""
hr
echo ""
echo -e "  ${BOLD}Start NovaPanel:${RESET}"
echo ""
echo -e "    ${CYAN}cd $INSTALL_DIR${RESET}"
echo -e "    ${CYAN}docker compose up -d${RESET}"
echo ""
echo -e "  ${BOLD}Then open:${RESET}"
echo ""
echo -e "    ${GREEN}${BOLD}http://$SERVER_IP:$PORT${RESET}"
echo ""
echo -e "  ${DIM}The first user you register becomes the administrator.${RESET}"
echo ""
hr
echo ""
