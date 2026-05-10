package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/github"
	"github.com/thuanho/pr-reviewers/internal/store"
	"github.com/thuanho/pr-reviewers/internal/ws"
	gh "github.com/google/go-github/v74/github"
)

const TypePostReview = "review:post"

type PostReviewHandler struct {
	store    *store.Store
	ghClient *github.Client
	wsHub    *ws.Hub
}

func NewPostReviewHandler(s *store.Store, ghClient *github.Client, hub *ws.Hub) *PostReviewHandler {
	return &PostReviewHandler{store: s, ghClient: ghClient, wsHub: hub}
}

func (h *PostReviewHandler) Handle(ctx context.Context, t *asynq.Task) error {
	var payload struct {
		ReviewID uint `json:"review_id"`
	}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	review, err := h.store.GetReview(payload.ReviewID)
	if err != nil {
		return fmt.Errorf("get review %d: %w", payload.ReviewID, err)
	}

	pr, err := h.store.GetPR(review.PullRequestID)
	if err != nil {
		return fmt.Errorf("get PR %d: %w", review.PullRequestID, err)
	}

	owner, repoName, _ := strings.Cut(pr.RepoFullName, "/")

	comments := make([]*gh.DraftReviewComment, 0, len(review.Comments))
	for _, c := range review.Comments {
		comments = append(comments, &gh.DraftReviewComment{
			Path:      gh.Ptr(c.FilePath),
			Line:      gh.Ptr(c.LineEnd),
			StartLine: gh.Ptr(c.LineStart),
			Side:      gh.Ptr("RIGHT"),
			Body:      gh.Ptr(c.Body),
		})
	}

	reviewReq := &gh.PullRequestReviewRequest{
		Body:     gh.Ptr(review.Summary),
		Event:    gh.Ptr(review.OverallVerdict),
		CommitID: gh.Ptr(pr.HeadSHA),
		Comments: comments,
	}

	if _, _, err := h.ghClient.GH().PullRequests.CreateReview(ctx, owner, repoName, pr.Number, reviewReq); err != nil {
		return fmt.Errorf("post review: %w", err)
	}

	h.store.UpdateReview(review.ID, map[string]any{"status": "posted"})

	reviewData, _ := json.Marshal(review)
	h.wsHub.Publish("pr-updates", reviewData)

	return nil
}
