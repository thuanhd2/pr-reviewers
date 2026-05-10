.PHONY: dev-infra dev-api dev-worker dev-web dev build start stop restart logs status

dev-infra:
	brew services start postgresql@16
	brew services start redis

dev-api:
	DATABASE_URL="postgres://pr_reviewer:pr_reviewer_dev@localhost:5432/pr_reviewer?sslmode=disable" \
	REDIS_URL="localhost:6379" \
	go run ./cmd/api

dev-worker:
	DATABASE_URL="postgres://pr_reviewer:pr_reviewer_dev@localhost:5432/pr_reviewer?sslmode=disable" \
	REDIS_URL="localhost:6379" \
	go run ./cmd/worker

dev-web:
	cd web && pnpm dev

dev: dev-infra
	@echo "Start in separate terminals:"
	@echo "  Terminal 1: make dev-api"
	@echo "  Terminal 2: make dev-worker"
	@echo "  Terminal 3: make dev-web"

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker
	cd web && pnpm build

start:
	pm2 start ecosystem.config.cjs

stop:
	pm2 stop ecosystem.config.cjs

restart:
	pm2 restart ecosystem.config.cjs

logs:
	pm2 logs

status:
	pm2 status
