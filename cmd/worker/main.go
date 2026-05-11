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

	dsn := cfg.Database.URL
	st, err := store.New(dsn)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	redisAddr := cfg.Redis.Addr
	redisOpt := asynq.RedisClientOpt{Addr: redisAddr}

	asynqClient := asynq.NewClient(redisOpt)
	defer asynqClient.Close()

	ghClient := github.NewClient(cfg.GitHub.Token)

	reg := executor.NewRegistry()
	// load executors from database
	executors, err := st.ListCLIConfigs()
	if err != nil {
		log.Fatalf("list cli configs: %v", err)
	}
	for _, ed := range executors {
		if !ed.Active {
			continue
		}
		switch ed.Name {
		case "claude-code":
			reg.Register(executor.NewClaudeCodeExecutor(60 * 60 * 1_000_000_000))
		case "codex":
			reg.Register(executor.NewCodexExecutor(60 * 60 * 1_000_000_000))
		case "deepseek":
			reg.Register(executor.NewDeepSeekExecutor(cfg.DeepSeek, 60*60*1_000_000_000))
		}
	}

	hub, err := ws.NewHub()
	if err != nil {
		log.Fatalf("create ws hub: %v", err)
	}

	tracker := scheduler.NewJobTracker(st, hub)

	fetchInterval := cfg.FetchInterval()
	cronSpec := "@every " + fetchInterval.String()

	tracker.UpsertJobDefinitions([]scheduler.JobDef{
		{JobName: "Fetch assigned PRs", TaskType: task.TypeFetchAssignedPRs, CronSpec: cronSpec},
		{JobName: "Sync PR status", TaskType: task.TypeSyncPRStatus, CronSpec: cronSpec},
		{JobName: "Cleanup worktrees", TaskType: task.TypeCleanupWorktree, CronSpec: "0 2 * * *"},
	})

	// Mux for task handlers
	mux := asynq.NewServeMux()

	fetchHandler := task.NewFetchPRsHandler(st, ghClient, hub, asynqClient)
	mux.HandleFunc(task.TypeFetchAssignedPRs, tracker.Wrap("Fetch assigned PRs", cronSpec, fetchHandler.Handle))

	syncHandler := task.NewSyncPRStatusHandler(st, ghClient, hub)
	mux.HandleFunc(task.TypeSyncPRStatus, tracker.Wrap("Sync PR status", cronSpec, syncHandler.Handle))

	reviewHandler := task.NewExecuteReviewHandler(st, reg, hub)
	mux.HandleFunc(task.TypeExecuteReview, reviewHandler.Handle)

	postHandler := task.NewPostReviewHandler(st, ghClient, hub)
	mux.HandleFunc(task.TypePostReview, postHandler.Handle)

	cleanupHandler := task.NewCleanupWorktreeHandler(st, cfg.Scheduler.CleanupWorktreeAfterDays)
	mux.HandleFunc(task.TypeCleanupWorktree, tracker.Wrap("Cleanup worktrees", "0 2 * * *", cleanupHandler.Handle))

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
