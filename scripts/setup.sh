#!/usr/bin/env bash
set -euo pipefail

echo "── CoreKit Setup ──"
echo ""

# Check prerequisites
command -v docker >/dev/null 2>&1 || { echo "Error: Docker is required"; exit 1; }
command -v go >/dev/null 2>&1 || { echo "Error: Go 1.25+ is required"; exit 1; }
command -v node >/dev/null 2>&1 || { echo "Warning: Node.js not found (needed for frontend)"; }

# Create backend/.env if missing
if [ ! -f backend/.env ]; then
  cp backend/.env.example backend/.env
  # Generate random JWT secret
  JWT_SECRET=$(openssl rand -hex 64 2>/dev/null || echo "change-this-to-a-random-64-byte-hex-string-in-production")
  if [ "$(uname)" = "Darwin" ]; then
    sed -i '' "s/change-this-to-a-random-64-byte-hex-string-in-production/$JWT_SECRET/" backend/.env
  else
    sed -i "s/change-this-to-a-random-64-byte-hex-string-in-production/$JWT_SECRET/" backend/.env
  fi
  echo "Created backend/.env from backend/.env.example with a generated JWT_SECRET."
fi

# Install frontend deps
if [ -f frontend/package.json ]; then
  echo "Installing frontend dependencies..."
  (cd frontend && npm install)
fi

# Start infrastructure
echo "Starting Docker infrastructure (Postgres, Redis, MinIO)..."
docker compose up -d db redis minio 2>/dev/null || true

echo ""
echo "── Setup complete ──"
echo ""
echo "Commands:"
echo "  make dev       Start backend + frontend"
echo "  make dev-be    Start backend only"
echo "  make lint      Run linters"
echo "  make test      Run tests"
echo "  make db-reset  Reset database"
echo ""
echo "Visit: http://localhost:5173 (frontend)"
echo "       http://localhost:8080/api/health (API)"
echo ""
