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

type PRHandler struct {
	store       *store.Store
	asynqClient *asynq.Client
}

func NewPRHandler(s *store.Store, ac *asynq.Client) *PRHandler {
	return &PRHandler{store: s, asynqClient: ac}
}

func (h *PRHandler) List(c *gin.Context) {
	page, perPage := GetPageAndPerPage(c)
	status := c.Query("status")
	repo := c.Query("repo")

	prs, total, err := h.store.ListPRs(status, repo, page, perPage)
	if err != nil {
		Error(c, http.StatusInternalServerError, 5999, "failed to list PRs")
		return
	}

	SuccessList(c, prs, ListMeta{Page: page, PerPage: perPage, Total: total})
}

func (h *PRHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	pr, err := h.store.GetPR(uint(id))
	if err != nil {
		Error(c, http.StatusNotFound, 4004, "pull request not found")
		return
	}
	Success(c, pr)
}

func (h *PRHandler) Refresh(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	pr, err := h.store.GetPR(uint(id))
	if err != nil {
		Error(c, http.StatusNotFound, 4004, "pull request not found")
		return
	}

	payload, _ := json.Marshal(map[string]uint{"pr_id": pr.ID})
	h.asynqClient.Enqueue(asynq.NewTask(task.TypeExecuteReview, payload))

	Success(c, map[string]string{"status": "review queued"})
}

func (h *PRHandler) ListReviews(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	reviews, err := h.store.ListReviewsForPR(uint(id))
	if err != nil {
		Error(c, http.StatusInternalServerError, 5999, "failed to list reviews")
		return
	}
	Success(c, reviews)
}
