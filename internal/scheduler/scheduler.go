package scheduler

import (
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/task"
)

func Register(scheduler *asynq.Scheduler, fetchInterval time.Duration, cleanupDays int) {
	cronSpec := fmt.Sprintf("@every %s", fetchInterval.String())
	scheduler.Register(cronSpec, asynq.NewTask(task.TypeFetchAssignedPRs, nil))
	scheduler.Register(cronSpec, asynq.NewTask(task.TypeSyncPRStatus, nil))

	// Daily at 2am
	scheduler.Register("0 2 * * *", asynq.NewTask(task.TypeCleanupWorktree, nil))
}
