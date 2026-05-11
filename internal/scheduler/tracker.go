package scheduler

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/robfig/cron/v3"

	"github.com/thuanho/pr-reviewers/internal/store"
	"github.com/thuanho/pr-reviewers/internal/ws"
)

type JobDef struct {
	JobName  string
	TaskType string
	CronSpec string
}

type JobTracker struct {
	store *store.Store
	hub   *ws.Hub
}

func NewJobTracker(s *store.Store, hub *ws.Hub) *JobTracker {
	return &JobTracker{store: s, hub: hub}
}

func (t *JobTracker) UpsertJobDefinitions(defs []JobDef) {
	for _, d := range defs {
		job := &store.SchedulerJob{
			JobName:  d.JobName,
			TaskType: d.TaskType,
			CronSpec: d.CronSpec,
			Status:   "idle",
		}
		if err := t.store.UpsertSchedulerJob(job); err != nil {
			log.Printf("upsert scheduler job %s: %v", d.JobName, err)
		}
	}
}

func (t *JobTracker) Wrap(jobName, cronSpec string, handler asynq.HandlerFunc) asynq.HandlerFunc {
	return func(ctx context.Context, task *asynq.Task) error {
		t.store.SetSchedulerJobRunning(jobName)

		now := time.Now()

		var lastError *string
		err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("scheduler job %s panicked: %v", jobName, r)
					err = nil // ensure we don't double-report
					s := "panic: " + formatPanic(r)
					lastError = &s
				}
			}()
			return handler(ctx, task)
		}()

		status := "idle"
		if err != nil {
			status = "failed"
			s := err.Error()
			lastError = &s
			log.Printf("scheduler job %s failed: %v", jobName, err)
		}

		nextRun := computeNextRun(cronSpec, now)
		if err := t.store.RecordSchedulerJobEnd(jobName, status, lastError, now, nextRun); err != nil {
			log.Printf("record scheduler job end %s: %v", jobName, err)
		}

		payload, _ := json.Marshal(map[string]interface{}{
			"type": "scheduler.tick",
			"payload": map[string]string{
				"lastRun": now.Format(time.RFC3339),
				"nextRun": nextRun.Format(time.RFC3339),
			},
		})
		t.hub.Publish("pr-updates", payload)

		return err
	}
}

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

func computeNextRun(cronSpec string, from time.Time) *time.Time {
	// Asynq @every syntax: parse as duration
	if spec, ok := strings.CutPrefix(cronSpec, "@every "); ok {
		d, err := time.ParseDuration(spec)
		if err != nil {
			return nil
		}
		t := from.Add(d)
		return &t
	}
	// Standard cron expression
	sched, err := cronParser.Parse(cronSpec)
	if err != nil {
		return nil
	}
	t := sched.Next(from)
	if t.IsZero() {
		return nil
	}
	return &t
}

func formatPanic(r any) string {
	switch v := r.(type) {
	case string:
		return v
	case error:
		return v.Error()
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}
