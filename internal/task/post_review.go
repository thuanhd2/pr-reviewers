package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/store"
	"github.com/thuanho/pr-reviewers/internal/ws"
)

const TypePostReview = "review:post"

type PostReviewHandler struct {
	store   *store.Store
	ghToken string
	wsHub   *ws.Hub
}

func NewPostReviewHandler(s *store.Store, ghToken string, hub *ws.Hub) *PostReviewHandler {
	return &PostReviewHandler{store: s, ghToken: ghToken, wsHub: hub}
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

	parts := strings.SplitN(pr.RepoFullName, "/", 2)
	owner, repoName := parts[0], parts[1]
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d/reviews", owner, repoName, pr.Number)

	comments := make([]map[string]any, 0, len(review.Comments))
	for _, c := range review.Comments {
		comments = append(comments, map[string]any{
			"path":       c.FilePath,
			"line":       c.LineEnd,
			"start_line": c.LineStart,
			"side":       "RIGHT",
			"body":       c.Body,
		})
	}

	body := map[string]any{
		"body":      review.Summary,
		"event":     review.OverallVerdict,
		"commit_id": pr.HeadSHA,
		"comments":  comments,
	}

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	req.Header.Set("Authorization", "Bearer "+h.ghToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("post review: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("post review returned %d", resp.StatusCode)
	}

	h.store.UpdateReview(review.ID, map[string]any{"status": "posted"})
	h.store.UpdatePRStatus(pr.ID, "posted")

	reviewData, _ := json.Marshal(review)
	h.wsHub.Publish("pr-updates", reviewData)

	return nil
}
