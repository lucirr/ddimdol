#!/bin/bash

# Edge DIP Local Development - Quick Start Script
# This script automates the setup of the local development environment

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "╔════════════════════════════════════════════════════════════╗"
echo "║   Edge DIP Local Development Environment - Quick Start     ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# Check prerequisites
echo "📋 Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    echo "✗ Docker is not installed. Please install Docker Desktop."
    exit 1
fi
echo "✓ Docker found: $(docker --version)"

if ! command -v docker compose &> /dev/null; then
    echo "✗ Docker Compose is not installed."
    exit 1
fi
echo "✓ Docker Compose found"

echo ""
echo "📁 Setting up environment..."

# Navigate to script directory
cd "$SCRIPT_DIR"

# Check if .env exists
if [ ! -f .env ]; then
    echo "Creating .env from template..."
    cp .env.example .env
    echo "✓ Created .env (using default values)"
else
    echo "✓ .env already exists"
fi

echo ""
echo "🐳 Starting Docker services..."

# Start services
docker compose up -d

echo "⏳ Waiting for services to be ready..."

# Function to check if service is healthy
check_service() {
    local service=$1
    local max_attempts=30
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if docker compose exec -T "$service" sh -c "exit 0" 2>/dev/null; then
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 1
    done
    return 1
}

# Wait for PostgreSQL
echo -n "Waiting for PostgreSQL... "
if docker compose exec -T postgres pg_isready -U edgedip > /dev/null 2>&1; then
    echo "✓"
else
    echo "⏳ (still starting...)"
fi

# Wait for Redis
echo -n "Waiting for Redis... "
if docker compose exec -T redis redis-cli ping > /dev/null 2>&1; then
    echo "✓"
else
    echo "⏳ (still starting...)"
fi

# Wait for NATS
echo -n "Waiting for NATS... "
if docker compose exec -T nats nc -z localhost 4222 2>/dev/null; then
    echo "✓"
else
    echo "⏳ (still starting...)"
fi

# Wait for Keycloak (longer timeout)
echo "Waiting for Keycloak... (this may take 30-60 seconds)"
for i in {1..60}; do
    if curl -s http://localhost:8180 > /dev/null 2>&1; then
        echo "✓"
        break
    fi
    echo -n "."
    sleep 1
done

echo ""
echo "✅ All services are running!"
echo ""
echo "╔════════════════════════════════════════════════════════════╗"
echo "║                    Service Access Information              ║"
echo "╠════════════════════════════════════════════════════════════╣"
echo "║ PostgreSQL                                                 ║"
echo "║   Connection: localhost:5432                              ║"
echo "║   User: edgedip | Password: edgedip_secret                ║"
echo "║   Databases: edgedip, keycloak                            ║"
echo "║                                                            ║"
echo "║ Redis                                                      ║"
echo "║   Connection: localhost:6379                              ║"
echo "║                                                            ║"
echo "║ NATS                                                       ║"
echo "║   Connection: localhost:4222                              ║"
echo "║   Dashboard: http://localhost:8222                        ║"
echo "║                                                            ║"
echo "║ Keycloak                                                   ║"
echo "║   URL: http://localhost:8180                              ║"
echo "║   User: admin | Password: admin                           ║"
echo "║   Realm: edgedip                                          ║"
echo "╠════════════════════════════════════════════════════════════╣"
echo "║                      Quick Commands                        ║"
echo "╠════════════════════════════════════════════════════════════╣"
echo "║ View logs:              make logs                          ║"
echo "║ Check status:           make status                       ║"
echo "║ Stop services:          make down                         ║"
echo "║ Reset everything:       make reset                        ║"
echo "║ PostgreSQL CLI:         make postgres                     ║"
echo "║ Redis CLI:              make redis                        ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""
echo "📚 Documentation:"
echo "   - README.md for service documentation"
echo "   - SETUP.md for detailed setup and troubleshooting"
echo ""
echo "🚀 Next steps:"
echo "   1. Run Portal API (in another terminal):"
echo "      cd $PROJECT_ROOT/portal-api && make run"
echo "   2. Run Portal Web (in another terminal):"
echo "      cd $PROJECT_ROOT/portal-web && npm install && npm run dev"
echo "   3. Access your services:"
echo "      - API: http://localhost:8080"
echo "      - Web: http://localhost:5173"
echo "      - Keycloak: http://localhost:8180"
echo ""
