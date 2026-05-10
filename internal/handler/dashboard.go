package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/thuanho/pr-reviewers/internal/store"
)

type DashboardHandler struct {
	store *store.Store
}

func NewDashboardHandler(s *store.Store) *DashboardHandler {
	return &DashboardHandler{store: s}
}

func (h *DashboardHandler) Get(c *gin.Context) {
	var counts struct {
		Pending   int64 `json:"pending"`
		Reviewing int64 `json:"reviewing"`
		Drafted   int64 `json:"drafted"`
		Posted    int64 `json:"posted"`
		Failed    int64 `json:"failed"`
	}

	db := h.store.DB()
	db.Model(&store.PullRequest{}).Where("status = 'pending'").Count(&counts.Pending)
	db.Model(&store.PullRequest{}).Where("status = 'reviewing'").Count(&counts.Reviewing)
	db.Model(&store.PullRequest{}).Where("status = 'drafted'").Count(&counts.Drafted)
	db.Model(&store.PullRequest{}).Where("status = 'posted'").Count(&counts.Posted)
	db.Model(&store.PullRequest{}).Where("status = 'failed'").Count(&counts.Failed)

	var recent []store.Review
	db.Preload("Comments").Order("created_at DESC").Limit(10).Find(&recent)

	Success(c, map[string]interface{}{
		"counts": counts,
		"recent": recent,
	})
}

func (h *DashboardHandler) History(c *gin.Context) {
	page, perPage := GetPageAndPerPage(c)
	repo := c.Query("repo")

	reviews, total, err := h.store.ListHistory(page, perPage, repo)
	if err != nil {
		Error(c, http.StatusInternalServerError, 5999, "failed to list history")
		return
	}

	SuccessList(c, reviews, ListMeta{Page: page, PerPage: perPage, Total: total})
}
