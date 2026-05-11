package executor

import (
	"context"
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
	prompt := BuildReviewPrompt(pr, rc.ExtraRules)

	return &ReviewCommand{
		Command:       "npx -y @anthropic-ai/claude-code@latest -p --dangerously-skip-permissions --output-format json",
		Prompt:        prompt,
		WorkingDir:    pr.WorktreePath,
		InjectEnvVars: map[string]string{},
		Timeout:       e.timeout,
	}, nil
}
