package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddListRemoveManifestSkill(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	src := t.TempDir()
	writeTestSkill(t, filepath.Join(src, "SKILL.md"), "demo", "Demo skill")

	added, err := Add(AddRequest{Source: src})
	if err != nil {
		t.Fatal(err)
	}
	if len(added.Entries) != 1 || added.Entries[0].Name != "demo" {
		t.Fatalf("added = %#v", added.Entries)
	}
	if _, err := os.Stat(filepath.Join(installedSkillsRoot(), "demo", "SKILL.md")); err != nil {
		t.Fatalf("installed skill missing: %v", err)
	}
	if len(added.ActivePaths) != 1 {
		t.Fatalf("active paths = %#v", added.ActivePaths)
	}

	listed, err := List(ListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Name != "demo" || listed[0].Missing {
		t.Fatalf("listed = %#v", listed)
	}

	removed, err := Remove(RemoveRequest{Name: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if !removed.Removed || len(removed.Deleted) != 1 {
		t.Fatalf("removed = %#v", removed)
	}
	listed, err = List(ListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 0 {
		t.Fatalf("listed after remove = %#v", listed)
	}
}

func TestAddWithName(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	src := t.TempDir()
	writeTestSkill(t, filepath.Join(src, "SKILL.md"), "upstream-name", "Demo skill")

	added, err := Add(AddRequest{Source: src, Name: "local-name"})
	if err != nil {
		t.Fatal(err)
	}
	if got := added.Entries[0].Name; got != "local-name" {
		t.Fatalf("name = %q", got)
	}
	if _, err := os.Stat(filepath.Join(installedSkillsRoot(), "local-name", "SKILL.md")); err != nil {
		t.Fatalf("installed renamed skill missing: %v", err)
	}
}

func TestCheckReportsMissingActiveLink(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	src := t.TempDir()
	writeTestSkill(t, filepath.Join(src, "SKILL.md"), "demo", "Demo skill")
	added, err := Add(AddRequest{Source: src})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(added.ActivePaths[0]); err != nil {
		t.Fatal(err)
	}
	result, err := Check(DoctorRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Checks) != 1 || result.Checks[0].Code != "missing-active-link" {
		t.Fatalf("checks = %#v", result.Checks)
	}
}

func TestListAllScansSupportedAgentDirs(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	cwd := t.TempDir()

	writeTestSkill(t, filepath.Join(cwd, ".agents", "skills", "project-skill", "SKILL.md"), "project-skill", "Project skill")
	writeTestSkill(t, filepath.Join(home, ".codex", "skills", "global-skill", "SKILL.md"), "global-skill", "Global skill")

	listed, err := List(ListRequest{CWD: cwd})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 0 {
		t.Fatalf("default list = %#v", listed)
	}

	listed, err = List(ListRequest{CWD: cwd, All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 {
		t.Fatalf("all list = %#v", listed)
	}
	byName := map[string]ListEntry{}
	for _, entry := range listed {
		byName[entry.Name] = entry
	}
	if got := byName["project-skill"]; got.Name == "" || got.Managed || got.Scope != "project" || len(got.Agents) != 4 {
		t.Fatalf("project-skill = %#v", got)
	}
	if got := byName["global-skill"]; got.Name == "" || got.Managed || got.Scope != "global" || len(got.Agents) != 1 || got.Agents[0] != "codex" {
		t.Fatalf("global-skill = %#v", got)
	}
}

func TestInitCreatesSkillRepository(t *testing.T) {
	root := t.TempDir()
	result, err := Init(InitRequest{CWD: root, Name: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if result.RepoName != "demo-skill" || result.Name != "demo" {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "demo-skill", "README.md")); err != nil {
		t.Fatalf("README missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "demo-skill", "skills", "demo", "SKILL.md")); err != nil {
		t.Fatalf("SKILL.md missing: %v", err)
	}
}

func TestInitAcceptsRepoNameWithSkillSuffix(t *testing.T) {
	root := t.TempDir()
	result, err := Init(InitRequest{CWD: root, Name: "demo-skill"})
	if err != nil {
		t.Fatal(err)
	}
	if result.RepoName != "demo-skill" || result.Name != "demo" {
		t.Fatalf("result = %#v", result)
	}
}

func writeTestSkill(t *testing.T, path, name, description string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n# " + name + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
