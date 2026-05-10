# CLAUDE.md

## Project

PR Reviewer - tự động review GitHub pull requests bằng AI CLI agents (Claude Code, Codex).

## Stack

- **Backend**: Go 1.25 + Gin (API) + Asynq (job queue) + GORM (ORM)
- **Database**: PostgreSQL 16
- **Cache/Queue**: Redis
- **Frontend**: React 19 + TypeScript + Vite + TanStack Router + Shadcn UI + Zustand
- **Real-time**: Centrifuge (WebSocket)
- **Process manager (prod)**: PM2

## Architecture

```
cmd/api       - HTTP server (:8080), phục vụ REST API, WebSocket, và static frontend
cmd/worker    - Background worker, chạy Asynq task handler + scheduler

internal/
  config      - Load YAML config (config.yaml)
  executor    - AI CLI abstraction (Claude Code, Codex)
  github      - GitHub API client
  handler     - HTTP handlers (prs, reviews, configs, dashboard)
  scheduler   - Cron scheduler (fetch PRs định kỳ, cleanup worktree cũ)
  store       - GORM models + DB operations
  task        - Asynq task handlers (fetch, sync, review, post, cleanup)
  ws          - WebSocket hub (Centrifuge)

web/          - React SPA, build ra web/dist/
```

## Data flow

1. Scheduler định kỳ fetch assigned PRs từ GitHub
2. Với mỗi PR có commit mới → tạo review task
3. Worker clone/fetch repo vào git worktree → chạy AI CLI agent review
4. Kết quả review lưu vào DB, có thể post lên GitHub PR

## Dev commands

```bash
# API server
DATABASE_URL="postgres://pr_reviewer:pr_reviewer_dev@localhost:5432/pr_reviewer?sslmode=disable" \
REDIS_URL="localhost:6379" \
go run ./cmd/api

# Worker
DATABASE_URL="postgres://pr_reviewer:pr_reviewer_dev@localhost:5432/pr_reviewer?sslmode=disable" \
REDIS_URL="localhost:6379" \
go run ./cmd/worker

# Frontend
cd web && pnpm dev

# Build all
make build

# Test (frontend)
cd web && pnpm test
```

## Config

Copy `config.example.yaml` → `config.yaml`. Cấu hình GitHub token, repo mappings, executor settings, scheduler intervals.

## Code conventions

- **Go**: Package `internal/`, không export ra ngoài module. Handler pattern: mỗi resource một file handler riêng, dùng `handler.Success()`/`handler.Error()` để trả JSON.
- **React**: File-based routing với TanStack Router (`src/routes/`), feature components trong `src/features/`, reusable UI trong `src/components/ui/` (Shadcn). Global state dùng Zustand (`src/stores/`). API calls qua `src/lib/api.ts`.
- **Không viết comment vô nghĩa**: Code tự document. Chỉ comment khi giải thích WHY, không phải WHAT.
- **Không refactor nếu không cần**: Sửa bug thì chỉ sửa bug, không dọn dẹp code xung quanh.
- **Không thêm abstraction sớm**: 3 dòng giống nhau vẫn tốt hơn 1 abstraction vội vàng.
