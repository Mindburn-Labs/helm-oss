#!/usr/bin/env bash
# HELM Demo — Manual state reset
# Use this to reset the demo database without waiting for the cron.
set -euo pipefail

COMPOSE_FILE="${1:-docker-compose.demo.yml}"

echo "HELM Demo Reset"
echo "═══════════════"

# 1. Stop HELM (not postgres)
echo "  Stopping helm..."
docker compose -f "$COMPOSE_FILE" stop helm

# 2. Reset database
echo "  Resetting database..."
docker compose -f "$COMPOSE_FILE" exec -T postgres \
    psql -U helm_demo -d postgres -c "
        SELECT pg_terminate_backend(pid)
        FROM pg_stat_activity
        WHERE datname='helm_demo' AND pid <> pg_backend_pid();
    " 2>/dev/null || true

docker compose -f "$COMPOSE_FILE" exec -T postgres \
    dropdb -U helm_demo --if-exists helm_demo 2>/dev/null || true

docker compose -f "$COMPOSE_FILE" exec -T postgres \
    createdb -U helm_demo helm_demo 2>/dev/null || true

# 3. Restart HELM
echo "  Starting helm..."
docker compose -f "$COMPOSE_FILE" start helm

echo "  ✅ Demo state cleared and restarted"
