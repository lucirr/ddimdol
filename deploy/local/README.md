# Edge DIP Local Development Infrastructure

Local development environment setup using docker-compose for the Edge DIP Central Control System.

## Services

- **PostgreSQL 16**: Primary database for application and Keycloak
  - Port: 5432
  - User: `edgedip`
  - Password: `edgedip_secret`
  - Databases: `edgedip` (application), `keycloak` (auth)

- **Redis 7**: In-memory cache and message broker
  - Port: 6379

- **NATS 2.10**: Message streaming and pub/sub
  - Port: 4222 (client)
  - Port: 8222 (monitoring)

- **Keycloak 24.0**: OpenID Connect identity provider
  - Port: 8180
  - Admin username: `admin`
  - Admin password: `admin`
  - Realm: `edgedip`

## Prerequisites

- Docker and Docker Compose
- Make (optional, for convenience commands)

## Quick Start

### 1. Start all services

```bash
cd deploy/local
docker compose up -d
```

Or using Make:

```bash
make up
```

### 2. Verify services are running

```bash
docker compose ps
```

Or using Make:

```bash
make ps
```

### 3. Check service health

```bash
make status
```

### 4. View logs

```bash
docker compose logs -f
```

Or follow specific service:

```bash
make postgres      # PostgreSQL logs
make redis         # Redis CLI
make nats          # NATS logs
make keycloak-logs # Keycloak logs
```

## Environment Configuration

1. Copy the environment template:

```bash
cp .env.example .env
```

2. Update `.env` with your local configuration if needed.

3. Load environment variables:

```bash
export $(cat .env | xargs)
```

## Database Access

### PostgreSQL

Connect to the database:

```bash
docker compose exec postgres psql -U edgedip -d edgedip
```

Or using Make:

```bash
make postgres
```

### Create initial schema

Migration files from `portal-api/migrations/` are automatically loaded on startup. Add SQL files to that directory to initialize the database schema.

## Redis Access

Connect to Redis:

```bash
docker compose exec redis redis-cli
```

Or using Make:

```bash
make redis
```

## NATS Access

View NATS web dashboard: http://localhost:8222

## Keycloak Administration

1. Open Keycloak console: http://localhost:8180
2. Log in with:
   - Username: `admin`
   - Password: `admin`
3. Create realm `edgedip` if needed
4. Configure clients for your applications

## Cleanup Commands

### Stop services (keep data)

```bash
docker compose down
```

Or:

```bash
make down
```

### Full reset (removes all data)

```bash
docker compose down -v
```

Or:

```bash
make reset
```

### Clean up Docker resources

```bash
make clean
```

## Troubleshooting

### Services not starting

Check service logs:

```bash
docker compose logs <service-name>
```

Available services: `postgres`, `redis`, `nats`, `keycloak`

### Database connection issues

Verify PostgreSQL is healthy:

```bash
docker compose exec postgres pg_isready -U edgedip
```

### Port conflicts

If ports are already in use, modify `docker-compose.yml`:

```yaml
services:
  postgres:
    ports:
      - "5433:5432"  # Change 5433 to your preferred port
```

And update `DATABASE_URL` in `.env` accordingly.

### Keycloak won't start

Ensure PostgreSQL is healthy first:

```bash
docker compose logs postgres
```

Keycloak depends on PostgreSQL being available and healthy.

## Development Workflow

### 1. Start infrastructure

```bash
make up
```

### 2. Run Portal API (in another terminal)

```bash
cd portal-api
make run
```

### 3. Run Portal Web (in another terminal)

```bash
cd portal-web
npm run dev
```

### 4. Access services

- Portal API: http://localhost:8080
- Portal Web: http://localhost:5173 (or configured port)
- Keycloak: http://localhost:8180
- NATS Dashboard: http://localhost:8222

## Backup and Restore

### Backup PostgreSQL

```bash
docker compose exec postgres pg_dump -U edgedip edgedip > backup.sql
```

### Restore PostgreSQL

```bash
docker compose exec -T postgres psql -U edgedip edgedip < backup.sql
```

### Backup Redis

```bash
docker compose exec redis redis-cli BGSAVE
docker compose cp redis:/data/dump.rdb ./redis-backup.rdb
```

## Notes

- All data is persisted in Docker volumes
- The `init-db.sql` script runs on first startup to set up Keycloak database
- Migration files in `portal-api/migrations/` are automatically applied
- Default credentials should be changed for production use
