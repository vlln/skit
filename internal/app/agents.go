package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/vlln/skit/internal/store"
	"github.com/vlln/skit/internal/xdg"
)

type agentTarget struct {
	Project string
	Global  func() string
}

var agentTargets = map[string]agentTarget{
	"codex": {
		Project: filepath.Join(".agents", "skills"),
		Global:  func() string { return filepath.Join(codexHome(), "skills") },
	},
	"claude-code": {
		Project: filepath.Join(".claude", "skills"),
		Global:  func() string { return filepath.Join(claudeHome(), "skills") },
	},
	"cursor": {
		Project: filepath.Join(".agents", "skills"),
		Global:  func() string { return filepath.Join(userHome(), ".cursor", "skills") },
	},
	"gemini-cli": {
		Project: filepath.Join(".agents", "skills"),
		Global:  func() string { return filepath.Join(userHome(), ".gemini", "skills") },
	},
	"opencode": {
		Project: filepath.Join(".agents", "skills"),
		Global:  func() string { return filepath.Join(xdg.ConfigHome(), "opencode", "skills") },
	},
}

func activeDirs(paths store.Paths, scope Scope, cwd string, agents []string) ([]string, error) {
	dirs := []string{paths.Active}
	seen := map[string]bool{filepath.Clean(paths.Active): true}
	for _, name := range agents {
		target, ok := agentTargets[name]
		if !ok {
			return nil, fmt.Errorf("unknown agent %q; valid agents: %s", name, validAgentList())
		}
		dir := target.Project
		if scope == Global {
			dir = target.Global()
		} else if !filepath.IsAbs(dir) {
			dir = filepath.Join(cwd, dir)
		}
		clean := filepath.Clean(dir)
		if seen[clean] {
			continue
		}
		seen[clean] = true
		dirs = append(dirs, clean)
	}
	return dirs, nil
}

func pathsForAgent(scope Scope, cwd, name string) (store.Paths, error) {
	paths := store.PathsFor(scope, cwd)
	dir, err := agentActiveDir(scope, cwd, name)
	if err != nil {
		return paths, err
	}
	paths.Active = dir
	paths.Lock = filepath.Join(dir, "skit.lock")
	return paths, nil
}

func agentActiveDir(scope Scope, cwd, name string) (string, error) {
	target, ok := agentTargets[name]
	if !ok {
		return "", fmt.Errorf("unknown agent %q; valid agents: %s", name, validAgentList())
	}
	dir := target.Project
	if scope == Global {
		dir = target.Global()
	} else if !filepath.IsAbs(dir) {
		dir = filepath.Join(cwd, dir)
	}
	return filepath.Clean(dir), nil
}

func validAgentList() string {
	names := make([]string, 0, len(agentTargets))
	for name := range agentTargets {
		names = append(names, name)
	}
	sort.Strings(names)
	return joinComma(names)
}

func joinComma(items []string) string {
	out := ""
	for i, item := range items {
		if i > 0 {
			out += ", "
		}
		out += item
	}
	return out
}

func codexHome() string {
	if home := os.Getenv("CODEX_HOME"); home != "" {
		return home
	}
	return filepath.Join(userHome(), ".codex")
}

func claudeHome() string {
	if home := os.Getenv("CLAUDE_CONFIG_DIR"); home != "" {
		return home
	}
	return filepath.Join(userHome(), ".claude")
}

func userHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}
