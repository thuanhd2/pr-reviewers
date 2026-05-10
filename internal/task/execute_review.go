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

	rc, err := h.store.GetRepoConfig(pr.RepoFullName)
	if err != nil {
		log.Printf("no repo config for %s: %v", pr.RepoFullName, err)
		h.store.UpdatePRStatus(pr.ID, "failed")
		return nil
	}

	exe, err := h.registry.Get(rc.CLI)
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

	cmd, err := exe.GetReviewCommand(ctx, pr, rc)
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
		ExecutorName:   exe.Name(),
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

	reviewData, _ := json.Marshal(review)
	h.wsHub.Publish("pr-updates", reviewData)

	return nil
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
		return &executor.ReviewResult{
			Summary:        strings.TrimSpace(stdout),
			OverallVerdict: "comment",
		}, nil
	}
	return &result, nil
}
