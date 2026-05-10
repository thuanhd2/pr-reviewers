package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	gh "github.com/google/go-github/v74/github"
	"golang.org/x/oauth2"
)

type Client struct {
	client *gh.Client
}

func NewClient(token string) *Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := &http.Client{
		Timeout: 180 * time.Second,
		Transport: &oauth2.Transport{
			Source: ts,
		},
	}
	return &Client{client: gh.NewClient(tc)}
}

// GH returns the underlying go-github client for operations beyond search/get.
func (c *Client) GH() *gh.Client { return c.client }

type SearchPR struct {
	NodeID  string `json:"node_id"`
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
	Number  int    `json:"number"`
	User    struct {
		Login string `json:"login"`
	} `json:"user"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
	Head struct {
		Ref  string `json:"ref"`
		SHA  string `json:"sha"`
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"head"`
}

func (c *Client) SearchAssignedPRs() ([]SearchPR, error) {
	ctx := context.Background()
	query := "is:pr is:open review-requested:@me"

	result, _, err := c.client.Search.Issues(ctx, query, &gh.SearchOptions{
		ListOptions: gh.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, fmt.Errorf("github search: %w", err)
	}

	var prs []SearchPR
	for _, issue := range result.Issues {
		owner, repo := parseIssueRepo(issue)
		if owner == "" || repo == "" {
			continue
		}

		pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, issue.GetNumber())
		if err != nil {
			continue
		}

		prs = append(prs, SearchPR{
			NodeID:  issue.GetNodeID(),
			Title:   issue.GetTitle(),
			HTMLURL: issue.GetHTMLURL(),
			Number:  issue.GetNumber(),
			User: struct {
				Login string `json:"login"`
			}{
				Login: issue.GetUser().GetLogin(),
			},
			Base: struct {
				Ref string `json:"ref"`
			}{
				Ref: pr.GetBase().GetRef(),
			},
			Head: struct {
				Ref  string `json:"ref"`
				SHA  string `json:"sha"`
				Repo struct {
					FullName string `json:"full_name"`
				} `json:"repo"`
			}{
				Ref: pr.GetHead().GetRef(),
				SHA: pr.GetHead().GetSHA(),
				Repo: struct {
					FullName string `json:"full_name"`
				}{
					FullName: pr.GetHead().GetRepo().GetFullName(),
				},
			},
		})
	}
	return prs, nil
}

func parseIssueRepo(issue *gh.Issue) (owner, repo string) {
	url := strings.TrimRight(issue.GetRepositoryURL(), "/")
	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2], parts[len(parts)-1]
	}
	return "", ""
}

type PRDetail struct {
	State  string `json:"state"`
	Merged bool   `json:"merged"`
	Head   struct {
		SHA string `json:"sha"`
	} `json:"head"`
}

func (c *Client) GetPR(repoFullName string, number int) (*PRDetail, error) {
	owner, repoName, ok := strings.Cut(repoFullName, "/")
	if !ok {
		return nil, fmt.Errorf("invalid repoFullName: %s", repoFullName)
	}

	pr, _, err := c.client.PullRequests.Get(context.Background(), owner, repoName, number)
	if err != nil {
		return nil, err
	}

	return &PRDetail{
		State:  pr.GetState(),
		Merged: pr.GetMerged(),
		Head: struct {
			SHA string `json:"sha"`
		}{
			SHA: pr.GetHead().GetSHA(),
		},
	}, nil
}
