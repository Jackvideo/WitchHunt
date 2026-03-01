#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[deploy]${NC} $*"; }
warn() { echo -e "${YELLOW}[warn]${NC} $*"; }
fail() { echo -e "${RED}[error]${NC} $*"; exit 1; }

# ── .env check ──
if [ ! -f .env ]; then
  fail ".env file not found. Copy .env.example and fill in the values."
fi
for var in DB_ROOT_PASSWORD DB_PASSWORD JWT_SECRET; do
  if ! grep -q "^${var}=" .env; then
    fail "Missing required variable: ${var} in .env"
  fi
done
log ".env check passed"

# ── Swap check (for small VMs) ──
TOTAL_MEM_MB=$(free -m 2>/dev/null | awk '/^Mem:/{print $2}' || echo 0)
SWAP_MB=$(free -m 2>/dev/null | awk '/^Swap:/{print $2}' || echo 0)

if [ "$TOTAL_MEM_MB" -gt 0 ] && [ "$TOTAL_MEM_MB" -lt 4096 ] && [ "$SWAP_MB" -lt 512 ]; then
  warn "Detected low memory (${TOTAL_MEM_MB}MB RAM, ${SWAP_MB}MB swap)"
  if [ "$(id -u)" = "0" ]; then
    log "Creating 2GB swap file to prevent OOM during build..."
    if [ ! -f /swapfile ]; then
      dd if=/dev/zero of=/swapfile bs=1M count=2048 status=progress
      chmod 600 /swapfile
      mkswap /swapfile
    fi
    swapon /swapfile 2>/dev/null || true
    log "Swap activated: $(free -m | awk '/^Swap:/{print $2}')MB"
  else
    warn "Run as root to auto-create swap, or add swap manually"
    warn "  sudo fallocate -l 2G /swapfile && sudo chmod 600 /swapfile"
    warn "  sudo mkswap /swapfile && sudo swapon /swapfile"
  fi
fi

# ── Docker check ──
if ! command -v docker &>/dev/null; then
  fail "docker not found. Install Docker first."
fi
if ! docker info &>/dev/null; then
  fail "Docker daemon not running."
fi
log "Docker OK"

export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1

COMPOSE_CMD="docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod"

# ── Build sequentially to avoid OOM ──
log "Building backend..."
$COMPOSE_CMD build backend

log "Building frontend..."
$COMPOSE_CMD build frontend

# ── Start services ──
log "Starting all services..."
$COMPOSE_CMD up -d

# ── Wait for health ──
log "Waiting for services to become healthy..."
for i in $(seq 1 60); do
  HEALTH=$(docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod ps --format json 2>/dev/null \
    | grep -o '"Health":"[^"]*"' | sort -u || true)
  TOTAL=$(docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod ps -q 2>/dev/null | wc -l | tr -d ' ')
  HEALTHY=$(docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod ps --format json 2>/dev/null \
    | grep -c '"Health":"healthy"' || echo 0)

  echo -ne "\r  containers: ${HEALTHY}/${TOTAL} healthy (${i}s)"

  if [ "$HEALTHY" -ge "$TOTAL" ] && [ "$TOTAL" -gt 0 ]; then
    echo ""
    log "All ${TOTAL} services healthy!"
    break
  fi
  sleep 2
done

echo ""
$COMPOSE_CMD ps
log "Deploy complete. Access at http://localhost:${HOST_PORT:-80}"
