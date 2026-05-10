.PHONY: dev-infra dev-api dev-worker dev-web dev build start stop restart logs status db/migrate-install db/migrate

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

DATABASE_URL ?= postgres://pr_reviewer:pr_reviewer_dev@localhost:5432/pr_reviewer?sslmode=disable

db/migrate-install:
	@mkdir -p vendors
	@GOOSE_OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
	GOOSE_ARCH=$$(uname -m); \
	case "$$GOOSE_ARCH" in \
		x86_64) GOOSE_ARCH="x86_64" ;; \
		arm64)  GOOSE_ARCH="aarch64" ;; \
		aarch64) GOOSE_ARCH="aarch64" ;; \
		*) echo "Unsupported arch: $$GOOSE_ARCH"; exit 1 ;; \
	esac; \
	case "$$GOOSE_OS" in \
		darwin) GOOSE_TARGET="$${GOOSE_ARCH}-apple-darwin" ;; \
		linux)  GOOSE_TARGET="$${GOOSE_ARCH}-unknown-linux-gnu" ;; \
		*)      echo "Unsupported OS: $$GOOSE_OS"; exit 1 ;; \
	esac; \
	ASSET="goose-$${GOOSE_TARGET}.tar.gz"; \
	URL="https://github.com/aaif-goose/goose/releases/latest/download/$${ASSET}"; \
	echo "Downloading goose from $$URL"; \
	curl -fsSL -o /tmp/goose.tar.gz "$$URL" && \
	tar -xzf /tmp/goose.tar.gz -C vendors/ && \
	chmod +x vendors/goose && \
	rm -f /tmp/goose.tar.gz && \
	echo "goose installed to vendors/goose"

db/migrate:
	vendors/goose -dir migrations postgres "$(DATABASE_URL)" up
