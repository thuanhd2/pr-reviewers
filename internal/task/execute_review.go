package task

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

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
	log.Println("Starting to execute review")
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

	var logsBuffer bytes.Buffer
	logLine := func(format string, args ...any) {
		ts := time.Now().Format("2006-01-02 15:04:05")
		line := fmt.Sprintf("[%s] "+format+"\n", append([]any{ts}, args...)...)
		logsBuffer.WriteString(line)
	}

	// Create or recover Review early in the process
	var review *store.Review
	latestReview, err := h.store.GetLatestReviewForPR(pr.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("get latest review: %w", err)
	}
	if err == nil && latestReview.CommitSHA == pr.HeadSHA {
		review = latestReview
		h.store.UpdateReview(review.ID, map[string]any{"status": "reviewing"})
	} else {
		review = &store.Review{
			PullRequestID: pr.ID,
			CommitSHA:     pr.HeadSHA,
			Status:        "reviewing",
		}
		if err := h.store.CreateReview(review); err != nil {
			return fmt.Errorf("create review: %w", err)
		}
	}

	logLine("Starting review for PR #%d (review_id=%d)", pr.Number, review.ID)

	rc, err := h.store.GetRepoConfig(pr.RepoFullName)
	if err != nil {
		logLine("ERROR: no repo config for %s: %v", pr.RepoFullName, err)
		h.store.UpdateReview(review.ID, map[string]any{
			"status":       "failed",
			"process_logs": logsBuffer.String(),
		})
		return nil
	}
	logLine("Repo config found: cli=%s", rc.CLI)

	exe, err := h.registry.Get(rc.CLI)
	if err != nil {
		logLine("ERROR: executor %q not found: %v", rc.CLI, err)
		h.store.UpdateReview(review.ID, map[string]any{
			"status":       "failed",
			"process_logs": logsBuffer.String(),
		})
		return nil
	}
	logLine("Executor ready: %s", exe.Name())

	if err := h.fetchRemoteBranch(pr, rc); err != nil {
		logLine("ERROR: fetch remote branch: %v", err)
		h.store.UpdateReview(review.ID, map[string]any{
			"status":       "failed",
			"process_logs": logsBuffer.String(),
		})
		return nil
	}
	logLine("Fetched remote branch: %s", pr.HeadBranch)

	worktreePath, err := h.ensureWorktree(pr, rc)
	if err != nil {
		logLine("ERROR: worktree for PR %d: %v", pr.ID, err)
		h.store.UpdateReview(review.ID, map[string]any{
			"status":       "failed",
			"process_logs": logsBuffer.String(),
		})
		return nil
	}
	pr.WorktreePath = worktreePath
	logLine("Worktree ready at %s", worktreePath)

	cmd, err := exe.GetReviewCommand(ctx, pr, rc)
	if err != nil {
		logLine("ERROR: get review command: %v", err)
		h.store.UpdateReview(review.ID, map[string]any{
			"status":       "failed",
			"process_logs": logsBuffer.String(),
		})
		return fmt.Errorf("get review command: %w", err)
	}
	logLine("Review command prepared")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	c := exec.CommandContext(ctx, "sh", "-c", cmd.Command)
	c.Stdin = strings.NewReader(cmd.Prompt)
	c.Dir = cmd.WorkingDir
	c.Env = os.Environ()
	for k, v := range cmd.InjectEnvVars {
		c.Env = append(c.Env, fmt.Sprintf("%s=%s", k, v))
	}
	c.Stdout = &stdout
	c.Stderr = &stderr

	startTime := time.Now()
	logLine("Running executor: %s", exe.Name())

	log.Println("Running executor to review PR: ", pr.URL)
	if err := c.Run(); err != nil {
		log.Println("Error when running executor to review PR: ", pr.URL, err)
		logLine("ERROR: executor run: %v", err)
		logLine("STDOUT: %s", stdout.String())
		logLine("STDERR: %s", stderr.String())
		h.store.UpdateReview(review.ID, map[string]any{
			"status":       "failed",
			"process_logs": logsBuffer.String(),
		})
		return nil
	}

	elapsed := time.Since(startTime)
	logLine("Executor completed in %s", elapsed.Round(time.Second))
	logLine("STDOUT: %s", stdout.String())
	if stderr.Len() > 0 {
		logLine("STDERR: %s", stderr.String())
	}

	result, err := parseReviewResult(stdout.String())
	if err != nil {
		logLine("ERROR: parse review result: %v", err)
		h.store.UpdateReview(review.ID, map[string]any{
			"status":       "failed",
			"process_logs": logsBuffer.String(),
		})
		return nil
	}
	logLine("Parsed result: verdict=%s, comments=%d", result.OverallVerdict, len(result.Comments))

	h.store.UpdateReview(review.ID, map[string]any{
		"summary":         result.Summary,
		"overall_verdict": result.OverallVerdict,
		"status":          "draft",
		"executor_name":   exe.Name(),
		"process_logs":    logsBuffer.String(),
	})

	review.Summary = result.Summary
	review.OverallVerdict = result.OverallVerdict
	review.Status = "draft"
	review.ExecutorName = exe.Name()

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

	reviewData, _ := json.Marshal(review)
	h.wsHub.Publish("pr-updates", reviewData)

	return nil
}

func remoteName(rc *store.RepoConfig) string {
	if rc.RemoteName == "" {
		return "origin"
	}
	return rc.RemoteName
}

func (h *ExecuteReviewHandler) fetchRemoteBranch(pr *store.PullRequest, rc *store.RepoConfig) error {
	c := exec.Command("git", "-C", rc.LocalPath, "fetch", remoteName(rc), pr.HeadBranch)
	if err := c.Run(); err != nil {
		return fmt.Errorf("fetch remote branch: %w", err)
	}
	return nil
}

func (h *ExecuteReviewHandler) ensureWorktree(pr *store.PullRequest, rc *store.RepoConfig) (string, error) {
	if pr.WorktreePath != "" {
		if _, err := os.Stat(pr.WorktreePath); err == nil {
			c := exec.Command("git", "-C", pr.WorktreePath, "fetch", remoteName(rc))
			c.Run()
			c = exec.Command("git", "-C", pr.WorktreePath, "checkout", pr.HeadSHA)
			c.Run()
			return pr.WorktreePath, nil
		}
	}

	worktreePath := fmt.Sprintf("/tmp/pr-reviews/%s/pr-review-%d", rc.RepoFullName, pr.ID)
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
		return &executor.ReviewResult{
			Summary:        strings.TrimSpace(stdout),
			OverallVerdict: "comment",
		}, nil
	}
	return &result, nil
}
