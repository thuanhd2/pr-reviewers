package store

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Store struct {
	db *gorm.DB
}

func New(dsn string) (*Store, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) DB() *gorm.DB { return s.db }

// PR methods
func (s *Store) UpsertPR(pr *PullRequest) error {
	return s.db.Where("git_hub_id = ?", pr.GitHubID).Assign(pr).FirstOrCreate(pr).Error
}

func (s *Store) GetPR(id uint) (*PullRequest, error) {
	var pr PullRequest
	err := s.db.Preload("Reviews", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC")
	}).Preload("Reviews.Comments").First(&pr, id).Error
	return &pr, err
}

func (s *Store) ListPRs(status string, repo string, page, perPage int) ([]PullRequest, int64, error) {
	var prs []PullRequest
	var total int64
	q := s.db.Model(&PullRequest{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if repo != "" {
		q = q.Where("repo_full_name = ?", repo)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&prs).Error
	return prs, total, err
}

func (s *Store) ListOpenPRs() ([]PullRequest, error) {
	var prs []PullRequest
	err := s.db.Where("closed_at IS NULL").Find(&prs).Error
	return prs, err
}

func (s *Store) UpdatePRStatus(id uint, status string) error {
	return s.db.Model(&PullRequest{}).Where("id = ?", id).Update("status", status).Error
}

func (s *Store) UpdatePRWorktree(id uint, worktreePath string) error {
	return s.db.Model(&PullRequest{}).Where("id = ?", id).Update("worktree_path", worktreePath).Error
}

func (s *Store) MarkPRClosed(id uint) error {
	return s.db.Model(&PullRequest{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":    "closed",
		"closed_at": gorm.Expr("NOW()"),
	}).Error
}

func (s *Store) ListPRsForWorktreeCleanup(days int) ([]PullRequest, error) {
	var prs []PullRequest
	err := s.db.Where("closed_at IS NOT NULL AND worktree_path != '' AND closed_at < NOW() - INTERVAL '1 day' * ?", days).Find(&prs).Error
	return prs, err
}

// Review methods
func (s *Store) CreateReview(review *Review) error {
	return s.db.Create(review).Error
}

func (s *Store) GetReview(id uint) (*Review, error) {
	var review Review
	err := s.db.Preload("Comments").First(&review, id).Error
	return &review, err
}

func (s *Store) GetLatestReviewForPR(prID uint) (*Review, error) {
	var review Review
	err := s.db.Where("pull_request_id = ?", prID).Order("created_at DESC").First(&review).Error
	return &review, err
}

func (s *Store) ListReviewsForPR(prID uint) ([]Review, error) {
	var reviews []Review
	err := s.db.Where("pull_request_id = ?", prID).Preload("Comments").Order("created_at DESC").Find(&reviews).Error
	return reviews, err
}

func (s *Store) UpdateReview(id uint, updates map[string]interface{}) error {
	return s.db.Model(&Review{}).Where("id = ?", id).Updates(updates).Error
}

func (s *Store) CreateComment(comment *ReviewComment) error {
	return s.db.Create(comment).Error
}

func (s *Store) UpdateComment(id uint, body string) error {
	return s.db.Model(&ReviewComment{}).Where("id = ?", id).Update("body", body).Error
}

func (s *Store) DeleteComment(id uint) error {
	return s.db.Delete(&ReviewComment{}, id).Error
}

func (s *Store) DeleteCommentsForReview(reviewID uint) error {
	return s.db.Where("review_id = ?", reviewID).Delete(&ReviewComment{}).Error
}

func (s *Store) ListHistory(page, perPage int, repo string) ([]Review, int64, error) {
	var reviews []Review
	var total int64
	q := s.db.Model(&Review{}).Joins("JOIN pull_requests ON pull_requests.id = reviews.pull_request_id")
	if repo != "" {
		q = q.Where("pull_requests.repo_full_name = ?", repo)
	}
	q.Count(&total)
	err := q.Preload("Comments").Order("reviews.created_at DESC").Offset((page - 1) * perPage).Limit(perPage).Find(&reviews).Error
	return reviews, total, err
}

// RepoConfig methods
func (s *Store) ListRepoConfigs() ([]RepoConfig, error) {
	var configs []RepoConfig
	err := s.db.Find(&configs).Error
	return configs, err
}

func (s *Store) GetRepoConfig(repoFullName string) (*RepoConfig, error) {
	var cfg RepoConfig
	err := s.db.Where("repo_full_name = ?", repoFullName).First(&cfg).Error
	return &cfg, err
}

func (s *Store) CreateRepoConfig(cfg *RepoConfig) error {
	return s.db.Create(cfg).Error
}

func (s *Store) UpdateRepoConfig(id uint, cfg *RepoConfig) error {
	return s.db.Model(&RepoConfig{}).Where("id = ?", id).Updates(cfg).Error
}

func (s *Store) DeleteRepoConfig(id uint) error {
	return s.db.Delete(&RepoConfig{}, id).Error
}

// CLIConfig methods
func (s *Store) ListCLIConfigs() ([]CLIConfig, error) {
	var configs []CLIConfig
	err := s.db.Find(&configs).Error
	return configs, err
}

// SchedulerJob methods
func (s *Store) UpsertSchedulerJob(job *SchedulerJob) error {
	return s.db.Where("job_name = ?", job.JobName).Assign(job).FirstOrCreate(job).Error
}

func (s *Store) ListSchedulerJobs() ([]SchedulerJob, error) {
	var jobs []SchedulerJob
	err := s.db.Order("job_name ASC").Find(&jobs).Error
	return jobs, err
}

func (s *Store) SetSchedulerJobRunning(jobName string) error {
	return s.db.Model(&SchedulerJob{}).Where("job_name = ?", jobName).Update("status", "running").Error
}

func (s *Store) RecordSchedulerJobEnd(jobName string, status string, lastError *string, lastRun time.Time, nextRun *time.Time) error {
	updates := map[string]interface{}{
		"status":      status,
		"last_run_at": lastRun,
	}
	if lastError != nil {
		updates["last_error"] = *lastError
	} else {
		updates["last_error"] = nil
	}
	if nextRun != nil && !nextRun.IsZero() {
		updates["next_run_at"] = nextRun
	}
	return s.db.Model(&SchedulerJob{}).Where("job_name = ?", jobName).Updates(updates).Error
}

func (s *Store) SeedCLIConfigs() error {
	configs := []CLIConfig{
		{Name: "claude-code", Description: "Claude Code CLI agent", Active: true},
	}
	for _, c := range configs {
		s.db.Where("name = ?", c.Name).FirstOrCreate(&c)
	}
	return nil
}
