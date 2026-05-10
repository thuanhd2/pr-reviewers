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

const TypeSyncPRStatus = "github:sync_pr_status"

type SyncPRStatusHandler struct {
	store    *store.Store
	ghClient *github.Client
	wsHub    *ws.Hub
}

func NewSyncPRStatusHandler(s *store.Store, gh *github.Client, hub *ws.Hub) *SyncPRStatusHandler {
	return &SyncPRStatusHandler{store: s, ghClient: gh, wsHub: hub}
}

func (h *SyncPRStatusHandler) Handle(ctx context.Context, t *asynq.Task) error {
	prs, err := h.store.ListOpenPRs()
	if err != nil {
		return err
	}

	for _, pr := range prs {
		detail, err := h.ghClient.GetPR(pr.RepoFullName, pr.Number)
		if err != nil {
			log.Printf("get PR %s#%d: %v", pr.RepoFullName, pr.Number, err)
			continue
		}

		if detail.State == "closed" || detail.Merged {
			if err := h.store.MarkPRClosed(pr.ID); err != nil {
				log.Printf("mark PR %d closed: %v", pr.ID, err)
			}

			payload, _ := json.Marshal(map[string]any{"id": pr.ID, "status": "closed"})
			h.wsHub.Publish("pr-updates", payload)
		}
	}

	return nil
}
