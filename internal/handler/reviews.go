package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/store"
	"github.com/thuanho/pr-reviewers/internal/task"
)

type ReviewHandler struct {
	store       *store.Store
	asynqClient *asynq.Client
}

func NewReviewHandler(s *store.Store, ac *asynq.Client) *ReviewHandler {
	return &ReviewHandler{store: s, asynqClient: ac}
}

func (h *ReviewHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	review, err := h.store.GetReview(uint(id))
	if err != nil {
		Error(c, http.StatusNotFound, 4004, "review not found")
		return
	}
	Success(c, review)
}

func (h *ReviewHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var body struct {
		Summary        *string `json:"summary"`
		OverallVerdict *string `json:"overall_verdict"`
		Comments       []struct {
			ID   uint   `json:"id"`
			Body string `json:"body"`
		} `json:"comments"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, http.StatusBadRequest, 4000, "invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if body.Summary != nil {
		updates["summary"] = *body.Summary
	}
	if body.OverallVerdict != nil {
		updates["overall_verdict"] = *body.OverallVerdict
	}
	if len(updates) > 0 {
		h.store.UpdateReview(uint(id), updates)
	}

	for _, cmt := range body.Comments {
		h.store.UpdateComment(cmt.ID, cmt.Body)
	}

	review, _ := h.store.GetReview(uint(id))
	Success(c, review)
}

func (h *ReviewHandler) Approve(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	review, err := h.store.GetReview(uint(id))
	if err != nil {
		Error(c, http.StatusNotFound, 4004, "review not found")
		return
	}

	h.store.UpdateReview(review.ID, map[string]interface{}{"status": "approved"})

	payload, _ := json.Marshal(map[string]uint{"review_id": review.ID})
	h.asynqClient.Enqueue(asynq.NewTask(task.TypePostReview, payload))

	Success(c, map[string]string{"status": "review posting"})
}

func (h *ReviewHandler) Rerun(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	review, err := h.store.GetReview(uint(id))
	if err != nil {
		Error(c, http.StatusNotFound, 4004, "review not found")
		return
	}

	h.store.DeleteCommentsForReview(review.ID)
	h.store.UpdateReview(review.ID, map[string]interface{}{
		"summary":         "",
		"overall_verdict": "",
		"process_logs":    "",
		"status":          "draft",
	})

	payload, _ := json.Marshal(map[string]uint{"pr_id": review.PullRequestID})
	h.asynqClient.Enqueue(asynq.NewTask(task.TypeExecuteReview, payload))

	Success(c, map[string]string{"status": "review re-running"})
}

func (h *ReviewHandler) Reject(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	h.store.UpdateReview(uint(id), map[string]interface{}{"status": "rejected"})

	Success(c, map[string]string{"status": "review rejected"})
}
