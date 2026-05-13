.PHONY: up down logs ps reset build clean help

help:
	@echo "Available commands:"
	@echo "  make up       - Start all services"
	@echo "  make down     - Stop all services"
	@echo "  make logs     - Follow service logs"
	@echo "  make ps       - Show running services"
	@echo "  make reset    - Stop and remove all volumes (WARNING: data loss)"
	@echo "  make build    - Build docker images"
	@echo "  make clean    - Remove all stopped containers and dangling images"

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

ps:
	docker compose ps

reset:
	docker compose down -v
	docker compose up -d

build:
	docker compose build --no-cache

clean:
	docker compose down
	docker system prune -f

# Service-specific commands
postgres:
	docker compose exec postgres psql -U edgedip -d edgedip

redis:
	docker compose exec redis redis-cli

nats:
	docker compose logs -f nats

keycloak-logs:
	docker compose logs -f keycloak

# Utility commands
status:
	@echo "=== Docker Compose Status ==="
	docker compose ps
	@echo ""
	@echo "=== Service Health Check ==="
	@docker compose exec -T postgres pg_isready -U edgedip > /dev/null 2>&1 && echo "✓ PostgreSQL is healthy" || echo "✗ PostgreSQL is down"
	@docker compose exec -T redis redis-cli ping > /dev/null 2>&1 && echo "✓ Redis is healthy" || echo "✗ Redis is down"
	@docker compose exec -T nats wget -qO- http://localhost:8222/healthz > /dev/null 2>&1 && echo "✓ NATS is healthy" || echo "✗ NATS is down"
	@curl -s http://localhost:8180 > /dev/null && echo "✓ Keycloak is healthy" || echo "✗ Keycloak is down"

.DEFAULT_GOAL := help
