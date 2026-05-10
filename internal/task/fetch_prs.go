package task

import (
	"context"
	"encoding/json"
	"log"

	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/github"
	"github.com/thuanho/pr-reviewers/internal/store"
	"github.com/thuanho/pr-reviewers/internal/ws"
)

const TypeFetchAssignedPRs = "github:fetch_assigned_prs"

type FetchPRsHandler struct {
	store       *store.Store
	ghClient    *github.Client
	wsHub       *ws.Hub
	asynqClient *asynq.Client
}

func NewFetchPRsHandler(s *store.Store, gh *github.Client, hub *ws.Hub, ac *asynq.Client) *FetchPRsHandler {
	return &FetchPRsHandler{store: s, ghClient: gh, wsHub: hub, asynqClient: ac}
}

func (h *FetchPRsHandler) Handle(ctx context.Context, t *asynq.Task) error {
	searchPRs, err := h.ghClient.SearchAssignedPRs()
	if err != nil {
		return err
	}

	for _, sp := range searchPRs {
		pr := &store.PullRequest{
			GitHubID:     sp.NodeID,
			RepoFullName: sp.Head.Repo.FullName,
			Title:        sp.Title,
			URL:          sp.HTMLURL,
			Number:       sp.Number,
			Author:       sp.User.Login,
			BaseBranch:   sp.Base.Ref,
			HeadBranch:   sp.Head.Ref,
			HeadSHA:      sp.Head.SHA,
			Status:       "pending",
		}

		if err := h.store.UpsertPR(pr); err != nil {
			log.Printf("upsert PR %s: %v", sp.NodeID, err)
			continue
		}

		latestReview, _ := h.store.GetLatestReviewForPR(pr.ID)
		if latestReview == nil || latestReview.CommitSHA != sp.Head.SHA {
			payload, _ := json.Marshal(map[string]uint{"pr_id": pr.ID})
			h.asynqClient.Enqueue(asynq.NewTask(TypeExecuteReview, payload))
		}

		prData, _ := json.Marshal(pr)
		h.wsHub.Publish("pr-updates", prData)
	}

	return nil
}
