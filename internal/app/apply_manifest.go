package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type ApplyManifestRequest struct {
	Context context.Context
	Path    string
	Agents  []string
	DryRun  bool
}

func ApplyManifest(req ApplyManifestRequest) (AddResult, error) {
	var result AddResult
	raw, err := readManifestFile(ctx(req.Context), req.Path)
	if err != nil {
		return result, err
	}
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return result, err
	}
	if manifest.Skills == nil {
		return result, nil
	}
	for _, name := range manifestNames(manifest) {
		entry := manifest.Skills[name]
		agents := entry.Agents
		if len(req.Agents) > 0 {
			agents = uniqueAgents(req.Agents)
		}
		if req.DryRun {
			entry.Agents = agents
			result.Entries = append(result.Entries, entry)
			continue
		}
		added, err := Add(AddRequest{
			Context: req.Context,
			Source:  sourceString(entry.Source),
			Name:    entry.Name,
			Skills:  []string{entry.Source.Skill},
			Agents:  agents,
			Force:   true,
		})
		if err != nil {
			return result, err
		}
		result.Entries = append(result.Entries, added.Entries...)
		result.ActivePaths = append(result.ActivePaths, added.ActivePaths...)
		result.Warnings = append(result.Warnings, added.Warnings...)
	}
	return result, nil
}

func readManifestFile(ctx context.Context, path string) ([]byte, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		url := convertGitHubBlobURL(path)
		return fetchManifestURL(ctx, url)
	}
	return os.ReadFile(path)
}

func convertGitHubBlobURL(raw string) string {
	after, ok := strings.CutPrefix(raw, "https://github.com/")
	if !ok {
		return raw
	}
	// owner/repo/blob/ref/path
	parts := strings.SplitN(after, "/", 4)
	if len(parts) < 4 || parts[2] != "blob" {
		return raw
	}
	refPath := strings.SplitN(parts[3], "/", 2)
	ref := refPath[0]
	filePath := ""
	if len(refPath) > 1 {
		filePath = "/" + refPath[1]
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s%s", parts[0], parts[1], ref, filePath)
}

func fetchManifestURL(ctx context.Context, url string) ([]byte, error) {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch %s: %s", url, resp.Status)
	}
	return ioReadAllLimit(resp.Body, 5_000_000)
}
