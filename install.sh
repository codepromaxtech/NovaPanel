#!/usr/bin/env bash
set -euo pipefail

REPO="https://raw.githubusercontent.com/codepromaxtech/novapanel/main"
INSTALL_DIR="novapanel"
PORT="8080"

# ── Colours ─────────────────────────────────────────────────────────────────
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

step()    { echo -e "  ${CYAN}▸${RESET} ${BOLD}$1${RESET}"; }
ok()      { echo -e "  ${GREEN}✔${RESET} $1"; }
warn()    { echo -e "  ${YELLOW}⚠${RESET}  $1"; }
skipped() { echo -e "  ${DIM}–  $1 (skipped)${RESET}"; }
die()     { echo -e "\n  ${RED}✖${RESET}  ${BOLD}$1${RESET}\n"; exit 1; }
hr()      { echo -e "  ${DIM}────────────────────────────────────────────${RESET}"; }

# Generate a random secret
rand_hex()    { openssl rand -hex "${1:-32}" 2>/dev/null || cat /dev/urandom | tr -dc 'a-f0-9' | head -c $(( ${1:-32} * 2 )); }
rand_pass()   { openssl rand -base64 24 2>/dev/null | tr -dc 'a-zA-Z0-9' | head -c 24; }

# Prompt helper: ask with a default, hide input if secret=1
# Usage: ask "Label" "default" [secret]
ask() {
  local label="$1" default="$2" secret="${3:-0}" value=""
  if [ "$secret" = "1" ]; then
    echo -ne "    ${YELLOW}${label}${RESET} ${DIM}[leave blank to auto-generate]${RESET}: "
    read -rs value; echo ""
  else
    echo -ne "    ${YELLOW}${label}${RESET} ${DIM}[${default}]${RESET}: "
    read -r value
  fi
  echo "${value:-$default}"
}

# ── Banner ───────────────────────────────────────────────────────────────────
print_banner

# ── Requirements ─────────────────────────────────────────────────────────────
hr
echo -e "  ${BOLD}Checking requirements...${RESET}"
hr

step "Docker"
command -v docker >/dev/null 2>&1 || die "Docker not found. Install: https://docs.docker.com/get-docker/"
DOCKER_VER=$(docker version --format '{{.Server.Version}}' 2>/dev/null || echo "unknown")
ok "Docker $DOCKER_VER"

step "Docker Compose"
docker compose version >/dev/null 2>&1 || die "Docker Compose plugin not found. Run: apt install docker-compose-plugin"
COMPOSE_VER=$(docker compose version --short 2>/dev/null || echo "unknown")
ok "Docker Compose $COMPOSE_VER"

step "curl"
command -v curl >/dev/null 2>&1 || die "curl not found. Run: apt install curl"
ok "curl $(curl --version | head -1 | awk '{print $2}')"

# ── Download files ────────────────────────────────────────────────────────────
echo ""
hr
echo -e "  ${BOLD}Downloading NovaPanel...${RESET}"
hr

mkdir -p "$INSTALL_DIR" && cd "$INSTALL_DIR"

step "docker-compose.yml"
curl -fsSL "$REPO/docker-compose.hub.yml" -o docker-compose.yml
ok "docker-compose.yml"

step "Detecting server IP"
SERVER_IP=$(curl -fsSL --max-time 5 ifconfig.me 2>/dev/null \
         || curl -fsSL --max-time 5 icanhazip.com 2>/dev/null \
         || hostname -I | awk '{print $1}')
ok "Server IP: $SERVER_IP"

# ── .env already exists ───────────────────────────────────────────────────────
if [ -f .env ]; then
  warn ".env already exists — skipping configuration (delete it to reconfigure)"
else

  # ── Configuration mode ────────────────────────────────────────────────────
  echo ""
  hr
  echo -e "  ${BOLD}Configuration${RESET}"
  hr
  echo ""
  echo -e "  How would you like to configure NovaPanel?"
  echo ""
  echo -e "    ${CYAN}${BOLD}[1]${RESET} Auto  — generate all secrets automatically ${DIM}(recommended, ready in seconds)${RESET}"
  echo -e "    ${CYAN}${BOLD}[2]${RESET} Manual — enter each value yourself"
  echo ""
  echo -ne "  ${BOLD}Your choice [1]:${RESET} "
  read -r MODE_INPUT
  MODE="${MODE_INPUT:-1}"
  echo ""

  # ── Pre-generate secrets (used in auto mode, or as defaults in manual) ────
  GEN_DB_PASS=$(rand_pass)
  GEN_REDIS_PASS=$(rand_pass)
  GEN_JWT=$(rand_hex 32)
  GEN_ENC=$(rand_hex 32)$(rand_hex 32)   # 64-char hex

  if [ "$MODE" = "2" ]; then
    # ── Manual mode ──────────────────────────────────────────────────────────
    echo -e "  ${BOLD}Core settings${RESET}  ${DIM}(press Enter to use the suggested value)${RESET}"
    echo ""

    DB_NAME=$(ask "Database name" "novapanel")
    DB_USER=$(ask "Database user" "novapanel")

    echo -ne "    ${YELLOW}DB_PASSWORD${RESET} ${DIM}[leave blank to auto-generate]${RESET}: "
    read -rs DB_PASS_IN; echo ""
    DB_PASS="${DB_PASS_IN:-$GEN_DB_PASS}"

    echo -ne "    ${YELLOW}REDIS_PASSWORD${RESET} ${DIM}[leave blank to auto-generate]${RESET}: "
    read -rs REDIS_PASS_IN; echo ""
    REDIS_PASS="${REDIS_PASS_IN:-$GEN_REDIS_PASS}"

    echo -ne "    ${YELLOW}JWT_SECRET${RESET} ${DIM}[leave blank to auto-generate]${RESET}: "
    read -rs JWT_IN; echo ""
    JWT="${JWT_IN:-$GEN_JWT}"

    echo -ne "    ${YELLOW}ENCRYPTION_KEY${RESET} ${DIM}[leave blank to auto-generate]${RESET}: "
    read -rs ENC_IN; echo ""
    ENC="${ENC_IN:-$GEN_ENC}"

    PANEL_PORT=$(ask "Panel port" "$PORT")
    PORT="$PANEL_PORT"

    echo ""
    echo -e "  ${BOLD}SMTP  ${DIM}(optional — enables password reset emails)${RESET}"
    echo ""
    SMTP_HOST=$(ask "SMTP host" "")
    SMTP_PORT=$(ask "SMTP port" "587")
    SMTP_USER=$(ask "SMTP user" "")
    echo -ne "    ${YELLOW}SMTP_PASSWORD${RESET}: "; read -rs SMTP_PASS; echo ""
    SMTP_FROM=$(ask "SMTP from" "NovaPanel <noreply@example.com>")

    echo ""
    echo -e "  ${BOLD}Stripe  ${DIM}(optional — enables billing)${RESET}"
    echo ""
    echo -ne "    ${YELLOW}STRIPE_SECRET_KEY${RESET}: "; read -rs STRIPE_KEY; echo ""
    echo -ne "    ${YELLOW}STRIPE_WEBHOOK_SECRET${RESET}: "; read -rs STRIPE_WH; echo ""

  else
    # ── Auto mode ─────────────────────────────────────────────────────────────
    DB_NAME="novapanel"
    DB_USER="novapanel"
    DB_PASS="$GEN_DB_PASS"
    REDIS_PASS="$GEN_REDIS_PASS"
    JWT="$GEN_JWT"
    ENC="$GEN_ENC"
    SMTP_HOST=""; SMTP_PORT="587"; SMTP_USER=""; SMTP_PASS=""; SMTP_FROM=""
    STRIPE_KEY=""; STRIPE_WH=""
  fi

  # ── Write .env ───────────────────────────────────────────────────────────
  curl -fsSL "$REPO/.env.example" -o .env

  sed -i "s|^DB_NAME=.*|DB_NAME=${DB_NAME}|"                             .env
  sed -i "s|^DB_USER=.*|DB_USER=${DB_USER}|"                             .env
  sed -i "s|change_me_strong_password|${DB_PASS}|g"                      .env
  sed -i "s|change_me_redis_password|${REDIS_PASS}|g"                    .env
  sed -i "s|change_me_jwt_secret_at_least_32_chars|${JWT}|g"             .env
  sed -i "s|change_me_64_char_hex_key_here_0000000000000000000000000000000000|${ENC}|g" .env
  sed -i "s|http://your-server-ip:8080|http://${SERVER_IP}:${PORT}|g"    .env
  sed -i "s|^PANEL_PORT=.*|PANEL_PORT=${PORT}|"                          .env
  [ -n "$SMTP_HOST" ]  && sed -i "s|^SMTP_HOST=.*|SMTP_HOST=${SMTP_HOST}|"         .env
  [ -n "$SMTP_USER" ]  && sed -i "s|^SMTP_USER=.*|SMTP_USER=${SMTP_USER}|"         .env
  [ -n "$SMTP_PASS" ]  && sed -i "s|^SMTP_PASSWORD=.*|SMTP_PASSWORD=${SMTP_PASS}|" .env
  [ -n "$SMTP_FROM" ]  && sed -i "s|^SMTP_FROM=.*|SMTP_FROM=${SMTP_FROM}|"         .env
  sed -i "s|^SMTP_PORT=.*|SMTP_PORT=${SMTP_PORT}|"                       .env
  [ -n "$STRIPE_KEY" ] && sed -i "s|^STRIPE_SECRET_KEY=.*|STRIPE_SECRET_KEY=${STRIPE_KEY}|"         .env
  [ -n "$STRIPE_WH" ]  && sed -i "s|^STRIPE_WEBHOOK_SECRET=.*|STRIPE_WEBHOOK_SECRET=${STRIPE_WH}|"  .env

  ok ".env written"
fi

# ── Summary ──────────────────────────────────────────────────────────────────
echo ""
hr
echo ""
echo -e "  ${GREEN}${BOLD}NovaPanel is ready to launch!${RESET}"
echo ""

if [ "$MODE" = "2" ] 2>/dev/null; then
  # Show which optionals were configured
  echo -e "  ${BOLD}Configuration summary:${RESET}"
  echo ""
  ok "Database password       set"
  ok "Redis password          set"
  ok "JWT secret              set"
  ok "Encryption key          set"
  [ -n "${SMTP_HOST:-}" ] && ok "SMTP                    configured" || skipped "SMTP"
  [ -n "${STRIPE_KEY:-}" ] && ok "Stripe billing          configured" || skipped "Stripe billing"
  echo ""
fi

hr
echo ""
echo -e "  ${BOLD}Start NovaPanel:${RESET}"
echo ""
echo -e "    ${CYAN}cd ${INSTALL_DIR}${RESET}"
echo -e "    ${CYAN}docker compose up -d${RESET}"
echo ""
echo -e "  ${BOLD}Then open:${RESET}"
echo ""
echo -e "    ${GREEN}${BOLD}http://${SERVER_IP}:${PORT}${RESET}"
echo ""
echo -e "  ${DIM}The first user you register becomes the administrator.${RESET}"
echo -e "  ${DIM}Your .env is saved at: $(pwd)/.env${RESET}"
echo ""
hr
echo ""
