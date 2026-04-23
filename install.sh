#!/usr/bin/env bash
set -euo pipefail

REPO="https://raw.githubusercontent.com/codepromaxtech/novapanel/main"

echo "NovaPanel installer"
echo "==================="

# Check docker + compose are available
command -v docker >/dev/null 2>&1 || { echo "Error: Docker not found. Install Docker first: https://docs.docker.com/get-docker/"; exit 1; }
docker compose version >/dev/null 2>&1 || { echo "Error: Docker Compose plugin not found."; exit 1; }

mkdir -p novapanel && cd novapanel

# Download compose + env template
curl -fsSL "$REPO/docker-compose.hub.yml" -o docker-compose.yml
[ -f .env ] || curl -fsSL "$REPO/.env.example" -o .env

echo ""
echo "Edit .env with your secrets before starting:"
echo "  nano .env"
echo ""
echo "Then start NovaPanel:"
echo "  docker compose up -d"
echo ""
echo "Open http://\$(curl -s ifconfig.me):8080 in your browser."
