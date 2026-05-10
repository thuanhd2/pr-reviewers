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
