package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GitHub    GitHubConfig    `yaml:"github"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Projects  []string        `yaml:"projects_root"`
	Executors []ExecutorDef   `yaml:"executors"`
	Repos     []RepoMapping   `yaml:"repo_mappings"`
}

type GitHubConfig struct {
	Token string `yaml:"token"`
}

type SchedulerConfig struct {
	FetchInterval            string `yaml:"fetch_interval"`
	CleanupWorktreeAfterDays int    `yaml:"cleanup_worktree_after_days"`
}

type ExecutorDef struct {
	Name    string `yaml:"name"`
	Active  bool   `yaml:"active"`
	Timeout string `yaml:"timeout"`
}

type RepoMapping struct {
	Repo       string  `yaml:"repo"`
	LocalPath  string  `yaml:"local_path"`
	CLI        string  `yaml:"cli"`
	ExtraRules *string `yaml:"extra_rules"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.GitHub.Token == "" {
		cfg.GitHub.Token = os.Getenv("GITHUB_TOKEN")
	}
	if cfg.Scheduler.FetchInterval == "" {
		cfg.Scheduler.FetchInterval = "15m"
	}
	if cfg.Scheduler.CleanupWorktreeAfterDays == 0 {
		cfg.Scheduler.CleanupWorktreeAfterDays = 30
	}
	return &cfg, nil
}

func (c *Config) FetchInterval() time.Duration {
	d, err := time.ParseDuration(c.Scheduler.FetchInterval)
	if err != nil {
		return 15 * time.Minute
	}
	return d
}
