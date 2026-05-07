package app

import "fmt"

type InstallRequest struct {
	CWD    string
	Scope  Scope
	Agents []string
}

type InstallResult struct {
	ActivePaths []string `json:"activePaths,omitempty"`
}

func Install(req InstallRequest) (InstallResult, error) {
	return InstallResult{}, fmt.Errorf("install requires a source or manifest file")
}
