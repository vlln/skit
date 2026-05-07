package app

import (
	"fmt"
	"os"
	"path/filepath"
)

type ExportManifestRequest struct {
	CWD  string
	Path string
}

type ExportManifestResult struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

func ExportManifest(req ExportManifestRequest) (ExportManifestResult, error) {
	manifest, err := readManifest()
	if err != nil {
		return ExportManifestResult{}, err
	}
	path := req.Path
	if path == "" {
		path = filepath.Join(cleanCWD(req.CWD), "skit.json")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(cleanCWD(req.CWD), path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return ExportManifestResult{}, err
	}
	raw, err := manifestJSON(manifest)
	if err != nil {
		return ExportManifestResult{}, err
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		return ExportManifestResult{}, fmt.Errorf("write %s: %w", path, err)
	}
	return ExportManifestResult{Path: path, Count: len(manifest.Skills)}, nil
}
