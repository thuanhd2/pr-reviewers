package executor

import (
	"context"
	"time"

	"github.com/thuanho/pr-reviewers/internal/config"
	"github.com/thuanho/pr-reviewers/internal/store"
)

type DeepSeekExecutor struct {
	timeout time.Duration
	cfg     config.DeepSeekConfig
}

func NewDeepSeekExecutor(cfg config.DeepSeekConfig, timeout time.Duration) *DeepSeekExecutor {
	if timeout == 0 {
		timeout = 60 * time.Minute
	}
	return &DeepSeekExecutor{timeout: timeout, cfg: cfg}
}

func (e *DeepSeekExecutor) Name() string { return "deepseek" }

func (e *DeepSeekExecutor) GetReviewCommand(ctx context.Context, pr *store.PullRequest, rc *store.RepoConfig) (*ReviewCommand, error) {
	prompt := BuildReviewPrompt(rc.ExtraRules)

	return &ReviewCommand{
		Command:    "npx -y @anthropic-ai/claude-code@latest -p --dangerously-skip-permissions --output-format json",
		Prompt:     prompt,
		WorkingDir: pr.WorktreePath,
		InjectEnvVars: map[string]string{
			"ANTHROPIC_BASE_URL":             e.cfg.BaseURL,
			"ANTHROPIC_AUTH_TOKEN":           e.cfg.AuthToken,
			"ANTHROPIC_MODEL":                e.cfg.Model,
			"ANTHROPIC_DEFAULT_OPUS_MODEL":   e.cfg.DefaultOpusModel,
			"ANTHROPIC_DEFAULT_SONNET_MODEL": e.cfg.DefaultSonetModel,
			"ANTHROPIC_DEFAULT_HAIKU_MODEL":  e.cfg.DefaultHakuModel,
		},
		Timeout: e.timeout,
	}, nil
}
