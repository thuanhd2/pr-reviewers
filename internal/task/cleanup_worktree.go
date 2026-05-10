package task

import (
	"context"
	"log"
	"os"
	"os/exec"

	"github.com/hibiken/asynq"

	"github.com/thuanho/pr-reviewers/internal/store"
)

const TypeCleanupWorktree = "cleanup:worktree"

type CleanupWorktreeHandler struct {
	store *store.Store
	days  int
}

func NewCleanupWorktreeHandler(s *store.Store, days int) *CleanupWorktreeHandler {
	return &CleanupWorktreeHandler{store: s, days: days}
}

func (h *CleanupWorktreeHandler) Handle(ctx context.Context, t *asynq.Task) error {
	prs, err := h.store.ListPRsForWorktreeCleanup(h.days)
	if err != nil {
		return err
	}

	for _, pr := range prs {
		rc, err := h.store.GetRepoConfig(pr.RepoFullName)
		if err != nil {
			log.Printf("no repo config for %s, force removing worktree dir", pr.RepoFullName)
			os.RemoveAll(pr.WorktreePath)
			h.store.UpdatePRWorktree(pr.ID, "")
			continue
		}

		cmd := exec.Command("git", "-C", rc.LocalPath, "worktree", "remove", "--force", pr.WorktreePath)
		if out, err := cmd.CombinedOutput(); err != nil {
			log.Printf("remove worktree %s: %v (out: %s)", pr.WorktreePath, err, out)
			os.RemoveAll(pr.WorktreePath)
		}

		h.store.UpdatePRWorktree(pr.ID, "")
		log.Printf("cleaned up worktree for PR %d: %s", pr.ID, pr.WorktreePath)
	}

	return nil
}
