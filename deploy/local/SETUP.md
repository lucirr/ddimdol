# Local Development Environment Setup Guide

This guide walks you through setting up the complete Edge DIP local development environment.

## Prerequisites

### Required
- Docker Desktop 4.0+ (or Docker Engine + Docker Compose)
- Make (macOS/Linux) or compatible alternative
- Git

### System Requirements
- Minimum 2GB available RAM (4GB recommended)
- Minimum 10GB available disk space
- Ports 5432, 6379, 4222, 8222, 8180 available (or modify in docker-compose.yml)

## Installation Steps

### 1. Verify Docker Installation

```bash
docker --version
docker compose --version
```

Expected output:
```
Docker version 24.0+
Docker Compose version 2.0+
```

### 2. Navigate to Local Environment Directory

```bash
cd /Users/lucirr/workspace/didimdol/deploy/local
```

### 3. Prepare Environment Configuration

```bash
# Copy environment template
cp .env.example .env

# Review and edit .env if needed
# Default values work for local development out of the box
```

### 4. Start All Services

```bash
# Option A: Using Make (recommended)
make up

# Option B: Using docker compose directly
docker compose up -d
```

### 5. Wait for Services to Be Ready

Services initialize in this order:
1. PostgreSQL (5-10 seconds)
2. Redis (2-3 seconds)
3. NATS (3-5 seconds)
4. Keycloak (30-60 seconds) - depends on PostgreSQL

Check service status:

```bash
make ps
```

Or monitor logs:

```bash
make logs
```

### 6. Verify All Services Are Healthy

```bash
make status
```

Expected output:
```
=== Docker Compose Status ===
NAME       IMAGE                              STATUS
postgres   postgres:16-alpine                 running (healthy)
redis      redis:7-alpine                     running (healthy)
nats       nats:2.10-alpine                   running (healthy)
keycloak   quay.io/keycloak/keycloak:24.0    running (healthy)

=== Service Health Check ===
✓ PostgreSQL is healthy
✓ Redis is healthy
✓ NATS is healthy
✓ Keycloak is healthy
```

## Service Access

### PostgreSQL Database

Access the database directly:

```bash
make postgres
# or
docker compose exec postgres psql -U edgedip -d edgedip
```

Common PostgreSQL commands:
```sql
\dt                  -- List tables
\l                   -- List databases
\c keycloak          -- Connect to keycloak database
SELECT * FROM ...    -- Query data
```

### Redis Cache

Interactive Redis CLI:

```bash
make redis
# or
docker compose exec redis redis-cli
```

Common Redis commands:
```
PING                 -- Check connection
KEYS *               -- List all keys
GET <key>            -- Get value
SET <key> <value>    -- Set value
FLUSHALL             -- Clear all data
```

### NATS Messaging

Web Dashboard: http://localhost:8222

View NATS logs:

```bash
make nats-logs
# or
docker compose logs -f nats
```

### Keycloak Identity Server

Web Console: http://localhost:8180

Default credentials:
- Username: `admin`
- Password: `admin`

Steps to configure a realm:
1. Log in to Keycloak
2. Create realm "edgedip"
3. Create clients for your applications
4. Configure users and roles

## Running Applications

### Terminal 1: Start Infrastructure

```bash
cd deploy/local
make up
```

### Terminal 2: Run Portal API

```bash
cd portal-api
make run
# or
go run ./cmd/server/main.go
```

Portal API will be available at: http://localhost:8080

### Terminal 3: Run Portal Web

```bash
cd portal-web
npm install
npm run dev
```

Portal Web will be available at: http://localhost:5173 (or configured port)

## Database Migrations

### Automatic Initialization

SQL migration files in `portal-api/migrations/` directory are automatically applied when PostgreSQL starts for the first time.

### Manual Migrations

To apply migrations manually:

```bash
# Run migrations against running PostgreSQL
docker compose exec postgres psql -U edgedip -d edgedip -f /docker-entrypoint-initdb.d/migrations/<migration-file>.sql
```

### Creating New Migrations

1. Create a SQL file in `portal-api/migrations/` directory
2. Follow naming convention: `001_create_tables.sql`, `002_add_columns.sql`, etc.
3. Stop and reset the environment:

```bash
make reset
```

4. Restart services - migrations will run automatically

## Troubleshooting

### Service Fails to Start

Check logs for the specific service:

```bash
# PostgreSQL
docker compose logs postgres

# Redis
docker compose logs redis

# NATS
docker compose logs nats

# Keycloak
docker compose logs keycloak
```

### Port Already in Use

If you get "port already in use" error:

1. Check what's using the port:
```bash
lsof -i :5432  # for PostgreSQL
lsof -i :6379  # for Redis
lsof -i :4222  # for NATS
lsof -i :8180  # for Keycloak
```

2. Either stop the conflicting service or modify ports in `docker-compose.yml`:

```yaml
postgres:
  ports:
    - "5433:5432"  # Change first number to unused port
```

3. Update `.env` accordingly:
```env
DATABASE_URL=postgres://edgedip:edgedip_secret@localhost:5433/edgedip?sslmode=disable
```

### Database Connection Refused

Ensure PostgreSQL is fully started and healthy:

```bash
docker compose exec postgres pg_isready -U edgedip
```

Wait 5-10 seconds and retry if not ready.

### Keycloak Won't Connect to Database

1. Verify PostgreSQL is healthy:
```bash
docker compose logs postgres
```

2. Check Keycloak logs:
```bash
docker compose logs keycloak
```

3. If keycloak DB wasn't created, manually create it:
```bash
docker compose exec postgres psql -U edgedip -c "CREATE DATABASE keycloak OWNER edgedip;"
```

### Out of Disk Space

Check Docker disk usage:

```bash
docker system df
```

Clean up unused containers and images:

```bash
make clean
```

Or completely reset everything:

```bash
make reset
```

## Data Persistence

### Understanding Volumes

- `postgres_data`: PostgreSQL database files (survives container restart)
- `redis_data`: Redis persistence file (survives container restart)
- `nats_data`: NATS journal files (survives container restart)

### Backup Data

#### PostgreSQL Full Backup

```bash
docker compose exec postgres pg_dump -U edgedip -d edgedip > backup-$(date +%Y%m%d_%H%M%S).sql
```

#### PostgreSQL Restore

```bash
docker compose exec -T postgres psql -U edgedip -d edgedip < backup.sql
```

#### Redis Backup

```bash
docker compose exec redis redis-cli BGSAVE
docker compose cp redis:/data/dump.rdb ./redis-backup.rdb
```

### Complete Environment Reset

WARNING: This removes all data and volumes.

```bash
make reset
```

Or step by step:

```bash
docker compose down -v  # Stop and remove volumes
docker compose up -d    # Start fresh
```

## Performance Tuning

### Increase Memory for Services

Edit `docker-compose.yml` to add resource limits:

```yaml
postgres:
  deploy:
    resources:
      limits:
        memory: 2G
  # ... rest of config
```

### Enable Persistence for Redis

Already configured with `appendonly yes` in docker-compose.yml.

## Security Notes for Development

### Default Credentials

These are development defaults only. Change before any production use:

```
PostgreSQL:
  User: edgedip
  Password: edgedip_secret

Keycloak:
  Admin: admin
  Password: admin
```

### Local Network Only

Services are bound to localhost (127.0.0.1) by default and not accessible from other machines. To expose to network:

```yaml
postgres:
  ports:
    - "0.0.0.0:5432:5432"  # WARNING: Exposes to network
```

## Next Steps

1. Configure Keycloak realm and clients
2. Create database schema using migrations
3. Start Portal API with environment variables loaded from `.env`
4. Start Portal Web development server
5. Access applications at configured URLs

## Support

For issues or questions:
1. Check logs: `make logs`
2. Verify service health: `make status`
3. Review README.md for additional information
4. Check Docker Desktop system resources

## Useful Commands Reference

```bash
# Service Management
make up              # Start all services
make down            # Stop services (keep data)
make reset           # Stop and reset everything
make logs            # Follow all logs
make ps              # Show running services
make status          # Health check all services

# Direct Service Access
make postgres        # PostgreSQL CLI
make redis           # Redis CLI
make nats            # NATS logs
make keycloak-logs   # Keycloak logs

# Cleanup
make clean           # Remove stopped containers

# Docker Compose (alternative)
docker compose ps
docker compose logs <service>
docker compose exec <service> <command>
docker compose down -v
```
