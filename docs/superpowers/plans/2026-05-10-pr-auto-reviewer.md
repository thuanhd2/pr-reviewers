# PR Auto Reviewer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build web app that periodically fetches GitHub PRs assigned to user, auto-reviews them via CLI agents, and lets user approve drafts before posting.

**Architecture:** Monorepo with Go backend (Gin API + Asynq worker, shared packages) and React frontend (Vite + ShadcnUI + TanStack Query/Zustand/Router). PostgreSQL + Redis on host, no Docker. PM2 for production process management.

**Tech Stack:** Go, Gin, GORM, Asynq, Centrifuge, React, TypeScript, Vite, ShadcnUI, TanStack Query, Zustand, TanStack Router, TailwindCSS

---

### Task 1: Initialize Go module and directory structure

**Files:**
- Create: `go.mod`
- Create: `go.sum` (auto-generated)
- Create: `cmd/api/main.go` (stub)
- Create: `cmd/worker/main.go` (stub)

- [ ] **Step 1: Initialize Go module**

Run: `cd /Users/thuanho/Documents/personal/pr-reviewers && go mod init github.com/thuanho/pr-reviewers`
Expected: `go: creating new go.mod: module github.com/thuanho/pr-reviewers`

- [ ] **Step 2: Create directory structure**

Run:
```bash
mkdir -p cmd/api cmd/worker
mkdir -p internal/{executor,github,store,scheduler,task,ws,config,handler}
```

- [ ] **Step 3: Create API entry point stub**

Write `cmd/api/main.go`:
```go
package main

import "fmt"

func main() {
	fmt.Println("pr-reviewer-api starting...")
}
```

- [ ] **Step 4: Create Worker entry point stub**

Write `cmd/worker/main.go`:
```go
package main

import "fmt"

func main() {
	fmt.Println("pr-reviewer-worker starting...")
}
```

- [ ] **Step 5: Verify it compiles**

Run: `go build ./cmd/api && go build ./cmd/worker`
Expected: No errors, binaries produced.

---

### Task 2: Create config package

**Files:**
- Create: `internal/config/config.go`
- Create: `config.example.yaml`

- [ ] **Step 1: Write config package**

Write `internal/config/config.go`:
```go
package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GitHub    GitHubConfig    `yaml:"github"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Projects  []string        `yaml:"projects_root"`
	Executors []ExecutorDef   `yaml:"executors"`
	Repos     []RepoMapping   `yaml:"repo_mappings"`
}

type GitHubConfig struct {
	Token string `yaml:"token"`
}

type SchedulerConfig struct {
	FetchInterval            string `yaml:"fetch_interval"`
	CleanupWorktreeAfterDays int    `yaml:"cleanup_worktree_after_days"`
}

type ExecutorDef struct {
	Name    string `yaml:"name"`
	Active  bool   `yaml:"active"`
	Timeout string `yaml:"timeout"`
}

type RepoMapping struct {
	Repo       string  `yaml:"repo"`
	LocalPath  string  `yaml:"local_path"`
	CLI        string  `yaml:"cli"`
	ExtraRules *string `yaml:"extra_rules"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.GitHub.Token == "" {
		cfg.GitHub.Token = os.Getenv("GITHUB_TOKEN")
	}
	if cfg.Scheduler.FetchInterval == "" {
		cfg.Scheduler.FetchInterval = "15m"
	}
	if cfg.Scheduler.CleanupWorktreeAfterDays == 0 {
		cfg.Scheduler.CleanupWorktreeAfterDays = 30
	}
	return &cfg, nil
}

func (c *Config) FetchInterval() time.Duration {
	d, err := time.ParseDuration(c.Scheduler.FetchInterval)
	if err != nil {
		return 15 * time.Minute
	}
	return d
}
```

- [ ] **Step 2: Create example config**

Write `config.example.yaml`:
```yaml
github:
  token: "" # or use GITHUB_TOKEN env var

scheduler:
  fetch_interval: "15m"
  cleanup_worktree_after_days: 30

projects_root:
  - /Users/thuanho/projects
  - /Users/thuanho/Documents/personal

executors:
  - name: claude-code
    active: true
    timeout: "60m"

repo_mappings:
  - repo: company/backend-api
    local_path: /Users/thuanho/projects/backend-api
    cli: claude-code
    extra_rules: ""
```

- [ ] **Step 3: Add dependencies**

Run: `go get gopkg.in/yaml.v3`
Expected: Module added to go.mod.

- [ ] **Step 4: Verify compilation**

Run: `go build ./internal/config/...`
Expected: No errors.

---

### Task 3: Create store models

**Files:**
- Create: `internal/store/models.go`

- [ ] **Step 1: Write GORM models**

Write `internal/store/models.go`:
```go
package store

import (
	"time"

	"gorm.io/gorm"
)

type PullRequest struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	GitHubID     string     `json:"github_id" gorm:"uniqueIndex"`
	RepoFullName string     `json:"repo_full_name"`
	Title        string     `json:"title"`
	URL          string     `json:"url"`
	Number       int        `json:"number"`
	Author       string     `json:"author"`
	BaseBranch   string     `json:"base_branch"`
	HeadBranch   string     `json:"head_branch"`
	HeadSHA      string     `json:"head_sha"`
	WorktreePath string     `json:"worktree_path"`
	Status       string     `json:"status" gorm:"default:pending"`
	ClosedAt     *time.Time `json:"closed_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Reviews      []Review   `json:"reviews,omitempty" gorm:"foreignKey:PullRequestID"`
}

type Review struct {
	ID             uint            `json:"id" gorm:"primaryKey"`
	PullRequestID  uint            `json:"pull_request_id" gorm:"index:idx_pr_commit,unique"`
	CommitSHA      string          `json:"commit_sha" gorm:"index:idx_pr_commit,unique"`
	Summary        string          `json:"summary"`
	OverallVerdict string          `json:"overall_verdict"`
	Status         string          `json:"status" gorm:"default:draft"`
	ExecutorName   string          `json:"executor_name"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Comments       []ReviewComment `json:"comments,omitempty" gorm:"foreignKey:ReviewID"`
}

type ReviewComment struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ReviewID  uint      `json:"review_id" gorm:"index"`
	FilePath  string    `json:"file_path"`
	LineStart int       `json:"line_start"`
	LineEnd   int       `json:"line_end"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type RepoConfig struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	RepoFullName string `json:"repo_full_name" gorm:"uniqueIndex"`
	LocalPath    string `json:"local_path"`
	CLI          string `json:"cli"`
	ExtraRules   *string `json:"extra_rules"`
	Active       bool   `json:"active" gorm:"default:true"`
}

func (RepoConfig) TableName() string { return "repo_configs" }

type CLIConfig struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"uniqueIndex"`
	Description string `json:"description"`
	Active      bool   `json:"active" gorm:"default:true"`
}

func (CLIConfig) TableName() string { return "cli_configs" }
```

- [ ] **Step 2: Add GORM dependency**

Run: `go get gorm.io/gorm gorm.io/driver/postgres`
Expected: Modules added.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/store/...`
Expected: No errors.

---

### Task 4: Create store CRUD layer

**Files:**
- Create: `internal/store/store.go`

- [ ] **Step 1: Write Store struct with AutoMigrate and DB init**

Write `internal/store/store.go`:
```go
package store

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Store struct {
	db *gorm.DB
}

func New(dsn string) (*Store, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&PullRequest{}, &Review{}, &ReviewComment{}, &RepoConfig{}, &CLIConfig{}); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) DB() *gorm.DB { return s.db }
```

- [ ] **Step 2: Add PR CRUD methods to store.go**

Append to `internal/store/store.go`:
```go
func (s *Store) UpsertPR(pr *PullRequest) error {
	return s.db.Where("github_id = ?", pr.GitHubID).Assign(pr).FirstOrCreate(pr).Error
}

func (s *Store) GetPR(id uint) (*PullRequest, error) {
	var pr PullRequest
	err := s.db.Preload("Reviews", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC")
	}).Preload("Reviews.Comments").First(&pr, id).Error
	return &pr, err
}

func (s *Store) ListPRs(status string, repo string, page, perPage int) ([]PullRequest, int64, error) {
	var prs []PullRequest
	var total int64
	q := s.db.Model(&PullRequest{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if repo != "" {
		q = q.Where("repo_full_name = ?", repo)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&prs).Error
	return prs, total, err
}

func (s *Store) ListOpenPRs() ([]PullRequest, error) {
	var prs []PullRequest
	err := s.db.Where("closed_at IS NULL").Find(&prs).Error
	return prs, err
}

func (s *Store) UpdatePRStatus(id uint, status string) error {
	return s.db.Model(&PullRequest{}).Where("id = ?", id).Update("status", status).Error
}

func (s *Store) UpdatePRWorktree(id uint, worktreePath string) error {
	return s.db.Model(&PullRequest{}).Where("id = ?", id).Update("worktree_path", worktreePath).Error
}

func (s *Store) MarkPRClosed(id uint) error {
	return s.db.Model(&PullRequest{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":    "closed",
		"closed_at": gorm.Expr("NOW()"),
	}).Error
}

func (s *Store) ListPRsForWorktreeCleanup(days int) ([]PullRequest, error) {
	var prs []PullRequest
	err := s.db.Where("closed_at IS NOT NULL AND worktree_path != '' AND closed_at < NOW() - INTERVAL '1 day' * ?", days).Find(&prs).Error
	return prs, err
}
```

- [ ] **Step 3: Add Review CRUD methods**

Append to `internal/store/store.go`:
```go
func (s *Store) CreateReview(review *Review) error {
	return s.db.Create(review).Error
}

func (s *Store) GetReview(id uint) (*Review, error) {
	var review Review
	err := s.db.Preload("Comments").First(&review, id).Error
	return &review, err
}

func (s *Store) GetLatestReviewForPR(prID uint) (*Review, error) {
	var review Review
	err := s.db.Where("pull_request_id = ?", prID).Order("created_at DESC").First(&review).Error
	return &review, err
}

func (s *Store) ListReviewsForPR(prID uint) ([]Review, error) {
	var reviews []Review
	err := s.db.Where("pull_request_id = ?", prID).Preload("Comments").Order("created_at DESC").Find(&reviews).Error
	return reviews, err
}

func (s *Store) UpdateReview(id uint, updates map[string]interface{}) error {
	return s.db.Model(&Review{}).Where("id = ?", id).Updates(updates).Error
}

func (s *Store) CreateComment(comment *ReviewComment) error {
	return s.db.Create(comment).Error
}

func (s *Store) UpdateComment(id uint, body string) error {
	return s.db.Model(&ReviewComment{}).Where("id = ?", id).Update("body", body).Error
}

func (s *Store) DeleteComment(id uint) error {
	return s.db.Delete(&ReviewComment{}, id).Error
}

func (s *Store) ListHistory(page, perPage int, repo string) ([]Review, int64, error) {
	var reviews []Review
	var total int64
	q := s.db.Model(&Review{}).Joins("JOIN pull_requests ON pull_requests.id = reviews.pull_request_id")
	if repo != "" {
		q = q.Where("pull_requests.repo_full_name = ?", repo)
	}
	q.Count(&total)
	err := q.Preload("Comments").Order("reviews.created_at DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&reviews).Error
	return reviews, total, err
}
```

- [ ] **Step 4: Add RepoConfig CRUD methods**

Append to `internal/store/store.go`:
```go
func (s *Store) ListRepoConfigs() ([]RepoConfig, error) {
	var configs []RepoConfig
	err := s.db.Find(&configs).Error
	return configs, err
}

func (s *Store) GetRepoConfig(repoFullName string) (*RepoConfig, error) {
	var cfg RepoConfig
	err := s.db.Where("repo_full_name = ?", repoFullName).First(&cfg).Error
	return &cfg, err
}

func (s *Store) CreateRepoConfig(cfg *RepoConfig) error {
	return s.db.Create(cfg).Error
}

func (s *Store) UpdateRepoConfig(id uint, cfg *RepoConfig) error {
	return s.db.Model(&RepoConfig{}).Where("id = ?", id).Updates(cfg).Error
}

func (s *Store) DeleteRepoConfig(id uint) error {
	return s.db.Delete(&RepoConfig{}, id).Error
}
```

- [ ] **Step 5: Add CLIConfig methods**

Append to `internal/store/store.go`:
```go
func (s *Store) ListCLIConfigs() ([]CLIConfig, error) {
	var configs []CLIConfig
	err := s.db.Find(&configs).Error
	return configs, err
}

func (s *Store) SeedCLIConfigs() error {
	configs := []CLIConfig{
		{Name: "claude-code", Description: "Claude Code CLI agent", Active: true},
	}
	for _, c := range configs {
		s.db.Where("name = ?", c.Name).FirstOrCreate(&c)
	}
	return nil
}
```

- [ ] **Step 6: Verify compilation**

Run: `go build ./internal/store/...`
Expected: No errors.

---

### Task 5: Create GitHub client

**Files:**
- Create: `internal/github/client.go`

- [ ] **Step 1: Write GitHub client**

Write `internal/github/client.go`:
```go
package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	token      string
	httpClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type SearchPR struct {
	NodeID  string `json:"node_id"`
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
	Number  int    `json:"number"`
	User    struct {
		Login string `json:"login"`
	} `json:"user"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
	Head struct {
		Ref  string `json:"ref"`
		SHA  string `json:"sha"`
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"head"`
}

type searchResponse struct {
	TotalCount int        `json:"total_count"`
	Items      []SearchPR `json:"items"`
}

func (c *Client) SearchAssignedPRs() ([]SearchPR, error) {
	query := "is:pr is:open review-requested:@me"
	url := fmt.Sprintf("https://api.github.com/search/issues?q=%s&per_page=100", query)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github search returned %d", resp.StatusCode)
	}

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	return result.Items, nil
}

type PRDetail struct {
	State  string `json:"state"`
	Merged bool   `json:"merged"`
	Head   struct {
		SHA string `json:"sha"`
	} `json:"head"`
}

func (c *Client) GetPR(repoFullName string, number int) (*PRDetail, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d", repoFullName, number)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("PR %s#%d not found", repoFullName, number)
	}

	var pr PRDetail
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, err
	}
	return &pr, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/github/...`
Expected: No errors.

---

### Task 6: Create executor interface, registry, and result types

**Files:**
- Create: `internal/executor/executor.go`
- Create: `internal/executor/registry.go`

- [ ] **Step 1: Write executor interface and types**

Write `internal/executor/executor.go`:
```go
package executor

import (
	"context"
	"time"

	"github.com/thuanho/pr-reviewers/internal/store"
)

type Executor interface {
	Name() string
	GetReviewCommand(ctx context.Context, pr *store.PullRequest, rc *store.RepoConfig) (*ReviewCommand, error)
}

type ReviewCommand struct {
	Command       string
	Prompt        string
	WorkingDir    string
	InjectEnvVars map[string]string
	Timeout       time.Duration
}

type ReviewResult struct {
	Summary        string          `json:"summary"`
	OverallVerdict string          `json:"overall_verdict"`
	Comments       []CommentResult `json:"comments"`
}

type CommentResult struct {
	FilePath  string `json:"file_path"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	Body      string `json:"body"`
}
```

- [ ] **Step 2: Write registry**

Write `internal/executor/registry.go`:
```go
package executor

import "fmt"

type Registry struct {
	executors map[string]Executor
}

func NewRegistry() *Registry {
	return &Registry{executors: make(map[string]Executor)}
}

func (r *Registry) Register(exec Executor) {
	r.executors[exec.Name()] = exec
}

func (r *Registry) Get(name string) (Executor, error) {
	exec, ok := r.executors[name]
	if !ok {
		return nil, fmt.Errorf("executor %q not found", name)
	}
	return exec, nil
}

func (r *Registry) List() []string {
	names := make([]string, 0, len(r.executors))
	for name := range r.executors {
		names = append(names, name)
	}
	return names
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/executor/...`
Expected: No errors.

---

### Task 7: Create Claude Code executor

**Files:**
- Create: `internal/executor/claude_code.go`

- [ ] **Step 1: Write ClaudeCode executor**

Write `internal/executor/claude_code.go`:
```go
package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/thuanho/pr-reviewers/internal/store"
)

type ClaudeCodeExecutor struct {
	timeout time.Duration
}

func NewClaudeCodeExecutor(timeout time.Duration) *ClaudeCodeExecutor {
	if timeout == 0 {
		timeout = 60 * time.Minute
	}
	return &ClaudeCodeExecutor{timeout: timeout}
}

func (e *ClaudeCodeExecutor) Name() string { return "claude-code" }

func (e *ClaudeCodeExecutor) GetReviewCommand(ctx context.Context, pr *store.PullRequest, rc *store.RepoConfig) (*ReviewCommand, error) {
	prompt := fmt.Sprintf("/review %s", pr.URL)
	if rc.ExtraRules != nil && *rc.ExtraRules != "" {
		prompt += fmt.Sprintf("\n\nAdditional rules: %s", *rc.ExtraRules)
	}

	return &ReviewCommand{
		Command:    "npx -y @anthropic-ai/claude-code@latest -p --dangerously-skip-permissions --output-format json",
		Prompt:     prompt,
		WorkingDir: pr.WorktreePath,
		InjectEnvVars: map[string]string{},
		Timeout:    e.timeout,
	}, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/executor/...`
Expected: No errors.

---

### Task 8: Create Codex executor (stub for extensibility)

**Files:**
- Create: `internal/executor/codex.go`

- [ ] **Step 1: Write Codex executor stub**

Write `internal/executor/codex.go`:
```go
package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/thuanho/pr-reviewers/internal/store"
)

type CodexExecutor struct {
	timeout time.Duration
}

func NewCodexExecutor(timeout time.Duration) *CodexExecutor {
	if timeout == 0 {
		timeout = 60 * time.Minute
	}
	return &CodexExecutor{timeout: timeout}
}

func (e *CodexExecutor) Name() string { return "codex" }

func (e *CodexExecutor) GetReviewCommand(ctx context.Context, pr *store.PullRequest, rc *store.RepoConfig) (*ReviewCommand, error) {
	prompt := fmt.Sprintf("Review this PR: %s", pr.URL)
	if rc.ExtraRules != nil && *rc.ExtraRules != "" {
		prompt += fmt.Sprintf("\n\nAdditional rules: %s", *rc.ExtraRules)
	}

	return &ReviewCommand{
		Command:    "codex",
		Prompt:     prompt,
		WorkingDir: pr.WorktreePath,
		InjectEnvVars: map[string]string{},
		Timeout:    e.timeout,
	}, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/executor/...`
Expected: No errors.

---

### Task 9: Create WebSocket package

**Files:**
- Create: `internal/ws/ws.go`

- [ ] **Step 1: Write Centrifuge WebSocket setup**

Write `internal/ws/ws.go`:
```go
package ws

import (
	"net/http"

	"github.com/centrifugal/centrifuge"
)

type Hub struct {
	Node *centrifuge.Node
}

func NewHub() (*Hub, error) {
	node, err := centrifuge.New(centrifuge.Config{})
	if err != nil {
		return nil, err
	}

	node.OnConnecting(func(ctx context.Context, e centrifuge.ConnectEvent) (centrifuge.ConnectReply, error) {
		return centrifuge.ConnectReply{}, nil
	})

	if err := node.Run(); err != nil {
		return nil, err
	}

	return &Hub{Node: node}, nil
}

func (h *Hub) Handler() http.Handler {
	return centrifuge.NewWebsocketHandler(h.Node, centrifuge.WebsocketConfig{})
}

func (h *Hub) Publish(channel string, data any) error {
	_, err := h.Node.Publish(channel, data)
	return err
}
```

- [ ] **Step 2: Add Centrifuge dependency**

Run: `go get github.com/centrifugal/centrifuge`
Expected: Module added.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/ws/...`
Expected: No errors.

---

### Task 10: Create Asynq tasks - fetch PRs

**Files:**
- Create: `internal/task/fetch_prs.go`

- [ ] **Step 1: Write fetch PRs task handler**

Write `internal/task/fetch_prs.go`:
```go
package task

import (
	"context"
	"encoding/json"
	"log"

	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/github"
	"github.com/thuanho/pr-reviewers/internal/store"
	"github.com/thuanho/pr-reviewers/internal/ws"
)

const TypeFetchAssignedPRs = "github:fetch_assigned_prs"

type FetchPRsHandler struct {
	store     *store.Store
	ghClient  *github.Client
	wsHub     *ws.Hub
	asynqClient *asynq.Client
}

func NewFetchPRsHandler(s *store.Store, gh *github.Client, hub *ws.Hub, ac *asynq.Client) *FetchPRsHandler {
	return &FetchPRsHandler{store: s, ghClient: gh, wsHub: hub, asynqClient: ac}
}

func (h *FetchPRsHandler) Handle(ctx context.Context, t *asynq.Task) error {
	searchPRs, err := h.ghClient.SearchAssignedPRs()
	if err != nil {
		return err
	}

	for _, sp := range searchPRs {
		pr := &store.PullRequest{
			GitHubID:     sp.NodeID,
			RepoFullName: sp.Head.Repo.FullName,
			Title:        sp.Title,
			URL:          sp.HTMLURL,
			Number:       sp.Number,
			Author:       sp.User.Login,
			BaseBranch:   sp.Base.Ref,
			HeadBranch:   sp.Head.Ref,
			HeadSHA:      sp.Head.SHA,
			Status:       "pending",
		}

		if err := h.store.UpsertPR(pr); err != nil {
			log.Printf("upsert PR %s: %v", sp.NodeID, err)
			continue
		}

		latestReview, _ := h.store.GetLatestReviewForPR(pr.ID)
		if latestReview == nil || latestReview.CommitSHA != sp.Head.SHA {
			payload, _ := json.Marshal(map[string]uint{"pr_id": pr.ID})
			h.asynqClient.Enqueue(asynq.NewTask(TypeExecuteReview, payload))
		}

		h.wsHub.Publish("pr-updates", map[string]any{
			"type":    "pr.updated",
			"payload": pr,
		})
	}

	return nil
}
```

- [ ] **Step 2: Add Asynq dependency**

Run: `go get github.com/hibiken/asynq`
Expected: Module added.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/task/...`
Expected: No errors.

---

### Task 11: Create Asynq tasks - sync PR status

**Files:**
- Create: `internal/task/sync_pr_status.go`

- [ ] **Step 1: Write sync PR status handler**

Write `internal/task/sync_pr_status.go`:
```go
package task

import (
	"context"
	"log"

	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/github"
	"github.com/thuanho/pr-reviewers/internal/store"
	"github.com/thuanho/pr-reviewers/internal/ws"
)

const TypeSyncPRStatus = "github:sync_pr_status"

type SyncPRStatusHandler struct {
	store    *store.Store
	ghClient *github.Client
	wsHub    *ws.Hub
}

func NewSyncPRStatusHandler(s *store.Store, gh *github.Client, hub *ws.Hub) *SyncPRStatusHandler {
	return &SyncPRStatusHandler{store: s, ghClient: gh, wsHub: hub}
}

func (h *SyncPRStatusHandler) Handle(ctx context.Context, t *asynq.Task) error {
	prs, err := h.store.ListOpenPRs()
	if err != nil {
		return err
	}

	for _, pr := range prs {
		detail, err := h.ghClient.GetPR(pr.RepoFullName, pr.Number)
		if err != nil {
			log.Printf("get PR %s#%d: %v", pr.RepoFullName, pr.Number, err)
			continue
		}

		if detail.State == "closed" || (detail.State == "merged") {
			if err := h.store.MarkPRClosed(pr.ID); err != nil {
				log.Printf("mark PR %d closed: %v", pr.ID, err)
			}

			h.wsHub.Publish("pr-updates", map[string]any{
				"type":    "pr.updated",
				"payload": map[string]any{"id": pr.ID, "status": "closed"},
			})
		}
	}

	return nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/task/...`
Expected: No errors.

---

### Task 12: Create Asynq tasks - execute review

**Files:**
- Create: `internal/task/execute_review.go`

- [ ] **Step 1: Write execute review handler (worktree + CLI run + parse)**

Write `internal/task/execute_review.go`:
```go
package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/executor"
	"github.com/thuanho/pr-reviewers/internal/store"
	"github.com/thuanho/pr-reviewers/internal/ws"
)

const TypeExecuteReview = "review:execute"

type ExecuteReviewHandler struct {
	store    *store.Store
	registry *executor.Registry
	wsHub    *ws.Hub
}

func NewExecuteReviewHandler(s *store.Store, reg *executor.Registry, hub *ws.Hub) *ExecuteReviewHandler {
	return &ExecuteReviewHandler{store: s, registry: reg, wsHub: hub}
}

func (h *ExecuteReviewHandler) Handle(ctx context.Context, t *asynq.Task) error {
	var payload struct {
		PRID uint `json:"pr_id"`
	}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	pr, err := h.store.GetPR(payload.PRID)
	if err != nil {
		return fmt.Errorf("get PR %d: %w", payload.PRID, err)
	}

	h.store.UpdatePRStatus(pr.ID, "reviewing")

	rc, err := h.resolveRepoConfig(pr.RepoFullName)
	if err != nil {
		log.Printf("no repo config for %s: %v", pr.RepoFullName, err)
		h.store.UpdatePRStatus(pr.ID, "failed")
		return nil
	}

	exec, err := h.registry.Get(rc.CLI)
	if err != nil {
		log.Printf("executor %q not found: %v", rc.CLI, err)
		h.store.UpdatePRStatus(pr.ID, "failed")
		return nil
	}

	worktreePath, err := h.ensureWorktree(pr, rc)
	if err != nil {
		log.Printf("worktree for PR %d: %v", pr.ID, err)
		h.store.UpdatePRStatus(pr.ID, "failed")
		return nil
	}
	pr.WorktreePath = worktreePath

	cmd, err := exec.GetReviewCommand(ctx, pr, rc)
	if err != nil {
		return fmt.Errorf("get review command: %w", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	c := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("echo '%s' | %s", cmd.Prompt, cmd.Command))
	c.Dir = cmd.WorkingDir
	for k, v := range cmd.InjectEnvVars {
		c.Env = append(c.Env, fmt.Sprintf("%s=%s", k, v))
	}
	c.Stdout = &stdout
	c.Stderr = &stderr

	if err := c.Run(); err != nil {
		log.Printf("executor run: %v, stderr: %s", err, stderr.String())
		h.store.UpdatePRStatus(pr.ID, "failed")
		return nil
	}

	result, err := parseReviewResult(stdout.String())
	if err != nil {
		log.Printf("parse review result: %v, raw: %s", err, stdout.String())
		h.store.UpdatePRStatus(pr.ID, "failed")
		return nil
	}

	review := &store.Review{
		PullRequestID:  pr.ID,
		CommitSHA:      pr.HeadSHA,
		Summary:        result.Summary,
		OverallVerdict: result.OverallVerdict,
		Status:         "draft",
		ExecutorName:   exec.Name(),
	}
	if err := h.store.CreateReview(review); err != nil {
		return fmt.Errorf("create review: %w", err)
	}

	for _, c := range result.Comments {
		comment := &store.ReviewComment{
			ReviewID:  review.ID,
			FilePath:  c.FilePath,
			LineStart: c.LineStart,
			LineEnd:   c.LineEnd,
			Body:      c.Body,
		}
		h.store.CreateComment(comment)
	}

	h.store.UpdatePRStatus(pr.ID, "drafted")

	h.wsHub.Publish("pr-updates", map[string]any{
		"type":    "review.created",
		"payload": review,
	})

	return nil
}

func (h *ExecuteReviewHandler) resolveRepoConfig(repoFullName string) (*store.RepoConfig, error) {
	rc, err := h.store.GetRepoConfig(repoFullName)
	if err == nil {
		return rc, nil
	}
	// TODO: auto-detect from projects_root config + git remote matching
	return nil, fmt.Errorf("no repo config for %s", repoFullName)
}

func (h *ExecuteReviewHandler) ensureWorktree(pr *store.PullRequest, rc *store.RepoConfig) (string, error) {
	if pr.WorktreePath != "" {
		if _, err := os.Stat(pr.WorktreePath); err == nil {
			c := exec.Command("git", "-C", pr.WorktreePath, "fetch", "origin")
			c.Run()
			c = exec.Command("git", "-C", pr.WorktreePath, "checkout", pr.HeadSHA)
			c.Run()
			return pr.WorktreePath, nil
		}
	}

	worktreePath := fmt.Sprintf("/tmp/pr-review-%d", pr.ID)
	c := exec.Command("git", "-C", rc.LocalPath, "worktree", "add", "--detach", worktreePath, pr.HeadSHA)
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("worktree add: %w", err)
	}

	h.store.UpdatePRWorktree(pr.ID, worktreePath)
	return worktreePath, nil
}

func parseReviewResult(stdout string) (*executor.ReviewResult, error) {
	var result executor.ReviewResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		// Fallback: treat entire stdout as summary
		return &executor.ReviewResult{
			Summary:        strings.TrimSpace(stdout),
			OverallVerdict: "comment",
		}, nil
	}
	return &result, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/task/...`
Expected: No errors.

---

### Task 13: Create Asynq tasks - post review

**Files:**
- Create: `internal/task/post_review.go`

- [ ] **Step 1: Write post review handler**

Write `internal/task/post_review.go`:
```go
package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/store"
	"github.com/thuanho/pr-reviewers/internal/ws"
)

const TypePostReview = "review:post"

type PostReviewHandler struct {
	store     *store.Store
	ghToken   string
	wsHub     *ws.Hub
}

func NewPostReviewHandler(s *store.Store, ghToken string, hub *ws.Hub) *PostReviewHandler {
	return &PostReviewHandler{store: s, ghToken: ghToken, wsHub: hub}
}

func (h *PostReviewHandler) Handle(ctx context.Context, t *asynq.Task) error {
	var payload struct {
		ReviewID uint `json:"review_id"`
	}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	review, err := h.store.GetReview(payload.ReviewID)
	if err != nil {
		return fmt.Errorf("get review %d: %w", payload.ReviewID, err)
	}

	pr, err := h.store.GetPR(review.PullRequestID)
	if err != nil {
		return fmt.Errorf("get PR %d: %w", review.PullRequestID, err)
	}

	parts := splitRepo(pr.RepoFullName)
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d/reviews", parts[0], parts[1], pr.Number)

	body := map[string]interface{}{
		"body":   review.Summary,
		"event":  review.OverallVerdict,
		"commit_id": pr.HeadSHA,
		"comments": make([]map[string]interface{}, 0),
	}

	for _, c := range review.Comments {
		body["comments"] = append(body["comments"].([]map[string]interface{}), map[string]interface{}{
			"path":       c.FilePath,
			"line":       c.LineEnd,
			"start_line": c.LineStart,
			"side":       "RIGHT",
			"body":       c.Body,
		})
	}

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	req.Header.Set("Authorization", "Bearer "+h.ghToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("post review: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("post review returned %d", resp.StatusCode)
	}

	h.store.UpdateReview(review.ID, map[string]interface{}{"status": "posted"})
	h.store.UpdatePRStatus(pr.ID, "posted")

	h.wsHub.Publish("pr-updates", map[string]any{
		"type":    "review.posted",
		"payload": review,
	})

	return nil
}

func splitRepo(fullName string) [2]string {
	parts := [2]string{}
	for i, p := range split(fullName, "/") {
		if i < 2 {
			parts[i] = p
		}
	}
	return parts
}

func split(s, sep string) []string {
	var parts []string
	for {
		idx := indexOf(s, sep)
		if idx == -1 {
			parts = append(parts, s)
			break
		}
		parts = append(parts, s[:idx])
		s = s[idx+1:]
	}
	return parts
}

func indexOf(s, sep string) int {
	for i := 0; i < len(s)-len(sep)+1; i++ {
		if s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 2: Simplify - use strings.Split instead**

Replace the custom split functions. Change the splitRepo function:
```go
func splitRepo(fullName string) [2]string {
	parts := strings.SplitN(fullName, "/", 2)
	var result [2]string
	if len(parts) > 0 {
		result[0] = parts[0]
	}
	if len(parts) > 1 {
		result[1] = parts[1]
	}
	return result
}
```
And add `"strings"` to imports. Remove `split` and `indexOf` functions.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/task/...`
Expected: No errors.

---

### Task 14: Create Asynq tasks - cleanup worktree

**Files:**
- Create: `internal/task/cleanup_worktree.go`

- [ ] **Step 1: Write cleanup worktree handler**

Write `internal/task/cleanup_worktree.go`:
```go
package task

import (
	"context"
	"log"
	"os/exec"

	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/store"
)

const TypeCleanupWorktree = "cleanup:worktree"

type CleanupWorktreeHandler struct {
	store *store.Store
	days  int
}

func NewCleanupWorktreeHandler(s *store.Store, days int) *CleanupWorktreeHandler {
	return &CleanupWorktreeHandler{store: s, days: days}
}

func (h *CleanupWorktreeHandler) Handle(ctx context.Context, t *asynq.Task) error {
	prs, err := h.store.ListPRsForWorktreeCleanup(h.days)
	if err != nil {
		return err
	}

	for _, pr := range prs {
		// Find parent repo path from worktree
		c := exec.Command("git", "-C", pr.WorktreePath, "rev-parse", "--git-common-dir")
		out, err := c.Output()
		if err != nil {
			log.Printf("get parent for worktree %s: %v", pr.WorktreePath, err)
			continue
		}
		parentDir := string(out)
		parentDir = parentDir[:len(parentDir)-1] // trim newline
		// parentDir is like /path/to/repo/.git/worktrees/pr-review-42
		// We need the main repo path
		repoPath := parentDir[:len(parentDir)-len("/.git/worktrees/pr-review-"+itoa(int(pr.ID)))]
		
		cmd := exec.Command("git", "-C", repoPath, "worktree", "remove", "--force", pr.WorktreePath)
		if err := cmd.Run(); err != nil {
			log.Printf("remove worktree %s: %v", pr.WorktreePath, err)
			continue
		}

		h.store.UpdatePRWorktree(pr.ID, "")
		log.Printf("cleaned up worktree for PR %d: %s", pr.ID, pr.WorktreePath)
	}

	return nil
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
```

Wait, this is getting complex with parent directory resolution. Let me simplify. The worktree path is `/tmp/pr-review-{id}`, and we need the main repo to run `git worktree remove`. Let me use a simpler approach: just remove the directory and let git prune handle it, or store the parent repo path.

Actually, let me take a cleaner approach - store the parent repo path relationship:

- [ ] **Step 1 (rewritten): Write cleanup worktree handler**

Write `internal/task/cleanup_worktree.go`:
```go
package task

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/store"
)

const TypeCleanupWorktree = "cleanup:worktree"

type CleanupWorktreeHandler struct {
	store     *store.Store
	days      int
}

func NewCleanupWorktreeHandler(s *store.Store, days int) *CleanupWorktreeHandler {
	return &CleanupWorktreeHandler{store: s, days: days}
}

func (h *CleanupWorktreeHandler) Handle(ctx context.Context, t *asynq.Task) error {
	prs, err := h.store.ListPRsForWorktreeCleanup(h.days)
	if err != nil {
		return err
	}

	for _, pr := range prs {
		// Get parent repo from RepoConfig
		rc, err := h.store.GetRepoConfig(pr.RepoFullName)
		if err != nil {
			log.Printf("no repo config for %s, skipping worktree cleanup", pr.RepoFullName)
			continue
		}

		cmd := exec.Command("git", "-C", rc.LocalPath, "worktree", "remove", "--force", pr.WorktreePath)
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Printf("remove worktree %s: %v (out: %s)", pr.WorktreePath, err, out)
			// Try force remove directory
			os.RemoveAll(pr.WorktreePath)
		}

		h.store.UpdatePRWorktree(pr.ID, "")
		log.Printf("cleaned up worktree for PR %d: %s", pr.ID, pr.WorktreePath)
		fmt.Println("done") // prevent unused import
	}

	return nil
}

```

Hmm, that `fmt.Println("done")` is ugly. Let me remove the `fmt` import by removing the unused reference.

Actually let me just write it clean once:

- [ ] **Step 1 (final): Write cleanup worktree handler**

Write `internal/task/cleanup_worktree.go`:
```go
package task

import (
	"context"
	"log"
	"os"
	"os/exec"

	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/store"
)

const TypeCleanupWorktree = "cleanup:worktree"

type CleanupWorktreeHandler struct {
	store *store.Store
	days  int
}

func NewCleanupWorktreeHandler(s *store.Store, days int) *CleanupWorktreeHandler {
	return &CleanupWorktreeHandler{store: s, days: days}
}

func (h *CleanupWorktreeHandler) Handle(ctx context.Context, t *asynq.Task) error {
	prs, err := h.store.ListPRsForWorktreeCleanup(h.days)
	if err != nil {
		return err
	}

	for _, pr := range prs {
		rc, err := h.store.GetRepoConfig(pr.RepoFullName)
		if err != nil {
			log.Printf("no repo config for %s, force removing worktree dir", pr.RepoFullName)
			os.RemoveAll(pr.WorktreePath)
			h.store.UpdatePRWorktree(pr.ID, "")
			continue
		}

		cmd := exec.Command("git", "-C", rc.LocalPath, "worktree", "remove", "--force", pr.WorktreePath)
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Printf("remove worktree %s: %v (out: %s)", pr.WorktreePath, err, out)
			os.RemoveAll(pr.WorktreePath)
		}

		h.store.UpdatePRWorktree(pr.ID, "")
		log.Printf("cleaned up worktree for PR %d: %s", pr.ID, pr.WorktreePath)
	}

	return nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/task/...`
Expected: No errors.

---

### Task 15: Create scheduler

**Files:**
- Create: `internal/scheduler/scheduler.go`

- [ ] **Step 1: Write scheduler**

Write `internal/scheduler/scheduler.go`:
```go
package scheduler

import (
	"time"

	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/task"
)

func Register(scheduler *asynq.Scheduler, fetchInterval time.Duration, cleanupDays int) {
	scheduler.Register(fetchInterval, asynq.NewTask(task.TypeFetchAssignedPRs, nil))
	scheduler.Register(fetchInterval, asynq.NewTask(task.TypeSyncPRStatus, nil))

	// Daily at 2am
	daily := 24 * time.Hour
	// Schedule first run at next 2am
	now := time.Now()
	next2AM := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
	if next2AM.Before(now) {
		next2AM = next2AM.Add(24 * time.Hour)
	}
	initialDelay := next2AM.Sub(now)
	scheduler.Register(daily, asynq.NewTask(task.TypeCleanupWorktree, nil), asynq.StartingAt(next2AM))
	_ = initialDelay
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/scheduler/...`
Expected: No errors.

---

### Task 16: Create API response helpers

**Files:**
- Create: `internal/handler/response.go`

- [ ] **Step 1: Write response helper**

Write `internal/handler/response.go`:
```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code  int    `json:"code"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

type ListMeta struct {
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Total   int64 `json:"total"`
}

type ListData struct {
	Items any      `json:"items"`
	Meta  ListMeta `json:"meta"`
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Code: 0, Data: data})
}

func SuccessList(c *gin.Context, items any, meta ListMeta) {
	c.JSON(http.StatusOK, Response{
		Code: 0,
		Data: ListData{Items: items, Meta: meta},
	})
}

func Error(c *gin.Context, status int, code int, msg string) {
	c.JSON(status, Response{Code: code, Error: msg})
}

func GetPageAndPerPage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return page, perPage
}
```

- [ ] **Step 2: Add Gin dependency**

Run: `go get github.com/gin-gonic/gin`
Expected: Module added.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/handler/...`
Expected: No errors.

---

### Task 17: Create PR handlers

**Files:**
- Create: `internal/handler/prs.go`

- [ ] **Step 1: Write PR handlers**

Write `internal/handler/prs.go`:
```go
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/store"
	"github.com/thuanho/pr-reviewers/internal/task"
)

type PRHandler struct {
	store       *store.Store
	asynqClient *asynq.Client
}

func NewPRHandler(s *store.Store, ac *asynq.Client) *PRHandler {
	return &PRHandler{store: s, asynqClient: ac}
}

func (h *PRHandler) List(c *gin.Context) {
	page, perPage := GetPageAndPerPage(c)
	status := c.Query("status")
	repo := c.Query("repo")

	prs, total, err := h.store.ListPRs(status, repo, page, perPage)
	if err != nil {
		Error(c, http.StatusInternalServerError, 5999, "failed to list PRs")
		return
	}

	SuccessList(c, prs, ListMeta{Page: page, PerPage: perPage, Total: total})
}

func (h *PRHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	pr, err := h.store.GetPR(uint(id))
	if err != nil {
		Error(c, http.StatusNotFound, 4004, "pull request not found")
		return
	}
	Success(c, pr)
}

func (h *PRHandler) Refresh(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	pr, err := h.store.GetPR(uint(id))
	if err != nil {
		Error(c, http.StatusNotFound, 4004, "pull request not found")
		return
	}

	payload, _ := json.Marshal(map[string]uint{"pr_id": pr.ID})
	h.asynqClient.Enqueue(asynq.NewTask(task.TypeExecuteReview, payload))

	Success(c, map[string]string{"status": "review queued"})
}

func (h *PRHandler) ListReviews(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	reviews, err := h.store.ListReviewsForPR(uint(id))
	if err != nil {
		Error(c, http.StatusInternalServerError, 5999, "failed to list reviews")
		return
	}
	Success(c, reviews)
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/handler/...`
Expected: No errors.

---

### Task 18: Create Review handlers

**Files:**
- Create: `internal/handler/reviews.go`

- [ ] **Step 1: Write Review handlers**

Write `internal/handler/reviews.go`:
```go
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/store"
	"github.com/thuanho/pr-reviewers/internal/task"
)

type ReviewHandler struct {
	store       *store.Store
	asynqClient *asynq.Client
}

func NewReviewHandler(s *store.Store, ac *asynq.Client) *ReviewHandler {
	return &ReviewHandler{store: s, asynqClient: ac}
}

func (h *ReviewHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	review, err := h.store.GetReview(uint(id))
	if err != nil {
		Error(c, http.StatusNotFound, 4004, "review not found")
		return
	}
	Success(c, review)
}

func (h *ReviewHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var body struct {
		Summary        *string `json:"summary"`
		OverallVerdict *string `json:"overall_verdict"`
		Comments       []struct {
			ID   uint   `json:"id"`
			Body string `json:"body"`
		} `json:"comments"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, http.StatusBadRequest, 4000, "invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if body.Summary != nil {
		updates["summary"] = *body.Summary
	}
	if body.OverallVerdict != nil {
		updates["overall_verdict"] = *body.OverallVerdict
	}
	if len(updates) > 0 {
		h.store.UpdateReview(uint(id), updates)
	}

	for _, cmt := range body.Comments {
		h.store.UpdateComment(cmt.ID, cmt.Body)
	}

	review, _ := h.store.GetReview(uint(id))
	Success(c, review)
}

func (h *ReviewHandler) Approve(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	review, err := h.store.GetReview(uint(id))
	if err != nil {
		Error(c, http.StatusNotFound, 4004, "review not found")
		return
	}

	h.store.UpdateReview(review.ID, map[string]interface{}{"status": "approved"})

	payload, _ := json.Marshal(map[string]uint{"review_id": review.ID})
	h.asynqClient.Enqueue(asynq.NewTask(task.TypePostReview, payload))

	Success(c, map[string]string{"status": "review posting"})
}

func (h *ReviewHandler) Reject(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	h.store.UpdateReview(uint(id), map[string]interface{}{"status": "rejected"})

	Success(c, map[string]string{"status": "review rejected"})
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/handler/...`
Expected: No errors.

---

### Task 19: Create Config and Dashboard handlers

**Files:**
- Create: `internal/handler/configs.go`
- Create: `internal/handler/dashboard.go`

- [ ] **Step 1: Write Config handlers**

Write `internal/handler/configs.go`:
```go
package handler

import (
	"net/http"
	"os/exec"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/thuanho/pr-reviewers/internal/executor"
	"github.com/thuanho/pr-reviewers/internal/store"
)

type ConfigHandler struct {
	store    *store.Store
	registry *executor.Registry
}

func NewConfigHandler(s *store.Store, reg *executor.Registry) *ConfigHandler {
	return &ConfigHandler{store: s, registry: reg}
}

func (h *ConfigHandler) ListRepos(c *gin.Context) {
	configs, err := h.store.ListRepoConfigs()
	if err != nil {
		Error(c, http.StatusInternalServerError, 5999, "failed to list repo configs")
		return
	}
	Success(c, configs)
}

func (h *ConfigHandler) CreateRepo(c *gin.Context) {
	var cfg store.RepoConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		Error(c, http.StatusBadRequest, 4000, "invalid request body")
		return
	}
	if err := h.store.CreateRepoConfig(&cfg); err != nil {
		Error(c, http.StatusInternalServerError, 5999, "failed to create repo config")
		return
	}
	Success(c, cfg)
}

func (h *ConfigHandler) UpdateRepo(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var cfg store.RepoConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		Error(c, http.StatusBadRequest, 4000, "invalid request body")
		return
	}
	if err := h.store.UpdateRepoConfig(uint(id), &cfg); err != nil {
		Error(c, http.StatusInternalServerError, 5999, "failed to update repo config")
		return
	}
	Success(c, cfg)
}

func (h *ConfigHandler) DeleteRepo(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.store.DeleteRepoConfig(uint(id)); err != nil {
		Error(c, http.StatusInternalServerError, 5999, "failed to delete repo config")
		return
	}
	Success(c, nil)
}

func (h *ConfigHandler) TestConnection(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	// Since we can't easily get RepoConfig by ID without adding that method, let's use the existing one
	// For now, get repo config but we need a GetByID method
	// Simplified: just check if local path exists and is a git repo
	configs, _ := h.store.ListRepoConfigs()
	var found *store.RepoConfig
	for _, cfg := range configs {
		if cfg.ID == uint(id) {
			found = &cfg
			break
		}
	}
	if found == nil {
		Error(c, http.StatusNotFound, 4004, "repo config not found")
		return
	}

	cmd := exec.Command("git", "-C", found.LocalPath, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		Error(c, http.StatusBadRequest, 5005, "git repo not accessible at local path")
		return
	}

	Success(c, map[string]string{
		"local_path": found.LocalPath,
		"git_remote": string(out[:len(out)-1]),
	})
}

func (h *ConfigHandler) ListCLIs(c *gin.Context) {
	names := h.registry.List()
	Success(c, names)
}
```

- [ ] **Step 2: Write Dashboard handler**

Write `internal/handler/dashboard.go`:
```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/thuanho/pr-reviewers/internal/store"
)

type DashboardHandler struct {
	store *store.Store
}

func NewDashboardHandler(s *store.Store) *DashboardHandler {
	return &DashboardHandler{store: s}
}

func (h *DashboardHandler) Get(c *gin.Context) {
	var counts struct {
		Pending   int64 `json:"pending"`
		Reviewing int64 `json:"reviewing"`
		Drafted   int64 `json:"drafted"`
		Posted    int64 `json:"posted"`
		Failed    int64 `json:"failed"`
	}

	db := h.store.DB()
	db.Model(&store.PullRequest{}).Where("status = 'pending'").Count(&counts.Pending)
	db.Model(&store.PullRequest{}).Where("status = 'reviewing'").Count(&counts.Reviewing)
	db.Model(&store.PullRequest{}).Where("status = 'drafted'").Count(&counts.Drafted)
	db.Model(&store.PullRequest{}).Where("status = 'posted'").Count(&counts.Posted)
	db.Model(&store.PullRequest{}).Where("status = 'failed'").Count(&counts.Failed)

	var recent []store.Review
	db.Preload("Comments").Order("created_at DESC").Limit(10).Find(&recent)

	Success(c, map[string]interface{}{
		"counts": counts,
		"recent": recent,
	})
}

func (h *DashboardHandler) History(c *gin.Context) {
	page, perPage := GetPageAndPerPage(c)
	repo := c.Query("repo")

	reviews, total, err := h.store.ListHistory(page, perPage, repo)
	if err != nil {
		Error(c, http.StatusInternalServerError, 5999, "failed to list history")
		return
	}

	SuccessList(c, reviews, ListMeta{Page: page, PerPage: perPage, Total: total})
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/handler/...`
Expected: No errors.

---

### Task 20: Wire up API main.go

**Files:**
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Write full API main.go**

Write `cmd/api/main.go`:
```go
package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/config"
	"github.com/thuanho/pr-reviewers/internal/executor"
	"github.com/thuanho/pr-reviewers/internal/github"
	"github.com/thuanho/pr-reviewers/internal/handler"
	"github.com/thuanho/pr-reviewers/internal/store"
	"github.com/thuanho/pr-reviewers/internal/ws"
)

func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://pr_reviewer:pr_reviewer_dev@localhost:5432/pr_reviewer?sslmode=disable"
	}

	st, err := store.New(dsn)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	st.SeedCLIConfigs()

	// Seed repo configs from config file
	for _, rm := range cfg.Repos {
		st.CreateRepoConfig(&store.RepoConfig{
			RepoFullName: rm.Repo,
			LocalPath:    rm.LocalPath,
			CLI:          rm.CLI,
			ExtraRules:   rm.ExtraRules,
			Active:       true,
		})
	}

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	defer asynqClient.Close()

	ghClient := github.NewClient(cfg.GitHub.Token)

	reg := executor.NewRegistry()
	for _, ed := range cfg.Executors {
		if !ed.Active {
			continue
		}
		switch ed.Name {
		case "claude-code":
			reg.Register(executor.NewClaudeCodeExecutor(60 * 60)) // 60 min default
		case "codex":
			reg.Register(executor.NewCodexExecutor(60 * 60))
		}
	}

	hub, err := ws.NewHub()
	if err != nil {
		log.Fatalf("create ws hub: %v", err)
	}

	r := gin.Default()

	// CORS for dev
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Handlers
	prHandler := handler.NewPRHandler(st, asynqClient)
	reviewHandler := handler.NewReviewHandler(st, asynqClient)
	configHandler := handler.NewConfigHandler(st, reg)
	dashHandler := handler.NewDashboardHandler(st)

	api := r.Group("/api")
	{
		api.GET("/prs", prHandler.List)
		api.GET("/prs/:id", prHandler.Get)
		api.POST("/prs/:id/refresh", prHandler.Refresh)
		api.GET("/prs/:id/reviews", prHandler.ListReviews)

		api.GET("/reviews/:id", reviewHandler.Get)
		api.PUT("/reviews/:id", reviewHandler.Update)
		api.POST("/reviews/:id/approve", reviewHandler.Approve)
		api.POST("/reviews/:id/reject", reviewHandler.Reject)

		api.GET("/configs/repos", configHandler.ListRepos)
		api.POST("/configs/repos", configHandler.CreateRepo)
		api.PUT("/configs/repos/:id", configHandler.UpdateRepo)
		api.DELETE("/configs/repos/:id", configHandler.DeleteRepo)
		api.POST("/configs/repos/:id/test", configHandler.TestConnection)
		api.GET("/configs/clis", configHandler.ListCLIs)

		api.GET("/dashboard", dashHandler.Get)
		api.GET("/history", dashHandler.History)

		api.GET("/system/health", func(c *gin.Context) {
			handler.Success(c, map[string]string{"status": "ok"})
		})
	}

	// WebSocket
	r.GET("/ws", gin.WrapH(hub.Handler()))

	log.Println("pr-reviewer-api listening on :8080")
	r.Run(":8080")
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./cmd/api`
Expected: No errors.

---

### Task 21: Wire up Worker main.go

**Files:**
- Modify: `cmd/worker/main.go`

- [ ] **Step 1: Write full Worker main.go**

Write `cmd/worker/main.go`:
```go
package main

import (
	"log"
	"os"

	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/config"
	"github.com/thuanho/pr-reviewers/internal/executor"
	"github.com/thuanho/pr-reviewers/internal/github"
	"github.com/thuanho/pr-reviewers/internal/scheduler"
	"github.com/thuanho/pr-reviewers/internal/store"
	"github.com/thuanho/pr-reviewers/internal/task"
	"github.com/thuanho/pr-reviewers/internal/ws"
)

func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://pr_reviewer:pr_reviewer_dev@localhost:5432/pr_reviewer?sslmode=disable"
	}

	st, err := store.New(dsn)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisOpt := asynq.RedisClientOpt{Addr: redisAddr}

	asynqClient := asynq.NewClient(redisOpt)
	defer asynqClient.Close()

	ghClient := github.NewClient(cfg.GitHub.Token)

	reg := executor.NewRegistry()
	for _, ed := range cfg.Executors {
		if !ed.Active {
			continue
		}
		switch ed.Name {
		case "claude-code":
			reg.Register(executor.NewClaudeCodeExecutor(60 * 60))
		case "codex":
			reg.Register(executor.NewCodexExecutor(60 * 60))
		}
	}

	hub, err := ws.NewHub()
	if err != nil {
		log.Fatalf("create ws hub: %v", err)
	}

	// Mux for task handlers
	mux := asynq.NewServeMux()

	fetchHandler := task.NewFetchPRsHandler(st, ghClient, hub, asynqClient)
	mux.HandleFunc(task.TypeFetchAssignedPRs, fetchHandler.Handle)

	syncHandler := task.NewSyncPRStatusHandler(st, ghClient, hub)
	mux.HandleFunc(task.TypeSyncPRStatus, syncHandler.Handle)

	reviewHandler := task.NewExecuteReviewHandler(st, reg, hub)
	mux.HandleFunc(task.TypeExecuteReview, reviewHandler.Handle)

	postHandler := task.NewPostReviewHandler(st, cfg.GitHub.Token, hub)
	mux.HandleFunc(task.TypePostReview, postHandler.Handle)

	cleanupHandler := task.NewCleanupWorktreeHandler(st, cfg.Scheduler.CleanupWorktreeAfterDays)
	mux.HandleFunc(task.TypeCleanupWorktree, cleanupHandler.Handle)

	// Scheduler
	sched := asynq.NewScheduler(redisOpt, nil)
	scheduler.Register(sched, cfg.FetchInterval(), cfg.Scheduler.CleanupWorktreeAfterDays)

	go func() {
		if err := sched.Start(); err != nil {
			log.Fatalf("start scheduler: %v", err)
		}
	}()

	// Server
	server := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 4,
		Queues: map[string]int{
			"default": 1,
		},
	})

	log.Println("pr-reviewer-worker starting...")
	if err := server.Run(mux); err != nil {
		log.Fatalf("run worker: %v", err)
	}
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./cmd/worker`
Expected: No errors.

---

### Task 22: Create Makefile, config.yaml, and ecosystem.config.cjs

**Files:**
- Create: `Makefile`
- Create: `config.yaml` (from example)
- Create: `ecosystem.config.cjs`

- [ ] **Step 1: Write Makefile**

Write `Makefile`:
```makefile
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
	cd web && npm run dev

dev: dev-infra
	@echo "Start in separate terminals:"
	@echo "  Terminal 1: make dev-api"
	@echo "  Terminal 2: make dev-worker"
	@echo "  Terminal 3: make dev-web"

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker
	cd web && npm run build

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
```

- [ ] **Step 2: Copy config**

Run: `cp config.example.yaml config.yaml`

- [ ] **Step 3: Write PM2 ecosystem config**

Write `ecosystem.config.cjs`:
```javascript
module.exports = {
  apps: [
    {
      name: 'pr-reviewer-api',
      script: './bin/api',
      env: {
        PORT: '8080',
        DATABASE_URL: 'postgres://pr_reviewer:pr_reviewer_dev@localhost:5432/pr_reviewer?sslmode=disable',
        REDIS_URL: 'localhost:6379',
        CONFIG_PATH: './config.yaml',
      },
      log_file: './logs/api.log',
      error_file: './logs/api-error.log',
      max_restarts: 5,
      restart_delay: 5000,
    },
    {
      name: 'pr-reviewer-worker',
      script: './bin/worker',
      env: {
        DATABASE_URL: 'postgres://pr_reviewer:pr_reviewer_dev@localhost:5432/pr_reviewer?sslmode=disable',
        REDIS_URL: 'localhost:6379',
        CONFIG_PATH: './config.yaml',
      },
      log_file: './logs/worker.log',
      error_file: './logs/worker-error.log',
      max_restarts: 5,
      restart_delay: 10000,
    },
  ],
}
```

- [ ] **Step 4: Ensure logs dir exists**

Run: `mkdir -p logs bin`

---

### Task 23: Clone shadcn-admin template vào web/

**Files:**
- Clone toàn bộ template vào `web/`

- [ ] **Step 1: Clone template**

Run:
```bash
cd /Users/thuanho/Documents/personal/pr-reviewers
git clone https://github.com/satnaing/shadcn-admin.git web-temp
cp -r web-temp/* web/
cp web-temp/.* web/ 2>/dev/null || true
rm -rf web-temp
```

- [ ] **Step 2: Cài dependencies**

Run: `cd web && pnpm install`
Expected: Dependencies installed, no errors.

- [ ] **Step 3: Verify dev server khởi động**

Run: `cd web && pnpm dev`
Expected: Vite dev server starts, app renders on :5173.

---
### Task 24: Strip template — xóa auth, Clerk, feature pages không cần

**Files:**
- Xóa: nhiều files/directories

- [ ] **Step 1: Strip Clerk auth**

Xóa toàn bộ auth-related code:
- Xóa `npm uninstall @clerk/clerk-react` (bỏ khỏi package.json)
- Xóa `src/routes/clerk/` (toàn bộ directory)
- Xóa `src/routes/(auth)/` (toàn bộ directory)
- Xóa `src/features/auth/` (toàn bộ directory)
- Xóa `src/stores/auth-store.ts` và `src/stores/auth-store.test.ts`
- Sửa `src/main.tsx`: bỏ ClerkProvider import và wrapper

- [ ] **Step 2: Strip feature pages không cần**

Xóa các feature directories:
- `src/features/users/`
- `src/features/tasks/`
- `src/features/chats/`
- `src/features/apps/`
- `src/features/help-center/` (nếu có)

Xóa các routes tương ứng:
- `src/routes/_authenticated/users/`
- `src/routes/_authenticated/tasks/`
- `src/routes/_authenticated/chats/`
- `src/routes/_authenticated/apps/`
- `src/routes/_authenticated/help-center/`

- [ ] **Step 3: Strip components không cần**

Xóa:
- `src/components/layout/team-switcher.tsx`
- `src/components/layout/nav-user.tsx`
- `src/components/profile-dropdown.tsx`
- `src/components/sign-out-dialog.tsx`
- `src/components/sign-out-dialog.test.tsx`
- `src/components/coming-soon.tsx`
- `src/components/learn-more.tsx`

- [ ] **Step 4: Verify vẫn build được**

Run: `cd web && pnpm build`
Expected: Build succeeds, all references resolved.

---
### Task 25: Customize layout — sidebar, authenticated-layout

**Files:**
- Sửa: `src/components/layout/data/sidebar-data.ts`
- Sửa: `src/components/layout/app-sidebar.tsx`
- Sửa: `src/components/layout/authenticated-layout.tsx`
- Sửa: `src/routes/_authenticated/route.tsx`

- [ ] **Step 1: Định nghĩa sidebar navigation**

Write `src/components/layout/data/sidebar-data.ts`:
```typescript
import {
  LayoutDashboard,
  GitPullRequest,
  History,
  Settings,
  Database,
  Terminal,
} from 'lucide-react'

export interface SidebarData {
  user: AppUser
  teams: never[]  // No team switcher
  navGroups: NavGroup[]
}

export interface NavGroup {
  title: string
  items: NavItem[]
}

export interface NavItem {
  title: string
  url: string
  icon?: React.ComponentType<{ className?: string }>
  items?: NavSubItem[]
}

export interface NavSubItem {
  title: string
  url: string
}

export function getSidebarData(): SidebarData {
  return {
    user: { name: 'PR Reviewer', email: '', avatar: '' },
    teams: [],
    navGroups: [
      {
        title: 'General',
        items: [
          { title: 'Dashboard', url: '/', icon: LayoutDashboard },
          { title: 'Pull Requests', url: '/prs', icon: GitPullRequest },
          { title: 'History', url: '/history', icon: History },
        ],
      },
      {
        title: 'Settings',
        items: [
          {
            title: 'Settings',
            url: '/settings',
            icon: Settings,
            items: [
              { title: 'Repo Configs', url: '/settings/repos' },
              { title: 'CLI Executors', url: '/settings/clis' },
            ],
          },
        ],
      },
    ],
  }
}
```

- [ ] **Step 2: Sửa app-sidebar.tsx**

Điều chỉnh sidebar component để không dùng team-switcher và nav-user:
- Xóa import `TeamSwitcher` và `NavUser`
- Thay vùng user info trong sidebar bằng static app name hoặc gọn hơn
- Giữ nguyên `NavGroup` để render navigation

- [ ] **Step 3: Sửa authenticated-layout.tsx**

Bỏ auth check, giữ layout structure (sidebar + header + main):
- Xóa bất kỳ redirect/auth check nào
- Component chỉ còn là layout wrapper thuần túy

- [ ] **Step 4: Sửa route.tsx cho _authenticated group**

Xóa auth guard, giữ layout structure.

- [ ] **Step 5: Verify navigation hoạt động**

Run: `cd web && pnpm dev`
Expected: Sidebar hiển thị đúng nav items, click điều hướng được.

---
### Task 26: Thêm dependencies mới

**Files:**
- Sửa: `web/package.json`

- [ ] **Step 1: Cài TanStack Query và Centrifuge**

Run:
```bash
cd web
pnpm add @tanstack/react-query centrifuge
```

Expected: Packages added to package.json.

- [ ] **Step 2: Bọc QueryClientProvider trong main.tsx**

Sửa `src/main.tsx`: thêm `QueryClientProvider` bọc ngoài app.

- [ ] **Step 3: Verify build**

Run: `cd web && pnpm build`
Expected: Build succeeds.

---
### Task 27: Tạo API client và WebSocket hook

**Files:**
- Tạo: `web/src/lib/api.ts`
- Tạo: `web/src/hooks/use-ws.ts`
- Tạo: `web/src/stores/app-store.ts`

- [ ] **Step 1: Tạo API client**

Write `web/src/lib/api.ts`:
```typescript
const BASE = '/api'

interface ApiResponse<T> {
  code: number
  data: T
  error?: string
}

export interface ListData<T> {
  items: T[]
  meta: { page: number; per_page: number; total: number }
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${url}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  const json: ApiResponse<T> = await res.json()
  if (json.code !== 0) throw new Error(json.error || 'API error')
  return json.data
}

export const api = {
  getPRs: (params: Record<string, string> = {}) => {
    const qs = new URLSearchParams(params).toString()
    return request<ListData<any>>(`/prs?${qs}`)
  },
  getPR: (id: number) => request<any>(`/prs/${id}`),
  refreshPR: (id: number) => request<any>(`/prs/${id}/refresh`, { method: 'POST' }),
  getReviews: (prId: number) => request<any[]>(`/prs/${prId}/reviews`),
  getReview: (id: number) => request<any>(`/reviews/${id}`),
  updateReview: (id: number, data: any) =>
    request<any>(`/reviews/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  approveReview: (id: number) =>
    request<any>(`/reviews/${id}/approve`, { method: 'POST' }),
  rejectReview: (id: number) =>
    request<any>(`/reviews/${id}/reject`, { method: 'POST' }),
  getRepoConfigs: () => request<any[]>('/configs/repos'),
  createRepoConfig: (data: any) =>
    request<any>('/configs/repos', { method: 'POST', body: JSON.stringify(data) }),
  updateRepoConfig: (id: number, data: any) =>
    request<any>(`/configs/repos/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteRepoConfig: (id: number) =>
    request<any>(`/configs/repos/${id}`, { method: 'DELETE' }),
  testRepoConfig: (id: number) =>
    request<any>(`/configs/repos/${id}/test`, { method: 'POST' }),
  getCLIConfigs: () => request<string[]>('/configs/clis'),
  getDashboard: () => request<any>('/dashboard'),
  getHistory: (params: Record<string, string> = {}) => {
    const qs = new URLSearchParams(params).toString()
    return request<ListData<any>>(`/history?${qs}`)
  },
}
```

- [ ] **Step 2: Tạo WebSocket hook**

Write `web/src/hooks/use-ws.ts`:
```typescript
import { useEffect } from 'react'
import { Centrifuge } from 'centrifuge'
import { useQueryClient } from '@tanstack/react-query'

type WSEvent =
  | { type: 'pr.updated'; payload: any }
  | { type: 'review.created'; payload: any }
  | { type: 'review.posted'; payload: any }
  | { type: 'scheduler.tick'; payload: { lastRun: string; nextRun: string } }

export function useWebSocket() {
  const queryClient = useQueryClient()

  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const centrifuge = new Centrifuge(`${protocol}://${window.location.host}/ws`)

    centrifuge.on('publication', (ctx) => {
      const event = ctx.data as WSEvent
      switch (event.type) {
        case 'pr.updated':
          queryClient.setQueryData(['prs', event.payload.id], event.payload)
          queryClient.invalidateQueries({ queryKey: ['prs'] })
          break
        case 'review.created':
          queryClient.setQueryData(
            ['prs', event.payload.pull_request_id, 'reviews'],
            (old: any[]) => (old ? [...old, event.payload] : [event.payload])
          )
          queryClient.invalidateQueries({ queryKey: ['prs'] })
          break
        case 'review.posted':
          queryClient.setQueryData(['reviews', event.payload.id], event.payload)
          queryClient.invalidateQueries({ queryKey: ['prs'] })
          break
        case 'scheduler.tick':
          queryClient.invalidateQueries({ queryKey: ['prs'] })
          queryClient.invalidateQueries({ queryKey: ['dashboard'] })
          break
      }
    })

    centrifuge.connect()
    return () => centrifuge.disconnect()
  }, [queryClient])
}
```

- [ ] **Step 3: Tạo app store**

Write `web/src/stores/app-store.ts`:
```typescript
import { create } from 'zustand'

interface AppStore {
  sidebarOpen: boolean
  toggleSidebar: () => void
  prStatusFilter: string
  setPrStatusFilter: (status: string) => void
  repoFilter: string
  setRepoFilter: (repo: string) => void
}

export const useAppStore = create<AppStore>((set) => ({
  sidebarOpen: true,
  toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),
  prStatusFilter: '',
  setPrStatusFilter: (prStatusFilter) => set({ prStatusFilter }),
  repoFilter: '',
  setRepoFilter: (repoFilter) => set({ repoFilter }),
}))
```

- [ ] **Step 4: Verify build**

Run: `cd web && pnpm build`
Expected: No TypeScript errors.

---
### Task 28: Tạo feature pages — Dashboard + PR List + PR Detail

**Files:**
- Tạo: `src/features/dashboard/index.tsx`
- Tạo: `src/features/pull-requests/index.tsx`
- Tạo: `src/features/pull-requests/pr-detail.tsx`
- Tạo: `src/features/pull-requests/review-detail.tsx`
- Tạo: `src/features/history/index.tsx`
- Tạo: `src/features/settings/repos.tsx`
- Tạo: `src/features/settings/clis.tsx`

- [ ] **Step 1: Tạo Dashboard page**

Write `src/features/dashboard/index.tsx`:
```typescript
import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { api } from '@/lib/api'
import { useWebSocket } from '@/hooks/use-ws'

const statusLabels: Record<string, string> = {
  pending: 'Pending', reviewing: 'Reviewing', drafted: 'Drafted',
  posted: 'Posted', failed: 'Failed', closed: 'Closed',
}

const verdictColors: Record<string, string> = {
  approve: 'bg-green-100 text-green-700',
  request_changes: 'bg-red-100 text-red-700',
  comment: 'bg-blue-100 text-blue-700',
}

export default function Dashboard() {
  useWebSocket()
  const { data, isLoading } = useQuery({
    queryKey: ['dashboard'],
    queryFn: api.getDashboard,
    refetchInterval: 30_000,
  })

  if (isLoading) return <div className="p-8 space-y-4"><Skeleton className="h-24" /><Skeleton className="h-48" /></div>

  return (
    <div className="p-6 space-y-6">
      <h2 className="text-2xl font-bold tracking-tight">Dashboard</h2>
      <div className="grid grid-cols-3 lg:grid-cols-6 gap-4">
        {Object.entries(data.counts).map(([key, count]) => (
          <Card key={key}>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">{statusLabels[key] || key}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold">{count as number}</div>
            </CardContent>
          </Card>
        ))}
      </div>
      <h3 className="text-lg font-semibold">Recent Reviews</h3>
      <div className="space-y-3">
        {data.recent?.map((r: any) => (
          <Card key={r.id}>
            <CardContent className="flex items-center justify-between py-4">
              <div>
                <span className="font-medium">Review #{r.id}</span>
                <span className="text-sm text-muted-foreground ml-2">PR #{r.pull_request_id}</span>
                <p className="text-sm mt-1 text-muted-foreground line-clamp-2">{r.summary}</p>
              </div>
              <Badge className={verdictColors[r.overall_verdict] || 'bg-gray-100'}>
                {r.overall_verdict}
              </Badge>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Tạo PR List page**

Write `src/features/pull-requests/index.tsx`:
```typescript
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Badge } from '@/components/ui/badge'
import { DataTable } from '@/components/data-table'
import { api } from '@/lib/api'
import { useWebSocket } from '@/hooks/use-ws'
import { useAppStore } from '@/stores/app-store'

export default function PRList() {
  useWebSocket()
  const { prStatusFilter } = useAppStore()

  const { data, isLoading } = useQuery({
    queryKey: ['prs', prStatusFilter],
    queryFn: () => api.getPRs(prStatusFilter ? { status: prStatusFilter } : {}),
    refetchInterval: 30_000,
  })

  const columns = [
    { accessorKey: 'repo_full_name', header: 'Repo' },
    { accessorKey: 'title', header: 'Title',
      cell: ({ row }: any) => (
        <Link to="/prs/$prId" params={{ prId: String(row.original.id) }} className="font-medium hover:underline">
          {row.original.title}
        </Link>
      ),
    },
    { accessorKey: 'author', header: 'Author' },
    { accessorKey: 'status', header: 'Status',
      cell: ({ row }: any) => <Badge variant="outline">{row.original.status}</Badge>,
    },
    { accessorKey: 'created_at', header: 'Created',
      cell: ({ row }: any) => new Date(row.original.created_at).toLocaleDateString(),
    },
  ]

  return (
    <div className="p-6 space-y-4">
      <h2 className="text-2xl font-bold tracking-tight">Pull Requests</h2>
      <DataTable
        columns={columns}
        data={data?.items ?? []}
        pageCount={Math.ceil((data?.meta?.total ?? 0) / (data?.meta?.per_page ?? 20))}
        isLoading={isLoading}
      />
    </div>
  )
}
```

- [ ] **Step 3: Tạo PR Detail page**

Write `src/features/pull-requests/pr-detail.tsx`:
```typescript
import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { api } from '@/lib/api'

export default function PRDetail() {
  const { prId } = useParams({ from: '/_authenticated/prs/$prId' })
  const { data: pr, isLoading } = useQuery({
    queryKey: ['prs', Number(prId)],
    queryFn: () => api.getPR(Number(prId)),
  })

  if (isLoading) return <div className="p-6 space-y-4"><Skeleton className="h-32" /><Skeleton className="h-64" /></div>
  if (!pr) return <div className="p-6">PR not found</div>

  return (
    <div className="p-6 space-y-6">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">{pr.title}</h2>
        <p className="text-sm text-muted-foreground mt-1">
          {pr.repo_full_name}#{pr.number} by {pr.author} · {pr.base_branch} ← {pr.head_branch}
        </p>
        <div className="flex gap-2 mt-2">
          <Badge variant="outline">{pr.status}</Badge>
          <a href={pr.url} target="_blank" rel="noopener noreferrer">
            <Button variant="outline" size="sm">View on GitHub</Button>
          </a>
        </div>
      </div>
      <Tabs defaultValue="reviews">
        <TabsList>
          <TabsTrigger value="reviews">Reviews ({pr.reviews?.length ?? 0})</TabsTrigger>
          <TabsTrigger value="info">Info</TabsTrigger>
        </TabsList>
        <TabsContent value="reviews" className="space-y-3">
          {pr.reviews?.map((r: any) => (
            <Link key={r.id} to="/prs/$prId/reviews/$reviewId" params={{ prId: String(pr.id), reviewId: String(r.id) }}>
              <Card className="hover:shadow-md transition-shadow cursor-pointer">
                <CardContent className="flex items-center justify-between py-4">
                  <div>
                    <span className="text-sm text-muted-foreground">Commit {r.commit_sha?.slice(0, 7)}</span>
                    <p className="mt-1 line-clamp-2">{r.summary}</p>
                  </div>
                  <div className="flex gap-2 items-center">
                    <Badge>{r.overall_verdict}</Badge>
                    <Badge variant="outline">{r.status}</Badge>
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))}
          {(!pr.reviews || pr.reviews.length === 0) && (
            <p className="text-muted-foreground">No reviews yet.</p>
          )}
        </TabsContent>
        <TabsContent value="info">
          <Card>
            <CardContent className="py-4 space-y-2 text-sm">
              <div><span className="font-medium">Head SHA:</span> {pr.head_sha}</div>
              <div><span className="font-medium">Worktree:</span> {pr.worktree_path || 'N/A'}</div>
              <div><span className="font-medium">First seen:</span> {new Date(pr.created_at).toLocaleDateString()}</div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
```

- [ ] **Step 4: Tạo Review Detail page**

Write `src/features/pull-requests/review-detail.tsx`:
```typescript
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useParams } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Skeleton } from '@/components/ui/skeleton'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { api } from '@/lib/api'

export default function ReviewDetail() {
  const { reviewId } = useParams({ from: '/_authenticated/prs/$prId/reviews/$reviewId' })
  const queryClient = useQueryClient()
  const [editedComments, setEditedComments] = useState<Record<number, string>>({})
  const [editedSummary, setEditedSummary] = useState('')
  const [showApprove, setShowApprove] = useState(false)

  const { data: review, isLoading } = useQuery({
    queryKey: ['reviews', Number(reviewId)],
    queryFn: () => api.getReview(Number(reviewId)),
  })

  const updateMutation = useMutation({
    mutationFn: (data: any) => api.updateReview(Number(reviewId), data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['reviews', Number(reviewId)] }),
  })

  const approveMutation = useMutation({
    mutationFn: () => api.approveReview(Number(reviewId)),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['prs'] }),
  })

  if (isLoading) return <div className="p-6 space-y-4"><Skeleton className="h-32" /><Skeleton className="h-64" /></div>
  if (!review) return <div className="p-6">Review not found</div>

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Review #{review.id}</h2>
          <div className="flex gap-2 mt-2">
            <Badge>{review.overall_verdict}</Badge>
            <Badge variant="outline">{review.status}</Badge>
            <span className="text-sm text-muted-foreground">by {review.executor_name}</span>
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => updateMutation.mutate({
            summary: editedSummary || review.summary,
            comments: Object.entries(editedComments).map(([id, body]) => ({ id: Number(id), body })),
          })}>
            Save Changes
          </Button>
          {review.status !== 'posted' && (
            <Button onClick={() => setShowApprove(true)}>Approve & Post</Button>
          )}
        </div>
      </div>

      <Card>
        <CardHeader><CardTitle>Summary</CardTitle></CardHeader>
        <CardContent>
          <Textarea
            value={editedSummary !== '' ? editedSummary : review.summary}
            onChange={(e) => setEditedSummary(e.target.value)}
            rows={4}
          />
        </CardContent>
      </Card>

      <h3 className="text-lg font-semibold">Comments ({review.comments?.length ?? 0})</h3>
      <div className="space-y-3">
        {review.comments?.map((c: any) => (
          <Card key={c.id}>
            <CardContent className="py-4">
              <div className="text-sm font-medium text-muted-foreground mb-2">
                {c.file_path}:{c.line_start}-{c.line_end}
              </div>
              <Textarea
                value={editedComments[c.id] ?? c.body}
                onChange={(e) => setEditedComments({ ...editedComments, [c.id]: e.target.value })}
                rows={3}
              />
            </CardContent>
          </Card>
        ))}
      </div>

      <ConfirmDialog
        open={showApprove}
        onOpenChange={setShowApprove}
        title="Approve & Post Review"
        description="This will post the review to GitHub. Are you sure?"
        onConfirm={() => { approveMutation.mutate(); setShowApprove(false) }}
      />
    </div>
  )
}
```

- [ ] **Step 5: Tạo History page**

Write `src/features/history/index.tsx`:
```typescript
import { useQuery } from '@tanstack/react-query'
import { Badge } from '@/components/ui/badge'
import { DataTable } from '@/components/data-table'
import { api } from '@/lib/api'

export default function History() {
  const { data, isLoading } = useQuery({
    queryKey: ['history'],
    queryFn: () => api.getHistory(),
    refetchInterval: 30_000,
  })

  const columns = [
    { accessorKey: 'id', header: 'Review ID' },
    { accessorKey: 'pull_request_id', header: 'PR' },
    { accessorKey: 'overall_verdict', header: 'Verdict',
      cell: ({ row }: any) => <Badge>{row.original.overall_verdict}</Badge>,
    },
    { accessorKey: 'status', header: 'Status',
      cell: ({ row }: any) => <Badge variant="outline">{row.original.status}</Badge>,
    },
    { accessorKey: 'executor_name', header: 'Executor' },
    { accessorKey: 'created_at', header: 'Date',
      cell: ({ row }: any) => new Date(row.original.created_at).toLocaleDateString(),
    },
  ]

  return (
    <div className="p-6 space-y-4">
      <h2 className="text-2xl font-bold tracking-tight">Review History</h2>
      <DataTable
        columns={columns}
        data={data?.items ?? []}
        pageCount={Math.ceil((data?.meta?.total ?? 0) / (data?.meta?.per_page ?? 20))}
        isLoading={isLoading}
      />
    </div>
  )
}
```

- [ ] **Step 6: Tạo Settings pages**

Write `src/features/settings/repos.tsx`:
```typescript
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { SelectDropdown } from '@/components/select-dropdown'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { api } from '@/lib/api'

export default function SettingsRepos() {
  const queryClient = useQueryClient()
  const { data: configs } = useQuery({ queryKey: ['repo-configs'], queryFn: api.getRepoConfigs })
  const [showAdd, setShowAdd] = useState(false)
  const [form, setForm] = useState({ repo_full_name: '', local_path: '', cli: 'claude-code', extra_rules: '' })
  const [deleteId, setDeleteId] = useState<number | null>(null)

  const createMutation = useMutation({
    mutationFn: (data: any) => api.createRepoConfig(data),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['repo-configs'] }); setShowAdd(false); setForm({ repo_full_name: '', local_path: '', cli: 'claude-code', extra_rules: '' }) },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.deleteRepoConfig(id),
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: ['repo-configs'] }); setDeleteId(null) },
  })

  const testMutation = useMutation({
    mutationFn: (id: number) => api.testRepoConfig(id),
  })

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold tracking-tight">Repo Configurations</h2>
        <Button onClick={() => setShowAdd(true)}>Add Repo</Button>
      </div>

      {showAdd && (
        <Card>
          <CardHeader><CardTitle>Add Repository</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            <Input placeholder="owner/repo" value={form.repo_full_name}
              onChange={(e) => setForm({ ...form, repo_full_name: e.target.value })} />
            <Input placeholder="/path/to/local/repo" value={form.local_path}
              onChange={(e) => setForm({ ...form, local_path: e.target.value })} />
            <Input placeholder="Extra rules (optional)" value={form.extra_rules}
              onChange={(e) => setForm({ ...form, extra_rules: e.target.value })} />
            <SelectDropdown value={form.cli} onChange={(v) => setForm({ ...form, cli: v })}
              options={[{ label: 'Claude Code', value: 'claude-code' }, { label: 'Codex', value: 'codex' }]} />
            <div className="flex gap-2">
              <Button onClick={() => createMutation.mutate({ ...form, active: true, extra_rules: form.extra_rules || null })}>Save</Button>
              <Button variant="outline" onClick={() => setShowAdd(false)}>Cancel</Button>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="space-y-2">
        {(Array.isArray(configs) ? configs : []).map((cfg: any) => (
          <Card key={cfg.id}>
            <CardContent className="flex items-center justify-between py-4">
              <div>
                <span className="font-medium">{cfg.repo_full_name}</span>
                <span className="text-sm text-muted-foreground ml-4">{cfg.local_path}</span>
                <span className="text-sm text-muted-foreground ml-2">[{cfg.cli}]</span>
              </div>
              <div className="flex gap-2">
                <Button variant="outline" size="sm" onClick={() => testMutation.mutate(cfg.id)}>Test</Button>
                <Button variant="destructive" size="sm" onClick={() => setDeleteId(cfg.id)}>Delete</Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <ConfirmDialog open={deleteId !== null} onOpenChange={() => setDeleteId(null)}
        title="Delete Repo Config" description="Are you sure you want to delete this configuration?"
        onConfirm={() => deleteMutation.mutate(deleteId!)} />
    </div>
  )
}
```

Write `src/features/settings/clis.tsx`:
```typescript
import { useQuery } from '@tanstack/react-query'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { api } from '@/lib/api'

export default function SettingsCLIs() {
  const { data: clis } = useQuery({ queryKey: ['cli-configs'], queryFn: api.getCLIConfigs })

  return (
    <div className="p-6 space-y-4">
      <h2 className="text-2xl font-bold tracking-tight">CLI Executors</h2>
      <div className="space-y-2">
        {(Array.isArray(clis) ? clis : []).map((name: string) => (
          <Card key={name}>
            <CardContent className="flex items-center justify-between py-4">
              <span className="font-medium">{name}</span>
              <Badge variant="outline">Available</Badge>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
```

- [ ] **Step 7: Verify build**

Run: `cd web && pnpm build`
Expected: No TypeScript errors.

---
### Task 29: Tạo routes (file-based routing)

**Files:**
- Tạo/sửa: `src/routes/__root.tsx`
- Tạo/sửa: `src/routes/_authenticated/route.tsx`
- Tạo: `src/routes/_authenticated/index.tsx`
- Tạo: `src/routes/_authenticated/prs.tsx`
- Tạo: `src/routes/_authenticated/prs.$prId.tsx`
- Tạo: `src/routes/_authenticated/prs.$prId.reviews.$reviewId.tsx`
- Tạo: `src/routes/_authenticated/history.tsx`
- Tạo: `src/routes/_authenticated/settings/route.tsx`
- Tạo: `src/routes/_authenticated/settings/index.tsx`
- Tạo: `src/routes/_authenticated/settings/repos.tsx`
- Tạo: `src/routes/_authenticated/settings/clis.tsx`

- [ ] **Step 1: Tạo __root.tsx**

Write `src/routes/__root.tsx`:
```typescript
import { createRootRoute, Outlet } from '@tanstack/react-router'

export const Route = createRootRoute({
  component: () => <Outlet />,
})
```

- [ ] **Step 2: Tạo _authenticated layout route**

Write `src/routes/_authenticated/route.tsx`:
```typescript
import { createRoute } from '@tanstack/react-router'
import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'
import { rootRoute } from '../__root'

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  id: '_authenticated',
  component: () => (
    <AuthenticatedLayout>
      <Outlet />
    </AuthenticatedLayout>
  ),
})
```

- [ ] **Step 3: Tạo page routes**

Mỗi route file đơn giản import feature component tương ứng:

Write `src/routes/_authenticated/index.tsx`:
```typescript
import { createRoute } from '@tanstack/react-router'
import { Route as AuthRoute } from './route'
import Dashboard from '@/features/dashboard'

export const Route = createRoute({
  getParentRoute: () => AuthRoute,
  path: '/',
  component: Dashboard,
})
```

Write `src/routes/_authenticated/prs.tsx`:
```typescript
import { createRoute } from '@tanstack/react-router'
import { Route as AuthRoute } from './route'
import PRList from '@/features/pull-requests'

export const Route = createRoute({
  getParentRoute: () => AuthRoute,
  path: '/prs',
  component: PRList,
})
```

Write `src/routes/_authenticated/prs.$prId.tsx`:
```typescript
import { createRoute } from '@tanstack/react-router'
import { Route as AuthRoute } from './route'
import PRDetail from '@/features/pull-requests/pr-detail'

export const Route = createRoute({
  getParentRoute: () => AuthRoute,
  path: '/prs/$prId',
  component: PRDetail,
})
```

Write `src/routes/_authenticated/prs.$prId.reviews.$reviewId.tsx`:
```typescript
import { createRoute } from '@tanstack/react-router'
import { Route as AuthRoute } from './route'
import ReviewDetail from '@/features/pull-requests/review-detail'

export const Route = createRoute({
  getParentRoute: () => AuthRoute,
  path: '/prs/$prId/reviews/$reviewId',
  component: ReviewDetail,
})
```

Write `src/routes/_authenticated/history.tsx`:
```typescript
import { createRoute } from '@tanstack/react-router'
import { Route as AuthRoute } from './route'
import History from '@/features/history'

export const Route = createRoute({
  getParentRoute: () => AuthRoute,
  path: '/history',
  component: History,
})
```

Write `src/routes/_authenticated/settings/route.tsx`:
```typescript
import { createRoute } from '@tanstack/react-router'
import { Route as AuthRoute } from '../route'

export const Route = createRoute({
  getParentRoute: () => AuthRoute,
  path: '/settings',
})
```

Write `src/routes/_authenticated/settings/index.tsx`:
```typescript
import { createRoute, Navigate } from '@tanstack/react-router'
import { Route as SettingsRoute } from './route'

export const Route = createRoute({
  getParentRoute: () => SettingsRoute,
  path: '/',
  component: () => <Navigate to="/settings/repos" />,
})
```

Write `src/routes/_authenticated/settings/repos.tsx`:
```typescript
import { createRoute } from '@tanstack/react-router'
import { Route as SettingsRoute } from './route'
import SettingsRepos from '@/features/settings/repos'

export const Route = createRoute({
  getParentRoute: () => SettingsRoute,
  path: '/repos',
  component: SettingsRepos,
})
```

Write `src/routes/_authenticated/settings/clis.tsx`:
```typescript
import { createRoute } from '@tanstack/react-router'
import { Route as SettingsRoute } from './route'
import SettingsCLIs from '@/features/settings/clis'

export const Route = createRoute({
  getParentRoute: () => SettingsRoute,
  path: '/clis',
  component: SettingsCLIs,
})
```

- [ ] **Step 2: Verify route tree generation**

Run: `cd web && pnpm dev`
Expected: No route errors, dev server starts.

---
### Task 30: Build & verify frontend

- [ ] **Step 1: Build production**

Run: `cd web && pnpm build`
Expected: Build succeeds, output in `web/dist/`.

- [ ] **Step 2: Verify all pages render**

Run: `cd web && pnpm dev`, then check each route:
- `/` → Dashboard with stats
- `/prs` → PR list with data table
- `/prs/1` → PR detail with reviews tab
- `/history` → History with data table
- `/settings/repos` → Repo config list
- `/settings/clis` → CLI list

---
### Task 31: Update Go API embed path + Makefile + PM2 config

- [ ] **Step 1: Update embed path**

Trong `cmd/api/main.go`, embed directive vẫn là `//go:embed web/dist/*` (không đổi, vì Vite build output vẫn là `web/dist/`).

- [ ] **Step 2: Update Makefile**

Sửa `make dev-web` và `make build` từ `npm` sang `pnpm`:
```makefile
dev-web:
	cd web && pnpm dev

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker
	cd web && pnpm build
```

- [ ] **Step 3: Verify full production build**

Run: `make build`
Expected: `bin/api`, `bin/worker` created, `web/dist/` populated.

- [ ] **Step 4: Verify Go binary serves frontend**

Run: `./bin/api` và mở browser tại `http://localhost:8080`
Expected: React app served từ Go binary.

---

### Task 32: Initialize git repository and final setup

- [ ] **Step 1: Initialize git**

Run:
```bash
cd /Users/thuanho/Documents/personal/pr-reviewers
git init
echo "bin/\nlogs/\nnode_modules/\nconfig.yaml\n/tmp/" > .gitignore
git add .
git commit -m "feat: initial project scaffold with shadcn-admin frontend"
```

- [ ] **Step 2: Setup database**

Run:
```bash
createdb pr_reviewer
# Or via psql:
# psql -c "CREATE DATABASE pr_reviewer"
```

- [ ] **Step 3: Run full dev cycle to verify**

Terminal 1: `make dev-api`
Terminal 2: `make dev-worker`
Terminal 3: `make dev-web`
Expected: API on :8080, Worker with scheduler, Frontend on :5173 proxying to API.

---
### Task 33: Production build and PM2 start

- [ ] **Step 1: Build production binaries**

Run: `make build`
Expected:
- `bin/api` binary created (with embedded frontend)
- `bin/worker` binary created
- `web/dist/` populated via pnpm build

- [ ] **Step 2: Configure config.yaml**

Edit `config.yaml` with real GitHub token and repo mappings.

- [ ] **Step 3: Start with PM2**

Run:
```bash
pm2 start ecosystem.config.cjs
pm2 status
pm2 logs
```
Expected: Both processes running, API on :8080 serving frontend, worker processing tasks.

- [ ] **Step 4: Enable PM2 startup on boot**

Run:
```bash
pm2 startup
pm2 save
```

---
### Task 34: Database setup script

**Files:**
- Create: `scripts/setup-db.sh`

- [ ] **Step 1: Write database setup script**

Write `scripts/setup-db.sh`:
```bash
#!/bin/bash
set -e

echo "Creating database..."
psql -U postgres -c "CREATE USER pr_reviewer WITH PASSWORD 'pr_reviewer_dev';" 2>/dev/null || true
psql -U postgres -c "CREATE DATABASE pr_reviewer OWNER pr_reviewer;" 2>/dev/null || true
echo "Database ready."
```

Run: `chmod +x scripts/setup-db.sh && ./scripts/setup-db.sh`
