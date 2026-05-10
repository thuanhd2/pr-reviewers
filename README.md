# PR Reviewer

Auto-review GitHub pull requests using AI CLI agents.

## Prerequisites

- Go 1.23+
- Node.js 22+
- pnpm
- PostgreSQL 16
- Redis
- Git

## Setup

```bash
# Database
./scripts/setup-db.sh

# Backend
cp config.example.yaml config.yaml
# Edit config.yaml with your GitHub token and repo mappings

# Frontend
cd web && pnpm install
```

## Development

```bash
# Terminal 1: API server
DATABASE_URL="postgres://pr_reviewer:pr_reviewer_dev@localhost:5432/pr_reviewer?sslmode=disable" \
REDIS_URL="localhost:6379" \
go run ./cmd/api

# Terminal 2: Worker
DATABASE_URL="postgres://pr_reviewer:pr_reviewer_dev@localhost:5432/pr_reviewer?sslmode=disable" \
REDIS_URL="localhost:6379" \
go run ./cmd/worker

# Terminal 3: Frontend
cd web && pnpm dev
```

- API: `http://localhost:8080`
- Frontend: `http://localhost:5173` (proxies API + WS to `:8080`)

## Production

```bash
make build    # Build Go binaries + frontend
make start    # Start via PM2
make stop     # Stop
make logs     # View logs
make status   # Check status
```

## Architecture

```
Host Machine
├── pm2
│   ├── pr-reviewer-api      (Go + Gin, serves API & frontend)
│   └── pr-reviewer-worker   (Go + Asynq, scheduler + tasks)
├── PostgreSQL :5432
├── Redis      :6379
└── Vite Dev   :5173
```
