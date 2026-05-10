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
			reg.Register(executor.NewClaudeCodeExecutor(60 * 60 * 1_000_000_000))
		case "codex":
			reg.Register(executor.NewCodexExecutor(60 * 60 * 1_000_000_000))
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

	postHandler := task.NewPostReviewHandler(st, ghClient, hub)
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
