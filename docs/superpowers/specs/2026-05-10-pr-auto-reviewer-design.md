# PR Auto Reviewer - Design Spec

## Overview

Web application định kỳ fetch các PR đang assigned cho user trên GitHub, dùng AI CLI agents để tự động review, tạo draft để user duyệt trước khi post lên GitHub.

## Core Flow

```
Fetch PR → AI phân tích → Draft review → User duyệt → Post lên GitHub
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go, Gin framework, GORM |
| Database | PostgreSQL |
| Cache/Queue | Redis, Asynq |
| Real-time | Centrifuge WebSocket |
| Frontend | shadcn-admin template (React 18, TypeScript, Vite, ShadcnUI, TanStack Router, TanStack Table, Zustand) + TanStack Query, Centrifuge client |
| AI Engine | CLI agents installed on host machine (Claude Code, Codex, etc.) |
| Process Manager | PM2 (production) |

## Architecture

```
Host Machine
├── pm2
│   ├── pr-reviewer-api      (Go binary, Gin + embed frontend dist)
│   └── pr-reviewer-worker   (Go binary, Asynq worker + scheduler)
├── PostgreSQL :5432
├── Redis      :6379
├── Vite Dev   :5173          (Vite dev server, proxy API + WS đến :8080)
└── Host tools: npx claude-code, git, local repos
```

- API và Worker là 2 process riêng biệt, cùng monorepo, shared packages
- API server embed static React build (`web/dist/`) khi production
- Worker chạy trực tiếp trên host để gọi được CLI agents và git
- Frontend dựa trên [shadcn-admin](https://github.com/satnaing/shadcn-admin) template, được clone về `web/` rồi strip down và custom hóa

## Monorepo Structure

```
pr-reviewers/
├── cmd/
│   ├── api/main.go            # Gin API server entry (+ embed frontend)
│   └── worker/main.go         # Asynq worker entry
├── internal/
│   ├── executor/              # CLI executor interface + implementations
│   │   ├── executor.go        # Interface
│   │   ├── claude_code.go     # Claude Code CLI
│   │   ├── codex.go           # Codex CLI (extensible)
│   │   ├── registry.go        # Registry: map CLI name → executor
│   │   └── result.go          # ReviewResult parser
│   ├── github/
│   │   └── client.go          # GitHub API client (PAT-based)
│   ├── store/
│   │   ├── models.go          # GORM entities
│   │   └── store.go           # CRUD operations
│   ├── scheduler/
│   │   └── scheduler.go       # Asynq periodic task registration
│   ├── task/
│   │   ├── fetch_prs.go       # Fetch PRs assigned to me
│   │   ├── sync_pr_status.go  # Check closed/merged PRs
│   │   ├── execute_review.go  # Run CLI agent, parse result
│   │   ├── post_review.go     # Post approved review to GitHub
│   │   └── cleanup_worktree.go # Daily cleanup old worktrees
│   ├── ws/
│   │   └── ws.go              # Centrifuge setup + publish helpers
│   ├── config/
│   │   └── config.go          # App config (env + YAML)
│   └── handler/               # Gin HTTP handlers
│       ├── prs.go
│       ├── reviews.go
│       ├── configs.go
│       └── dashboard.go
├── web/                       # React frontend (based on shadcn-admin template)
│   ├── src/
│   │   ├── components/
│   │   │   ├── ui/             # shadcn UI primitives
│   │   │   ├── layout/         # app-sidebar, authenticated-layout, header, main
│   │   │   └── data-table/     # TanStack Table với filter, sort, pagination
│   │   ├── features/
│   │   │   ├── dashboard/      # Dashboard page
│   │   │   ├── pull-requests/  # PR list, detail, review detail
│   │   │   ├── history/        # Review history
│   │   │   └── settings/       # Repo configs, CLI list
│   │   ├── routes/             # TanStack Router file-based routing
│   │   ├── stores/             # Zustand stores (app-store)
│   │   ├── hooks/              # use-ws, use-dialog-state, use-mobile
│   │   ├── lib/                # api client, utils
│   │   ├── context/            # theme, layout, font, search providers
│   │   └── config/             # fonts config
│   ├── package.json
│   └── vite.config.ts
├── ecosystem.config.cjs       # PM2 config
├── Makefile
└── config.yaml
```

## Data Models

### PullRequest
```go
type PullRequest struct {
    ID            uint       `gorm:"primaryKey"`
    GitHubID      string     `gorm:"uniqueIndex"`
    RepoFullName  string
    Title         string
    URL           string
    Number        int
    Author        string
    BaseBranch    string
    HeadBranch    string
    HeadSHA       string     // Latest commit SHA from GitHub
    WorktreePath  string     // e.g. "/tmp/pr-review-42"
    Status        string     // pending | reviewing | drafted | approved | posted | failed | closed
    ClosedAt      *time.Time
    CreatedAt     time.Time
}
```

### Review
```go
type Review struct {
    ID             uint      `gorm:"primaryKey"`
    PullRequestID  uint      `gorm:"index:idx_pr_commit,unique"`
    CommitSHA      string    `gorm:"index:idx_pr_commit,unique"` // Review gắn với 1 commit cụ thể
    Summary        string
    OverallVerdict string    // approve | request_changes | comment
    Status         string    // draft | approved | posted | rejected
    ExecutorName   string
    CreatedAt      time.Time
}
```

### ReviewComment
```go
type ReviewComment struct {
    ID        uint
    ReviewID  uint    `gorm:"index"`
    FilePath  string
    LineStart int
    LineEnd   int
    Body      string
    CreatedAt time.Time
}
```

### RepoConfig
```go
type RepoConfig struct {
    ID           uint    `gorm:"primaryKey"`
    RepoFullName string  `gorm:"uniqueIndex"`
    LocalPath    string  // Local path to cloned repo
    CLI          string  // Executor name: "claude-code", "codex"
    ExtraRules   *string // Additional prompt rules
    Active       bool
}
```

### CLIConfig
```go
type CLIConfig struct {
    ID          uint   `gorm:"primaryKey"`
    Name        string `gorm:"uniqueIndex"`
    Description string
    Active      bool
}
```

## CLI Executor Interface

```go
type Executor interface {
    Name() string
    GetReviewCommand(ctx context.Context, pr *PullRequest, rc *RepoConfig) (*ReviewCommand, error)
}

type ReviewCommand struct {
    Command       string            // CLI command to execute
    Prompt        string            // stdin prompt
    WorkingDir    string            // Worktree path
    InjectEnvVars map[string]string
    Timeout       time.Duration     // Default 60 minutes
}

type ReviewResult struct {
    Summary        string
    OverallVerdict string           // approve | request_changes | comment
    Comments       []CommentResult
}

type CommentResult struct {
    FilePath  string
    LineStart int
    LineEnd   int
    Body      string
}
```

## Worktree Management

- Worktree gắn với PullRequest, không gắn với Review
- Tạo worktree từ parent repo khi review lần đầu, reuse cho các lần review sau
- Cleanup bởi scheduled job mỗi ngày: xóa worktree của PR đã closed > 30 ngày

### Worktree Flow
```
1. PR.WorktreePath rỗng → git worktree add /tmp/pr-review-{id} --detach <head_sha>
2. PR.WorktreePath có sẵn → git -C <worktree> fetch && git -C <worktree> checkout <new_sha>
3. Chạy CLI executor với WorkingDir = PR.WorktreePath
4. Cleanup (daily job): PR closed > 30 ngày → git worktree remove --force <path>
```

## Repo Mapping Logic

1. Check RepoConfig trong DB trước (manual mapping)
2. Nếu không có → Auto-detect: scan các project root được config, check git remote match với repo full name
3. Nếu không tìm thấy → Skip PR, ghi warning log

## Asynq Tasks

| Task Type | Trigger | Description |
|-----------|---------|-------------|
| `github:fetch_assigned_prs` | Periodic 15 phút | Fetch PR open assigned to me |
| `github:sync_pr_status` | Periodic 15 phút | Check PRs trong DB đã closed chưa |
| `review:execute` | Enqueued bởi fetch khi có commit mới | Run CLI agent review |
| `review:post` | Enqueued khi user approve | Post review lên GitHub |
| `cleanup:worktree` | Daily lúc 2h sáng | Xóa worktree của PR closed > 30 ngày |

### Fetch Task Flow
```
1. GitHub API: is:pr is:open review-requested:@me
2. Mỗi PR → Upsert vào DB
3. So sánh HeadSHA với Review.CommitSHA mới nhất
   - Khác → Enqueue review:execute
   - Giống → Skip
4. Push WS: pr.updated
```

### Execute Review Flow
```
1. Set PR.Status = "reviewing"
2. Resolve RepoConfig (mapping → auto-detect)
3. Tạo/reuse worktree
4. Lấy executor từ Registry
5. Chạy CLI, timeout 60 phút, max retry 2
6. Parse stdout → ReviewResult
7. Tạo Review + ReviewComments trong DB
8. Set PR.Status = "drafted"
9. Push WS: review.created
```

### Sync PR Status Flow
```
1. Query DB: PR.Status != 'closed' AND ClosedAt IS NULL
2. Mỗi PR → GitHub API GET /repos/{owner}/{repo}/pulls/{number}
3. Nếu closed/merged → Set PR.ClosedAt, PR.Status = 'closed'
4. Push WS: pr.updated
```

## API Design

### Response Format
```json
// Success (single object)
{ "code": 0, "data": { "id": 1, "title": "Fix login bug" } }

// Success (list)
{ "code": 0, "data": { "items": [...], "meta": { "page": 1, "per_page": 20, "total": 42 } } }

// Error
{ "code": 4004, "error": "pull request not found" }
```

### Endpoints

```
GET    /api/prs                      # List PRs (filter: status, repo)
GET    /api/prs/:id                  # PR detail + reviews history
POST   /api/prs/:id/refresh          # Force re-fetch & review
GET    /api/prs/:id/reviews          # List reviews for PR
GET    /api/reviews/:id              # Review detail + comments
PUT    /api/reviews/:id              # Edit review draft
POST   /api/reviews/:id/approve      # Approve → post to GitHub
POST   /api/reviews/:id/reject       # Reject review
GET    /api/configs/repos            # List RepoConfigs
POST   /api/configs/repos            # Create RepoConfig
PUT    /api/configs/repos/:id        # Update RepoConfig
DELETE /api/configs/repos/:id        # Delete RepoConfig
POST   /api/configs/repos/:id/test   # Test connection
GET    /api/configs/clis             # List available CLI executors
GET    /api/dashboard                # Stats + recent activity
GET    /api/history                  # Paginated review history
GET    /api/system/health            # Health check
GET    /api/system/scheduler/state   # Scheduler status
WS     /ws                           # WebSocket (Centrifuge, no auth)
```

### Error Codes
| Code | Description |
|------|-------------|
| 0 | Success |
| 4000 | Invalid request |
| 4004 | Not found |
| 5001 | Executor timeout |
| 5002 | Executor failed |
| 5003 | GitHub API failed |
| 5004 | Worktree failed |
| 5005 | No repo config found |
| 5999 | Internal server error |

## WebSocket Events (no auth)

Backend publish events kèm payload, frontend nhận và cập nhật TanStack Query cache ngay:

| Event | Payload | Action |
|-------|---------|--------|
| `pr.updated` | PullRequest | `setQueryData` + invalidate |
| `review.created` | Review | `setQueryData` append vào review list |
| `review.posted` | Review | `setQueryData` update review |
| `scheduler.tick` | {lastRun, nextRun} | Invalidate dashboard + PRs |

## Frontend

Dựa trên [shadcn-admin](https://github.com/satnaing/shadcn-admin) template. Clone template về `web/`, strip các phần không cần, custom hóa cho app.

### Template features được giữ
- **Layout system**: `app-sidebar`, `authenticated-layout` (bỏ auth check), `header`, `main`, `nav-group`, `top-nav`
- **Theme**: Light/dark mode toggle (`theme-switch`, `theme-provider`)
- **UI components**: Toàn bộ `components/ui/` (shadcn primitives)
- **Data table**: `components/data-table/` với filter, sort, pagination, column visibility
- **Search**: `command-menu.tsx`, `search.tsx`
- **Providers**: `theme-provider`, `layout-provider`, `font-provider`, `search-provider`
- **Hooks**: `use-dialog-state`, `use-mobile`, `use-table-url-state`
- **Error pages**: 401, 403, 404, 500

### Template features bị xóa
- Clerk authentication (provider, routes, `auth-store.ts`)
- Auth pages: sign-in, sign-up, forgot-password, OTP
- `clerk/` route group
- `team-switcher.tsx`, `nav-user.tsx`, `profile-dropdown.tsx`, `sign-out-dialog.tsx`
- Feature pages cũ: users, tasks, chats, apps, help-center

### Thêm mới
- **TanStack Query**: `@tanstack/react-query` để fetch/cache server state
- **Centrifuge client**: `centrifuge` package, WebSocket hook trong `hooks/use-ws.ts`
- **API client**: `lib/api.ts` — `request<T>()` wrapper quanh `fetch`, gọi `/api/*`
- **App store**: `stores/app-store.ts` — sidebar state, filter selections

### Routes
```
/                          → Dashboard (stats + recent reviews)
/prs                       → PR list (filterable by status, repo)
/prs/$prId                 → PR detail (info + diff link + review history)
/prs/$prId/reviews/$reviewId → Review detail (edit, approve, reject)
/history                   → Review history (paginated, filterable)
/settings                  → Redirect to /settings/repos
/settings/repos            → Repo config CRUD
/settings/clis             → Available CLI executors
```

### Sidebar Navigation
```
Dashboard
Pull Requests
Review History
Settings
  ├─ Repo Configs
  └─ CLI Executors
```

### Data Flow
- Server state → TanStack Query (fetch, cache, `setQueryData` from WS)
- UI state → Zustand `app-store` (sidebar open, filters)
- Real-time → Centrifuge WebSocket hook `use-ws`:
  - Nhận events: `pr.updated`, `review.created`, `review.posted`, `scheduler.tick`
  - `setQueryData` update cache ngay + `invalidateQueries` đồng bộ với server
- Table state → `use-table-url-state` sync filter/pagination lên URL params

### Dev Setup
```bash
cd web && pnpm install && pnpm dev
# Vite dev server trên :5173, proxy /api và /ws đến Go backend :8080
```

### Production Build
```bash
cd web && pnpm build
# Output: web/dist/ — được Go API binary embed qua //go:embed
```

## Production Build & PM2

### Build
```bash
make build
# → bin/api (Go binary with embedded frontend)
# → bin/worker (Go binary)
# → web/dist/ (React static build via pnpm build, embedded in api binary)
```

### PM2 Configuration
- `pr-reviewer-api`: serve API + static frontend on :8080, max restart 5
- `pr-reviewer-worker`: scheduler + executor, max restart 5, restart delay 10s

### Commands
```bash
make build && make start    # Build and start
make stop / make restart    # Stop / restart
make logs                   # pm2 logs
make status                 # pm2 status
```

## Local Dev

```bash
make dev-infra   # Start PostgreSQL & Redis (brew services)
make dev-api     # go run ./cmd/api
make dev-worker  # go run ./cmd/worker
make dev-web     # cd web && pnpm dev
```

No Docker - PostgreSQL và Redis chạy trực tiếp trên host.

## Error Handling

| Scenario | Handling |
|----------|----------|
| CLI timeout (60 phút) | Mark PR failed, log error, Asynq retry 2 lần |
| GitHub API rate limit | Exponential backoff, wait for reset |
| Worktree add failed | Retry with different path |
| Parse stdout failed | Save raw output, mark failed, show log in UI |
| No repo config | Skip PR, log warning |
| Executor crash | Asynq retry, then mark failed |
