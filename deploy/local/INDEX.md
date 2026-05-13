# Edge DIP Local Development Infrastructure - File Index

## Quick Navigation

| File | Purpose | Size |
|------|---------|------|
| **quick-start.sh** | Automated setup script | 5.9 KB |
| **docker-compose.yml** | Docker services definition | 1.6 KB |
| **Makefile** | Convenient command shortcuts | 1.6 KB |
| **.env.example** | Environment variables template | 622 B |
| **init-db.sql** | PostgreSQL initialization script | 510 B |
| **README.md** | Service documentation | 4.2 KB |
| **SETUP.md** | Detailed setup guide | 8.3 KB |
| **.gitignore** | Git ignore patterns | 211 B |

## Getting Started (Choose One)

### Option 1: Automated Setup (Recommended)
```bash
cd /Users/lucirr/workspace/didimdol/deploy/local
./quick-start.sh
```

### Option 2: Manual Setup
```bash
cd /Users/lucirr/workspace/didimdol/deploy/local
cp .env.example .env
make up
make status
```

## File Descriptions

### docker-compose.yml
Complete Docker Compose configuration for:
- **PostgreSQL 16**: Database server with Keycloak support
- **Redis 7**: In-memory cache with persistence
- **NATS 2.10**: Message streaming platform
- **Keycloak 24.0**: Identity and access management

All services include health checks and proper dependencies.

### Makefile
Convenient commands for service management:
- `make up` - Start services
- `make down` - Stop services
- `make logs` - Follow logs
- `make status` - Health check
- `make postgres` - PostgreSQL CLI
- `make redis` - Redis CLI
- `make reset` - Full reset with data wipe

### .env.example
Template for environment variables. Copy to `.env` and customize as needed for:
- Database credentials
- Service URLs
- Keycloak configuration
- API ports

### init-db.sql
PostgreSQL initialization script that:
- Creates the `keycloak` database
- Sets up proper permissions for the `edgedip` user
- Runs automatically on first PostgreSQL startup

### quick-start.sh
Automated setup script that:
1. Checks Docker prerequisites
2. Creates `.env` from template
3. Starts all services
4. Waits for services to be healthy
5. Displays access information

### README.md
Complete documentation including:
- Service overview with ports and credentials
- Quick start instructions
- Environment configuration
- Database and service access methods
- Troubleshooting guide
- Backup and restore procedures

### SETUP.md
Comprehensive setup guide with:
- Detailed installation steps
- Service verification procedures
- Running applications
- Database migration instructions
- Extensive troubleshooting section
- Performance tuning tips
- Security notes

### .gitignore
Prevents accidental commits of:
- Environment variables (`.env`)
- Backup files
- OS files (`.DS_Store`, `Thumbs.db`)
- IDE configuration
- Log files

## Service Access

Once services are running:

| Service | URL/Port | Credentials |
|---------|----------|-------------|
| PostgreSQL | localhost:5432 | edgedip / edgedip_secret |
| Redis | localhost:6379 | None (auth optional) |
| NATS Client | localhost:4222 | None |
| NATS Dashboard | http://localhost:8222 | None |
| Keycloak | http://localhost:8180 | admin / admin |

## Common Tasks

### Start Development Environment
```bash
cd deploy/local
./quick-start.sh
```

### View All Service Logs
```bash
make logs
```

### Check Service Health
```bash
make status
```

### Connect to PostgreSQL
```bash
make postgres
```

### Reset Everything (WARNING: Data Loss)
```bash
make reset
```

### Stop Without Losing Data
```bash
make down
```

## Project Structure

```
didimdol/
├── deploy/
│   └── local/                 # This directory
│       ├── docker-compose.yml # Service definitions
│       ├── Makefile          # Command shortcuts
│       ├── quick-start.sh    # Automated setup
│       ├── .env.example      # Config template
│       ├── init-db.sql       # DB initialization
│       ├── README.md         # Documentation
│       ├── SETUP.md          # Setup guide
│       └── .gitignore        # Git patterns
├── portal-api/               # Go API server
├── portal-web/               # React web frontend
└── docs/                     # Documentation
```

## Troubleshooting

Consult **SETUP.md** for detailed troubleshooting including:
- Service startup issues
- Port conflicts
- Database connection problems
- Performance optimization
- Backup and restore procedures

## Key Configuration Files

### docker-compose.yml
- Service images and versions
- Port mappings
- Environment variables
- Volume definitions
- Health check configurations
- Service dependencies

### .env.example / .env
- Database connection strings
- Service credentials
- API configuration
- Client URLs
- Log levels

### Makefile
- Service lifecycle commands
- Database and cache access
- Health monitoring
- Cleanup operations

## Next Steps

1. Run `./quick-start.sh` to set up the environment
2. Read **README.md** for service documentation
3. Check **SETUP.md** if you encounter any issues
4. Start Portal API and Portal Web in separate terminals
5. Access services at configured URLs

## Support Resources

- **Setup Issues**: See SETUP.md troubleshooting section
- **Service Details**: See README.md
- **Docker Compose Reference**: https://docs.docker.com/compose/
- **Keycloak Documentation**: https://www.keycloak.org/documentation

---
Last updated: 2026-05-13
