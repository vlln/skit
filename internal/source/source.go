package source

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Type string

const (
	Local     Type = "local"
	GitHub    Type = "github"
	GitLab    Type = "gitlab"
	Git       Type = "git"
	WellKnown Type = "well-known"
	Registry  Type = "registry"
)

type Source struct {
	Type               Type
	Locator            string
	URL                string
	Ref                string
	UnresolvedTreePath string
	Subpath            string
	Skill              string
	Implemented        bool
	Warnings           []string
}

var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var ownerRepoPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(?:/.+)?$`)

func Parse(input string, opts ...Option) (Source, error) {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return Source{}, fmt.Errorf("source is required")
	}

	base, refSelector, hasRef, err := splitFragment(input)
	if err != nil {
		return Source{}, err
	}
	inlineSkill := ""
	if hasRef {
		refSelector, inlineSkill = splitRefSkill(refSelector)
	}

	src, err := parseBase(base)
	if err != nil {
		return Source{}, err
	}
	if hasRef {
		src.Ref = refSelector
		if inlineSkill != "" {
			src.Skill = inlineSkill
		}
	}
	if cfg.skill != "" {
		if src.Skill != "" && src.Skill != cfg.skill {
			src.Warnings = append(src.Warnings, "inline skill selector ignored because --skill was provided")
		}
		src.Skill = cfg.skill
	}
	if src.Subpath != "" && unsafeSubpath(src.Subpath) {
		return Source{}, fmt.Errorf("unsafe source subpath %q", src.Subpath)
	}
	return src, nil
}

type Option func(*config)

type config struct {
	skill string
}

func WithSkill(skill string) Option {
	return func(c *config) {
		c.skill = skill
	}
}

func parseBase(base string) (Source, error) {
	if src, ok, err := parseLocal(base); ok || err != nil {
		return src, err
	}
	if strings.HasPrefix(base, "github:") {
		return parseGitHubShorthand(strings.TrimPrefix(base, "github:"))
	}
	if strings.HasPrefix(base, "gitlab:") {
		return parseGitLabShorthand(strings.TrimPrefix(base, "gitlab:"))
	}
	if strings.HasPrefix(base, "registry:") {
		return parseRegistry(strings.TrimPrefix(base, "registry:"))
	}
	if strings.HasPrefix(base, "clawhub:") {
		return parseRegistry(strings.TrimPrefix(base, "clawhub:"))
	}
	if strings.HasPrefix(base, "http://") || strings.HasPrefix(base, "https://") {
		return parseURL(base)
	}
	if isSSHGit(base) {
		return Source{Type: Git, Locator: base, URL: base, Implemented: true}, nil
	}
	return parseGitHubShorthand(base)
}

func parseLocal(input string) (Source, bool, error) {
	if input == "." || strings.HasPrefix(input, "./") || strings.HasPrefix(input, "../") || filepath.IsAbs(input) {
		abs, err := filepath.Abs(input)
		if err != nil {
			return Source{}, true, err
		}
		return Source{Type: Local, Locator: abs, Implemented: true}, true, nil
	}
	if _, err := os.Stat(input); err == nil {
		abs, err := filepath.Abs(input)
		if err != nil {
			return Source{}, true, err
		}
		return Source{Type: Local, Locator: abs, Implemented: true}, true, nil
	}
	return Source{}, false, nil
}

func parseGitHubShorthand(raw string) (Source, error) {
	locator, skill := splitLocatorSkill(raw)
	if !ownerRepoPattern.MatchString(locator) {
		return Source{}, fmt.Errorf("unsupported source %q", raw)
	}
	parts := strings.Split(locator, "/")
	owner, repo := parts[0], parts[1]
	if owner == "" || repo == "" {
		return Source{}, fmt.Errorf("empty GitHub owner or repo")
	}
	subpath := ""
	if len(parts) > 2 {
		subpath = strings.Join(parts[2:], "/")
		if unsafeSubpath(subpath) {
			return Source{}, fmt.Errorf("unsafe source subpath %q", subpath)
		}
	}
	return Source{
		Type:        GitHub,
		Locator:     owner + "/" + repo,
		URL:         "https://github.com/" + owner + "/" + repo + ".git",
		Subpath:     subpath,
		Skill:       skill,
		Implemented: true,
	}, nil
}

func parseGitLabShorthand(raw string) (Source, error) {
	locator, skill := splitLocatorSkill(raw)
	if !ownerRepoPattern.MatchString(locator) {
		return Source{}, fmt.Errorf("unsupported source %q", raw)
	}
	return Source{
		Type:        GitLab,
		Locator:     locator,
		URL:         "https://gitlab.com/" + locator + ".git",
		Skill:       skill,
		Implemented: true,
	}, nil
}

func parseRegistry(raw string) (Source, error) {
	locator, skill := splitRegistrySkill(raw)
	locator = strings.TrimSpace(locator)
	if locator == "" {
		return Source{}, fmt.Errorf("registry locator is required")
	}
	return Source{Type: Registry, Locator: locator, Skill: skill, Implemented: false}, nil
}

func splitRegistrySkill(raw string) (string, string) {
	last := strings.LastIndex(raw, "@")
	if last <= 0 {
		return raw, ""
	}
	skill := raw[last+1:]
	if skillNamePattern.MatchString(skill) {
		return raw[:last], skill
	}
	return raw, ""
}

func parseURL(raw string) (Source, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Source{}, err
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return Source{}, fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
	host := strings.ToLower(u.Hostname())
	switch host {
	case "github.com":
		return parseGitHubURL(u)
	case "gitlab.com":
		return parseGitLabURL(u)
	default:
		if strings.Contains(u.Path, "/-/tree/") {
			return parseGitLabURL(u)
		}
		if isGenericGitURL(u) {
			return Source{Type: Git, Locator: raw, URL: raw, Implemented: true}, nil
		}
		return Source{Type: WellKnown, Locator: raw, URL: raw, Implemented: false}, nil
	}
}

func parseGitHubURL(u *url.URL) (Source, error) {
	parts := cleanParts(u.Path)
	if len(parts) < 2 {
		return Source{}, fmt.Errorf("empty GitHub owner or repo")
	}
	src := Source{
		Type:        GitHub,
		Locator:     parts[0] + "/" + trimGitSuffix(parts[1]),
		URL:         "https://github.com/" + parts[0] + "/" + trimGitSuffix(parts[1]) + ".git",
		Implemented: true,
	}
	if len(parts) >= 4 && parts[2] == "tree" {
		src.UnresolvedTreePath = strings.Join(parts[3:], "/")
		if unsafeSubpath(src.UnresolvedTreePath) {
			return Source{}, fmt.Errorf("unsafe GitHub tree path %q", src.UnresolvedTreePath)
		}
		src.Ref = parts[3]
		if len(parts) > 4 {
			src.Subpath = strings.Join(parts[4:], "/")
		}
	}
	return src, nil
}

func parseGitLabURL(u *url.URL) (Source, error) {
	parts := cleanParts(u.Path)
	tree := indexPart(parts, "-")
	if tree >= 0 && tree+2 < len(parts) && parts[tree+1] == "tree" {
		locatorParts := parts[:tree]
		if len(locatorParts) == 0 {
			return Source{}, fmt.Errorf("empty GitLab locator")
		}
		treePath := strings.Join(parts[tree+2:], "/")
		if unsafeSubpath(treePath) {
			return Source{}, fmt.Errorf("unsafe GitLab tree path %q", treePath)
		}
		locator := strings.Join(locatorParts, "/")
		return Source{Type: GitLab, Locator: locator, URL: u.Scheme + "://" + u.Host + "/" + locator + ".git", Ref: parts[tree+2], Subpath: strings.Join(parts[tree+3:], "/"), UnresolvedTreePath: treePath, Implemented: true}, nil
	}
	if len(parts) == 0 {
		return Source{}, fmt.Errorf("empty GitLab locator")
	}
	locator := strings.Join(parts, "/")
	return Source{Type: GitLab, Locator: locator, URL: u.Scheme + "://" + u.Host + "/" + locator + ".git", Implemented: true}, nil
}

func splitFragment(input string) (string, string, bool, error) {
	hash := strings.Index(input, "#")
	if hash < 0 {
		return input, "", false, nil
	}
	if strings.Count(input, "#") > 1 {
		return "", "", false, fmt.Errorf("source contains multiple # selectors")
	}
	base := input[:hash]
	fragment := input[hash+1:]
	if fragment == "" || !looksGitLike(base) {
		return input, "", false, nil
	}
	return base, fragment, true, nil
}

func looksGitLike(input string) bool {
	if strings.HasPrefix(input, "github:") || strings.HasPrefix(input, "gitlab:") || strings.HasPrefix(input, "git@") {
		return true
	}
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		u, err := url.Parse(input)
		if err != nil {
			return false
		}
		host := strings.ToLower(u.Hostname())
		return host == "github.com" || host == "gitlab.com" || strings.HasSuffix(u.Path, ".git") || strings.Contains(u.Path, "/-/tree/")
	}
	return !strings.Contains(input, ":") && !strings.HasPrefix(input, ".") && !strings.HasPrefix(input, "/") && ownerRepoPattern.MatchString(input)
}

func splitRefSkill(selector string) (string, string) {
	before, after, ok := strings.Cut(selector, "@")
	if !ok {
		return selector, ""
	}
	last := strings.LastIndex(selector, "@")
	ref := selector[:last]
	skill := selector[last+1:]
	if skillNamePattern.MatchString(skill) {
		return ref, skill
	}
	return before + "@" + after, ""
}

func splitLocatorSkill(raw string) (string, string) {
	last := strings.LastIndex(raw, "@")
	if last <= 0 {
		return raw, ""
	}
	skill := raw[last+1:]
	locator := raw[:last]
	if skillNamePattern.MatchString(skill) && ownerRepoPattern.MatchString(locator) {
		return locator, skill
	}
	return raw, ""
}

func isSSHGit(s string) bool {
	return strings.Contains(s, "@") && strings.Contains(s, ":") && (strings.HasSuffix(s, ".git") || strings.HasPrefix(s, "git@"))
}

func cleanParts(path string) []string {
	raw := strings.Split(strings.Trim(path, "/"), "/")
	out := raw[:0]
	for _, part := range raw {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func indexPart(parts []string, want string) int {
	for i, part := range parts {
		if part == want {
			return i
		}
	}
	return -1
}

func trimGitSuffix(repo string) string {
	return strings.TrimSuffix(repo, ".git")
}

func isGenericGitURL(u *url.URL) bool {
	return strings.HasSuffix(u.Path, ".git")
}

func unsafeSubpath(path string) bool {
	if path == "" || strings.HasPrefix(path, "/") || filepath.IsAbs(path) {
		return true
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." || part == ".." {
			return true
		}
	}
	return strings.Contains(path, "\\")
}
