package handler

import (
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/thuanho/pr-reviewers/internal/executor"
	"github.com/thuanho/pr-reviewers/internal/store"
)

type ConfigHandler struct {
	store    *store.Store
	registry *executor.Registry
}

func NewConfigHandler(s *store.Store, reg *executor.Registry) *ConfigHandler {
	return &ConfigHandler{store: s, registry: reg}
}

func (h *ConfigHandler) ListRepos(c *gin.Context) {
	configs, err := h.store.ListRepoConfigs()
	if err != nil {
		Error(c, http.StatusInternalServerError, 5999, "failed to list repo configs")
		return
	}
	Success(c, configs)
}

func (h *ConfigHandler) CreateRepo(c *gin.Context) {
	var cfg store.RepoConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		Error(c, http.StatusBadRequest, 4000, "invalid request body")
		return
	}
	if err := h.store.CreateRepoConfig(&cfg); err != nil {
		Error(c, http.StatusInternalServerError, 5999, "failed to create repo config")
		return
	}
	Success(c, cfg)
}

func (h *ConfigHandler) UpdateRepo(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var cfg store.RepoConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		Error(c, http.StatusBadRequest, 4000, "invalid request body")
		return
	}
	if err := h.store.UpdateRepoConfig(uint(id), &cfg); err != nil {
		Error(c, http.StatusInternalServerError, 5999, "failed to update repo config")
		return
	}
	Success(c, cfg)
}

func (h *ConfigHandler) DeleteRepo(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.store.DeleteRepoConfig(uint(id)); err != nil {
		Error(c, http.StatusInternalServerError, 5999, "failed to delete repo config")
		return
	}
	Success(c, nil)
}

func (h *ConfigHandler) TestConnection(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	configs, _ := h.store.ListRepoConfigs()
	var found *store.RepoConfig
	for _, cfg := range configs {
		if cfg.ID == uint(id) {
			found = &cfg
			break
		}
	}
	if found == nil {
		Error(c, http.StatusNotFound, 4004, "repo config not found")
		return
	}

	remoteName := found.RemoteName
	if remoteName == "" {
		remoteName = "origin"
	}
	cmd := exec.Command("git", "-C", found.LocalPath, "remote", "get-url", remoteName)
	out, err := cmd.Output()
	if err != nil {
		Error(c, http.StatusBadRequest, 5005, "git repo not accessible at local path")
		return
	}

	Success(c, map[string]string{
		"local_path": found.LocalPath,
		"git_remote": strings.TrimSpace(string(out)),
	})
}

func (h *ConfigHandler) ListCLIs(c *gin.Context) {
	names := h.registry.List()
	Success(c, names)
}
