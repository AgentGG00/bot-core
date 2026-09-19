package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Release is a single stable (non-draft, non-prerelease) GitHub release.
type Release struct {
	TagName     string
	Name        string
	Body        string
	HTMLURL     string
	PublishedAt time.Time
}

type Client struct {
	HTTPClient *http.Client
}

func New() *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type apiRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
	Prerelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
}

// LatestStable returns the most recent stable release for "owner/repo", or
// nil if the repo has no stable (non-draft, non-prerelease) release.
func (c *Client) LatestStable(repo string) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases", repo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("github: build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github: unexpected status %d for %s", resp.StatusCode, repo)
	}

	var releases []apiRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("github: decode response: %w", err)
	}

	for _, r := range releases {
		if r.Draft || r.Prerelease {
			continue
		}
		return &Release{
			TagName:     r.TagName,
			Name:        r.Name,
			Body:        r.Body,
			HTMLURL:     r.HTMLURL,
			PublishedAt: r.PublishedAt,
		}, nil
	}
	return nil, nil
}
