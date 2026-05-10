package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hibiken/asynq"

	gh "github.com/google/go-github/v74/github"
	"github.com/thuanho/pr-reviewers/internal/github"
	"github.com/thuanho/pr-reviewers/internal/store"
	"github.com/thuanho/pr-reviewers/internal/ws"
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
	log.Println("Starting to post review to github")
	var payload struct {
		ReviewID uint `json:"review_id"`
	}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		log.Printf("Error when unmarshal payload: %v", err)
		return err
	}

	review, err := h.store.GetReview(payload.ReviewID)
	if err != nil {
		log.Printf("Error when get review %d: %v", payload.ReviewID, err)
		return fmt.Errorf("get review %d: %w", payload.ReviewID, err)
	}

	log.Printf("Review found: %+v", review)

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

	log.Printf("Total comments for review %d to post: %d", review.ID, len(comments))

	reviewReq := &gh.PullRequestReviewRequest{
		Body:     gh.Ptr(review.Summary),
		Event:    gh.Ptr(review.OverallVerdict),
		CommitID: gh.Ptr(pr.HeadSHA),
		Comments: comments,
	}

	if _, _, err := h.ghClient.GH().PullRequests.CreateReview(ctx, owner, repoName, pr.Number, reviewReq); err != nil {
		return fmt.Errorf("Error when post review to github: %w", err)
	}

	log.Printf("Review posted to github successfully: %+v", reviewReq)

	h.store.UpdateReview(review.ID, map[string]any{"status": "posted"})

	reviewData, _ := json.Marshal(review)
	h.wsHub.Publish("pr-updates", reviewData)

	return nil
}
