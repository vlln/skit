package source

import (
	"strings"
	"testing"
)

func TestParseGitHubShorthand(t *testing.T) {
	src, err := Parse("vercel-labs/agent-skills")
	if err != nil {
		t.Fatal(err)
	}
	if src.Type != GitHub || src.Locator != "vercel-labs/agent-skills" || !src.Implemented {
		t.Fatalf("src = %+v", src)
	}
}

func TestParseGitHubShorthandSkillShortcut(t *testing.T) {
	src, err := Parse("vercel-labs/agent-skills@frontend-design")
	if err != nil {
		t.Fatal(err)
	}
	if src.Skill != "frontend-design" {
		t.Fatalf("skill = %q", src.Skill)
	}
}

func TestParseGitHubShorthandSubpath(t *testing.T) {
	src, err := Parse("vercel-labs/agent-skills/skills/frontend-design")
	if err != nil {
		t.Fatal(err)
	}
	if src.Locator != "vercel-labs/agent-skills" || src.Subpath != "skills/frontend-design" {
		t.Fatalf("src = %+v", src)
	}
}

func TestParseRefSkillShortcut(t *testing.T) {
	src, err := Parse("github:vercel-labs/agent-skills#main@frontend-design")
	if err != nil {
		t.Fatal(err)
	}
	if src.Ref != "main" || src.Skill != "frontend-design" {
		t.Fatalf("src = %+v", src)
	}
}

func TestParseSkillOptionWins(t *testing.T) {
	src, err := Parse("github:vercel-labs/agent-skills#main@frontend-design", WithSkill("skill-creator"))
	if err != nil {
		t.Fatal(err)
	}
	if src.Skill != "skill-creator" {
		t.Fatalf("skill = %q", src.Skill)
	}
	if len(src.Warnings) != 1 {
		t.Fatalf("warnings = %#v", src.Warnings)
	}
}

func TestParseSSHGitDoesNotSplitAt(t *testing.T) {
	src, err := Parse("git@github.com:owner/repo.git")
	if err != nil {
		t.Fatal(err)
	}
	if src.Type != Git || src.Skill != "" || !src.Implemented {
		t.Fatalf("src = %+v", src)
	}
}

func TestParseGitHubTreeURL(t *testing.T) {
	src, err := Parse("https://github.com/owner/repo/tree/main/skills/foo")
	if err != nil {
		t.Fatal(err)
	}
	if src.Type != GitHub || src.UnresolvedTreePath != "main/skills/foo" || src.Ref != "main" || src.Subpath != "skills/foo" {
		t.Fatalf("src = %+v", src)
	}
}

func TestParseGitHubTreeURLKeepsSlashRefCandidate(t *testing.T) {
	src, err := Parse("https://github.com/owner/repo/tree/feature/foo/skills/bar")
	if err != nil {
		t.Fatal(err)
	}
	if src.UnresolvedTreePath != "feature/foo/skills/bar" || src.Ref != "feature" || src.Subpath != "foo/skills/bar" {
		t.Fatalf("src = %+v", src)
	}
}

func TestParseGitLabRecognized(t *testing.T) {
	src, err := Parse("https://gitlab.com/group/repo/-/tree/main/skills/foo")
	if err != nil {
		t.Fatal(err)
	}
	if src.Type != GitLab || !src.Implemented || src.Ref != "main" || src.Subpath != "skills/foo" {
		t.Fatalf("src = %+v", src)
	}
}

func TestParseGitLabShorthand(t *testing.T) {
	src, err := Parse("gitlab:group/subgroup/repo#main@foo")
	if err != nil {
		t.Fatal(err)
	}
	if src.Type != GitLab || !src.Implemented || src.Locator != "group/subgroup/repo" || src.URL != "https://gitlab.com/group/subgroup/repo.git" || src.Subpath != "" || src.Ref != "main" || src.Skill != "foo" {
		t.Fatalf("src = %+v", src)
	}
}

func TestParseGitLabCustomHostRecognized(t *testing.T) {
	src, err := Parse("https://git.example.com/group/sub/repo/-/tree/dev/skills/foo")
	if err != nil {
		t.Fatal(err)
	}
	if src.Type != GitLab || src.Locator != "group/sub/repo" || src.Ref != "dev" || src.Subpath != "skills/foo" {
		t.Fatalf("src = %+v", src)
	}
}

func TestFragmentIgnoredForWellKnownURL(t *testing.T) {
	src, err := Parse("https://example.com/docs#intro")
	if err != nil {
		t.Fatal(err)
	}
	if src.Type != WellKnown || src.URL != "https://example.com/docs#intro" || src.Ref != "" || src.Implemented {
		t.Fatalf("src = %+v", src)
	}
}

func TestParseGenericGitURL(t *testing.T) {
	src, err := Parse("https://example.com/owner/repo.git#main")
	if err != nil {
		t.Fatal(err)
	}
	if src.Type != Git || src.URL != "https://example.com/owner/repo.git" || src.Ref != "main" || !src.Implemented {
		t.Fatalf("src = %+v", src)
	}
}

func TestParseRegistrySource(t *testing.T) {
	src, err := Parse("registry:demo-skill@demo-skill")
	if err != nil {
		t.Fatal(err)
	}
	if src.Type != Registry || src.Locator != "demo-skill" || src.Skill != "demo-skill" || src.Implemented {
		t.Fatalf("src = %+v", src)
	}
}

func TestRejectMultipleHashSelectors(t *testing.T) {
	_, err := Parse("owner/repo#main#other")
	if err == nil || !strings.Contains(err.Error(), "multiple #") {
		t.Fatalf("err = %v", err)
	}
}

func TestRejectTraversal(t *testing.T) {
	_, err := Parse("https://github.com/owner/repo/tree/main/../foo")
	if err == nil {
		t.Fatal("expected traversal error")
	}
}
