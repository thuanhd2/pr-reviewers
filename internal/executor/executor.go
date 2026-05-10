package executor

import (
	"context"
	"time"

	"github.com/thuanho/pr-reviewers/internal/store"
)

type Executor interface {
	Name() string
	GetReviewCommand(ctx context.Context, pr *store.PullRequest, rc *store.RepoConfig) (*ReviewCommand, error)
}

type ReviewCommand struct {
	Command       string
	Prompt        string
	WorkingDir    string
	InjectEnvVars map[string]string
	Timeout       time.Duration
}

type ReviewResult struct {
	Summary        string          `json:"summary"`
	OverallVerdict string          `json:"overall_verdict"`
	Comments       []CommentResult `json:"comments"`
}

type CommentResult struct {
	FilePath  string `json:"file_path"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	Body      string `json:"body"`
}
