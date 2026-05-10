package store

import "time"

type PullRequest struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	GitHubID     string     `json:"github_id" gorm:"uniqueIndex"`
	RepoFullName string     `json:"repo_full_name"`
	Title        string     `json:"title"`
	URL          string     `json:"url"`
	Number       int        `json:"number"`
	Author       string     `json:"author"`
	BaseBranch   string     `json:"base_branch"`
	HeadBranch   string     `json:"head_branch"`
	HeadSHA      string     `json:"head_sha"`
	WorktreePath string     `json:"worktree_path"`
	Status       string     `json:"status" gorm:"default:pending"`
	ClosedAt     *time.Time `json:"closed_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Reviews      []Review   `json:"reviews,omitempty" gorm:"foreignKey:PullRequestID"`
}

type Review struct {
	ID             uint            `json:"id" gorm:"primaryKey"`
	PullRequestID  uint            `json:"pull_request_id" gorm:"index:idx_pr_commit,unique"`
	CommitSHA      string          `json:"commit_sha" gorm:"index:idx_pr_commit,unique"`
	Summary        string          `json:"summary"`
	OverallVerdict string          `json:"overall_verdict"`
	Status         string          `json:"status" gorm:"default:draft"`
	ExecutorName   string          `json:"executor_name"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Comments       []ReviewComment `json:"comments,omitempty" gorm:"foreignKey:ReviewID"`
}

type ReviewComment struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ReviewID  uint      `json:"review_id" gorm:"index"`
	FilePath  string    `json:"file_path"`
	LineStart int       `json:"line_start"`
	LineEnd   int       `json:"line_end"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type RepoConfig struct {
	ID           uint    `json:"id" gorm:"primaryKey"`
	RepoFullName string  `json:"repo_full_name" gorm:"uniqueIndex"`
	LocalPath    string  `json:"local_path"`
	CLI          string  `json:"cli"`
	ExtraRules   *string `json:"extra_rules"`
	Active       bool    `json:"active" gorm:"default:true"`
}

func (RepoConfig) TableName() string { return "repo_configs" }

type CLIConfig struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"uniqueIndex"`
	Description string `json:"description"`
	Active      bool   `json:"active" gorm:"default:true"`
}

func (CLIConfig) TableName() string { return "cli_configs" }
