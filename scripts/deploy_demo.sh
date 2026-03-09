#!/usr/bin/env bash
# Deploy HELM Demo to DigitalOcean
# Usage: ./deploy_demo.sh [IP_ADDRESS]

set -euo pipefail

HOST="${1:-demo.mindburn.org}"
USER="root"
DIR="/opt/helm"

echo "Deploying to $USER@$HOST..."

# 1. Sync Code
echo "Syncing code..."
rsync -avz --exclude '.git' --exclude 'bin' --exclude 'data' ./ "$USER@$HOST:$DIR"

# 2. Rebuild & Restart
echo "Rebuilding containers..."
ssh "$USER@$HOST" "cd $DIR && docker compose -f docker-compose.demo.yml up -d --build"

# 3. Check Health
echo "Checking health..."
ssh "$USER@$HOST" "curl -s http://localhost:8081/healthz && echo ' OK'"

echo "✅ Deployed successfully!"
