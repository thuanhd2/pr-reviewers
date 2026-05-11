package main

import (
	"log"
	"os"

	"github.com/hibiken/asynq"
	"github.com/thuanho/pr-reviewers/internal/config"
	"github.com/thuanho/pr-reviewers/internal/task"
)

// enqueue task TypeFetchAssignedPRs to the queue

func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	redisAddr := cfg.Redis.Addr
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	defer asynqClient.Close()

	asynqClient.Enqueue(asynq.NewTask(task.TypeFetchAssignedPRs, nil))

	log.Println("Task TypeFetchAssignedPRs enqueued")
}
