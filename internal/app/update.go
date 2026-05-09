package app

import (
	"context"
	"fmt"

	"github.com/vlln/skit/internal/source"
)

type UpdateRequest struct {
	Context context.Context
	CWD     string
	Scope   Scope
	Name    string
	Agents  []string

	// Deprecated pre-1.0 flag.
	IgnoreDeps bool
}

type UpdateResult = AddResult

func Update(req UpdateRequest) (UpdateResult, error) {
	var result UpdateResult
	manifest, err := readManifest()
	if err != nil {
		return result, err
	}
	names := manifestNames(manifest)
	if req.Name != "" {
		if _, ok := manifest.Skills[req.Name]; !ok {
			return result, fmt.Errorf("%s is not installed", req.Name)
		}
		names = []string{req.Name}
	}
	if len(names) == 0 {
		return result, fmt.Errorf("no installed skills to update")
	}
	for _, name := range names {
		entry := manifest.Skills[name]
		src := sourceString(entry.Source)
		agents := entry.Agents
		if len(req.Agents) > 0 {
			agents = uniqueAgents(req.Agents)
		}
		updated, err := Add(AddRequest{
			Context: req.Context,
			CWD:     req.CWD,
			Scope:   req.Scope,
			Source:  src,
			Name:    entry.Name,
			Skills:  []string{entry.Source.Skill},
			Agents:  agents,
			Force:   true,
		})
		if err != nil {
			return result, err
		}
		result.Entries = append(result.Entries, updated.Entries...)
		result.ActivePaths = append(result.ActivePaths, updated.ActivePaths...)
		result.Warnings = append(result.Warnings, updated.Warnings...)
	}
	return result, nil
}

func sourceString(src ManifestSource) string {
	if src.Type == string(source.Local) {
		return src.Locator
	}
	if src.Type == string(source.GitHub) || src.Type == string(source.GitLab) {
		s := src.Locator
		if src.Subpath != "" {
			s += "/" + src.Subpath
		}
		if src.Ref != "" {
			s += "#" + src.Ref
		}
		if src.Skill != "" {
			s += "@" + src.Skill
		}
		return s
	}
	if src.URL != "" {
		return src.URL
	}
	return src.Locator
}
