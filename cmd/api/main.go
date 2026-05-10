package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/config"
	"github.com/thuanho/pr-reviewers/internal/executor"
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

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	defer asynqClient.Close()

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

	// Serve frontend static files (SPA fallback)
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/ws") {
			c.JSON(http.StatusNotFound, handler.Response{Code: 4004, Error: "not found"})
			return
		}
		c.File("web/dist/index.html")
	})
	r.Static("/assets", "web/dist/assets")

	log.Println("pr-reviewer-api listening on :8080")
	r.Run(":8080")
}
