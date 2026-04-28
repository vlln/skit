package app

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vlln/skit/internal/gitfetch"
	"github.com/vlln/skit/internal/search"
	"github.com/vlln/skit/internal/skill"
	"github.com/vlln/skit/internal/source"
	"github.com/vlln/skit/internal/store"
)

type Scope = store.Scope

const (
	Project = store.Project
	Global  = store.Global
)

type SearchRequest struct {
	Context context.Context
	CWD     string
	Query   string
	Limit   int
	Source  string

	FullDepth bool
}

type SearchResult struct {
	Name        string `json:"name"`
	Slug        string `json:"slug,omitempty"`
	Source      string `json:"source"`
	Install     string `json:"install,omitempty"`
	Path        string `json:"path,omitempty"`
	Description string `json:"description,omitempty"`
	Installs    int    `json:"installs,omitempty"`
}

func Search(req SearchRequest) ([]SearchResult, error) {
	if strings.TrimSpace(req.Source) != "" {
		return searchSource(req)
	}
	client := search.NewClient()
	results, err := client.Search(ctx(req.Context), req.Query, req.Limit)
	if err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(results))
	for _, result := range results {
		out = append(out, SearchResult{
			Name:     result.Name,
			Slug:     result.Slug,
			Source:   result.Source,
			Installs: result.Installs,
		})
	}
	return out, nil
}

func searchSource(req SearchRequest) ([]SearchResult, error) {
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
	paths := store.PathsFor(Project, cwd)
	src, err := source.Parse(req.Source)
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
		clone, err := gitfetch.CloneWithOptions(ctx(req.Context), src.URL, src.Ref, paths.Tmp, cloneOptions(src))
		if err != nil {
			return nil, err
		}
		workRoot = clone.Dir
		src.Ref = clone.Ref
		cleanup = func() { _ = gitfetch.RemoveUnder(clone.Dir, paths.Tmp) }
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
				Source:      req.Source,
				Install:     installSelector(src, req.Source, found.Name, path),
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
