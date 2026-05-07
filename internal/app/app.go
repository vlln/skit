package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vlln/skit/internal/gitfetch"
	"github.com/vlln/skit/internal/search"
	"github.com/vlln/skit/internal/skill"
	"github.com/vlln/skit/internal/source"
)

type Scope string

const (
	Project Scope = "project"
	Global  Scope = "global"
)

type SearchRequest struct {
	Context context.Context
	CWD     string
	Query   string
	Limit   int
	Source  string

	FullDepth bool
}

func ioReadAllLimit(r io.Reader, limit int64) ([]byte, error) {
	limited := io.LimitReader(r, limit+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > limit {
		return nil, fmt.Errorf("response too large")
	}
	return raw, nil
}

type SearchResult struct {
	Name        string `json:"name"`
	Slug        string `json:"slug,omitempty"`
	Source      string `json:"source"`
	SourceName  string `json:"sourceName,omitempty"`
	Install     string `json:"install,omitempty"`
	Path        string `json:"path,omitempty"`
	Description string `json:"description,omitempty"`
	Installs    int    `json:"installs,omitempty"`
}

func Search(req SearchRequest) ([]SearchResult, error) {
	if strings.TrimSpace(req.Source) != "" {
		return searchNamedOrInlineSource(req)
	}
	sources, err := ListSearchSources()
	if err != nil {
		return nil, err
	}
	var out []SearchResult
	var errs []string
	successCount := 0
	for _, src := range sources {
		if !src.Enabled {
			continue
		}
		results, err := searchConfiguredSource(req, src)
		if err != nil {
			errs = append(errs, src.Name+": "+err.Error())
			continue
		}
		successCount++
		out = append(out, results...)
	}
	sortSearchResults(out)
	if req.Limit > 0 && len(out) > req.Limit {
		out = out[:req.Limit]
	}
	if successCount == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return out, nil
}

func searchNamedOrInlineSource(req SearchRequest) ([]SearchResult, error) {
	sourceName := strings.TrimSpace(req.Source)
	sources, err := ListSearchSources()
	if err != nil {
		return nil, err
	}
	for _, src := range sources {
		if src.Name == sourceName {
			return searchConfiguredSource(req, src)
		}
	}
	return searchRepoSource(req, SearchSource{Name: sourceName, Type: "repo", Source: sourceName, Enabled: true})
}

func searchConfiguredSource(req SearchRequest, src SearchSource) ([]SearchResult, error) {
	switch src.Type {
	case "registry":
		return searchRegistrySource(req, src)
	case "repo", "local":
		return searchRepoSource(req, src)
	case "json":
		return searchJSONSource(req, src)
	default:
		return nil, fmt.Errorf("unsupported source type %q", src.Type)
	}
}

func searchRegistrySource(req SearchRequest, src SearchSource) ([]SearchResult, error) {
	base := strings.TrimSpace(src.URL)
	if base == "" {
		base = strings.TrimSpace(src.Source)
	}
	if base == "" {
		return nil, fmt.Errorf("registry URL is required")
	}
	client := search.NewClient()
	client.BaseURL = base
	results, err := client.Search(ctx(req.Context), req.Query, req.Limit)
	if err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(results))
	for _, result := range results {
		out = append(out, SearchResult{
			Name:        result.Name,
				Description: result.Description,
			Slug:       result.Slug,
			Source:     result.Source,
			SourceName: src.Name,
			Installs:   result.Installs,
		})
	}
	return out, nil
}

func searchRepoSource(req SearchRequest, configured SearchSource) ([]SearchResult, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	cwd := req.CWD
	if cwd == "" {
		cwd = "."
	}
	rawSource := configured.Source
	if rawSource == "" {
		rawSource = configured.URL
	}
	src, err := source.Parse(rawSource)
	if err != nil {
		return nil, err
	}
	src.Skill = ""
	workRoot := src.Locator
	cleanup := func() {}
	if isGitProvider(src.Type) {
		if err := resolveTreeSource(ctx(req.Context), &src); err != nil {
			return nil, err
		}
		clone, err := gitfetch.CloneWithOptions(ctx(req.Context), src.URL, src.Ref, tmpRoot(), cloneOptions(src))
		if err != nil {
			return nil, err
		}
		workRoot = clone.Dir
		src.Ref = clone.Ref
		cleanup = func() { _ = gitfetch.RemoveUnder(clone.Dir, tmpRoot()) }
		defer cleanup()
	} else if src.Type != source.Local {
		return nil, fmt.Errorf("provider %q is not implemented yet", src.Type)
	}

	discoverRoot := workRoot
	if src.Subpath != "" {
		discoverRoot = filepath.Join(workRoot, filepath.FromSlash(src.Subpath))
	}
	opts := parseOptionsForSource(src)
	opts.FullDepth = req.FullDepth
	opts.IncludeInternal = false
	opts.IgnoreInvalid = true
	discovered, _, err := skill.DiscoverWithOptions(discoverRoot, opts)
	if err != nil {
		return nil, err
	}

	type scored struct {
		result SearchResult
		score  int
	}
	var matches []scored
	for _, found := range discovered {
		path := relativeSubpath(discoverRoot, found.Root)
		score := searchScore(query, found, path)
		if score <= 0 {
			continue
		}
		matches = append(matches, scored{
			score: score,
			result: SearchResult{
				Name:        found.Name,
				Source:      rawSource,
				SourceName:  configured.Name,
				Install:     installSelector(src, rawSource, found.Name, path),
				Path:        path,
				Description: found.Description,
			},
		})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		if matches[i].result.Name != matches[j].result.Name {
			return matches[i].result.Name < matches[j].result.Name
		}
		return matches[i].result.Path < matches[j].result.Path
	})
	if len(matches) > req.Limit {
		matches = matches[:req.Limit]
	}
	out := make([]SearchResult, 0, len(matches))
	for _, match := range matches {
		out = append(out, match.result)
	}
	return out, nil
}

type catalogFile struct {
	Schema string        `json:"schema"`
	Skills []catalogItem `json:"skills"`
}

type catalogItem struct {
	Name        string   `json:"name"`
	Target      string   `json:"target"`
	Source      string   `json:"source"`
	Install     string   `json:"install"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
}

func searchJSONSource(req SearchRequest, src SearchSource) ([]SearchResult, error) {
	raw, err := readJSONSource(ctx(req.Context), src)
	if err != nil {
		return nil, err
	}
	items, err := parseCatalogItems(raw)
	if err != nil {
		return nil, err
	}
	type scored struct {
		result SearchResult
		score  int
	}
	var matches []scored
	for _, item := range items {
		target := firstNonEmpty(item.Target, item.Install, item.Source)
		if target == "" {
			continue
		}
		score := catalogScore(req.Query, item, target)
		if score <= 0 {
			continue
		}
		matches = append(matches, scored{
			score: score,
			result: SearchResult{
				Name:        item.Name,
				Source:      target,
				SourceName:  src.Name,
				Install:     target,
				Description: item.Description,
			},
		})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].result.Name < matches[j].result.Name
	})
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if len(matches) > limit {
		matches = matches[:limit]
	}
	out := make([]SearchResult, 0, len(matches))
	for _, match := range matches {
		out = append(out, match.result)
	}
	return out, nil
}

func readJSONSource(ctx context.Context, src SearchSource) ([]byte, error) {
	locator := firstNonEmpty(src.URL, src.Source)
	if locator == "" {
		return nil, fmt.Errorf("json source is required")
	}
	if strings.HasPrefix(locator, "http://") || strings.HasPrefix(locator, "https://") {
		httpClient := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, locator, nil)
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
			return nil, fmt.Errorf("json request failed: %s", resp.Status)
		}
		return ioReadAllLimit(resp.Body, 5_000_000)
	}
	return os.ReadFile(locator)
}

func parseCatalogItems(raw []byte) ([]catalogItem, error) {
	var header struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return nil, err
	}
	if header.Schema == manifestSchema {
		var manifest Manifest
		if err := json.Unmarshal(raw, &manifest); err != nil {
			return nil, err
		}
		items := make([]catalogItem, 0, len(manifest.Skills))
		for _, name := range manifestNames(manifest) {
			entry := manifest.Skills[name]
			items = append(items, catalogItem{
				Name:        entry.Name,
				Target:      sourceString(entry.Source),
				Description: entry.Description,
			})
		}
		return items, nil
	}
	var catalog catalogFile
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return nil, err
	}
	return catalog.Skills, nil
}

func catalogScore(query string, item catalogItem, target string) int {
	tokens := strings.Fields(strings.ToLower(query))
	if len(tokens) == 0 {
		return 0
	}
	haystack := strings.ToLower(item.Name + " " + item.Description + " " + target + " " + strings.Join(item.Keywords, " "))
	score := 0
	for _, token := range tokens {
		if !strings.Contains(haystack, token) {
			return 0
		}
		if strings.Contains(strings.ToLower(item.Name), token) {
			score += 5
		}
		if strings.Contains(strings.ToLower(item.Description), token) {
			score += 2
		}
		if strings.Contains(strings.ToLower(target), token) {
			score++
		}
	}
	return score
}

func sortSearchResults(results []SearchResult) {
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Installs != results[j].Installs {
			return results[i].Installs > results[j].Installs
		}
		if results[i].SourceName != results[j].SourceName {
			return results[i].SourceName < results[j].SourceName
		}
		return results[i].Name < results[j].Name
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func searchScore(query string, found skill.Skill, relPath string) int {
	tokens := strings.Fields(strings.ToLower(query))
	if len(tokens) == 0 {
		return 0
	}
	name := strings.ToLower(found.Name)
	description := strings.ToLower(found.Description)
	path := strings.ToLower(relPath)
	haystack := name + " " + description + " " + path
	score := 0
	for _, token := range tokens {
		if !strings.Contains(haystack, token) {
			return 0
		}
		if strings.Contains(name, token) {
			score += 5
		}
		if strings.Contains(description, token) {
			score += 2
		}
		if strings.Contains(path, token) {
			score++
		}
	}
	if name == strings.ToLower(query) {
		score += 10
	}
	return score
}

func installSelector(src source.Source, rawSource, name, relPath string) string {
	rawSource = strings.TrimSpace(rawSource)
	if rawSource == "" {
		return rawSource
	}
	if src.Type == source.Local {
		if relPath == "" {
			return rawSource
		}
		return filepath.ToSlash(filepath.Join(rawSource, filepath.FromSlash(relPath)))
	}
	if name == "" {
		return rawSource
	}
	if strings.HasPrefix(rawSource, "http://") || strings.HasPrefix(rawSource, "https://") || src.Type == source.Git {
		return rawSource + " --skill " + name
	}
	return rawSource + "@" + name
}
