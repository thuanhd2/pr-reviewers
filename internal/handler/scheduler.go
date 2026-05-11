package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/thuanho/pr-reviewers/internal/store"
)

type SchedulerHandler struct {
	store *store.Store
}

func NewSchedulerHandler(s *store.Store) *SchedulerHandler {
	return &SchedulerHandler{store: s}
}

func (h *SchedulerHandler) ListJobs(c *gin.Context) {
	jobs, err := h.store.ListSchedulerJobs()
	if err != nil {
		Error(c, http.StatusInternalServerError, 5000, "failed to list scheduler jobs")
		return
	}

	Success(c, jobs)
}
