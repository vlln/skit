package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

const defaultAPIBase = "https://skills.sh"

type Result struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Source      string `json:"source"`
	Description string `json:"description"`
	Installs    int    `json:"installs"`
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient() Client {
	base := strings.TrimRight(os.Getenv("SKIT_SEARCH_API_URL"), "/")
	if base == "" {
		base = strings.TrimRight(os.Getenv("SKILLS_API_URL"), "/")
	}
	if base == "" {
		base = defaultAPIBase
	}
	return Client{
		BaseURL: base,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c Client) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}
	if limit <= 0 {
		limit = 10
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if !strings.HasSuffix(base, "/api/search") {
		base += "/api/search"
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("q", query)
	q.Set("limit", fmt.Sprintf("%d", limit))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("search request failed: %s", resp.Status)
	}
	var payload struct {
		Skills []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Installs    int    `json:"installs"`
			Source      string `json:"source"`
		} `json:"skills"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(payload.Skills))
	for _, item := range payload.Skills {
		name := strings.TrimSpace(item.Name)
		slug := strings.TrimSpace(item.ID)
		source := strings.TrimSpace(item.Source)
		if name == "" && slug == "" {
			continue
		}
		results = append(results, Result{
			Name:        name,
			Slug:        slug,
			Source:      source,
			Description: strings.TrimSpace(item.Description),
			Installs:    item.Installs,
		})
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Installs > results[j].Installs
	})
	return results, nil
}
