package app

import (
	"os"
	"path/filepath"
	"strings"
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

func TestUpdateReinstallsFromSource(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	src := t.TempDir()
	writeTestSkill(t, filepath.Join(src, "SKILL.md"), "demo", "Before update")
	if _, err := Add(AddRequest{Source: src}); err != nil {
		t.Fatal(err)
	}
	writeTestSkill(t, filepath.Join(src, "SKILL.md"), "demo", "After update")
	result, err := Update(UpdateRequest{Name: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 1 || result.Entries[0].Description != "After update" {
		t.Fatalf("updated skill description = %q", result.Entries[0].Description)
	}
}

func TestUpdateAllSkills(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	src1 := t.TempDir()
	writeTestSkill(t, filepath.Join(src1, "SKILL.md"), "alpha", "Alpha")
	if _, err := Add(AddRequest{Source: src1}); err != nil {
		t.Fatal(err)
	}
	src2 := t.TempDir()
	writeTestSkill(t, filepath.Join(src2, "SKILL.md"), "beta", "Beta")
	if _, err := Add(AddRequest{Source: src2, Name: "beta"}); err != nil {
		t.Fatal(err)
	}
	result, err := Update(UpdateRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 2 {
		t.Fatalf("updated = %d", len(result.Entries))
	}
}

func TestUpdateNonExistentSkill(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	_, err := Update(UpdateRequest{Name: "nope"})
	if err == nil {
		t.Fatal("expected error updating non-existent skill")
	}
}

func TestUpdateNoSkillsInstalled(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	_, err := Update(UpdateRequest{})
	if err == nil || err.Error() != "no installed skills to update" {
		t.Fatalf("expected 'no installed skills to update', got %v", err)
	}
}

func TestExportManifestToDefaultPath(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	cwd := t.TempDir()

	src := t.TempDir()
	writeTestSkill(t, filepath.Join(src, "SKILL.md"), "demo", "Demo skill")
	if _, err := Add(AddRequest{Source: src}); err != nil {
		t.Fatal(err)
	}
	result, err := ExportManifest(ExportManifestRequest{CWD: cwd})
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(cwd, "skit.json")
	if result.Path != expected || result.Count != 1 {
		t.Fatalf("export result = %#v", result)
	}
	raw, err := os.ReadFile(expected)
	if err != nil {
		t.Fatalf("skit.json missing: %v", err)
	}
	if !strings.Contains(string(raw), `"demo"`) || !strings.Contains(string(raw), `"skit.manifest/v1"`) {
		t.Fatalf("skit.json content = %s", raw)
	}
}

func TestExportManifestToCustomPath(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	src := t.TempDir()
	writeTestSkill(t, filepath.Join(src, "SKILL.md"), "demo", "Demo skill")
	if _, err := Add(AddRequest{Source: src}); err != nil {
		t.Fatal(err)
	}
	custom := filepath.Join(t.TempDir(), "my-manifest.json")
	result, err := ExportManifest(ExportManifestRequest{Path: custom})
	if err != nil {
		t.Fatal(err)
	}
	if result.Path != custom {
		t.Fatalf("export path = %s, want %s", result.Path, custom)
	}
}

func TestRemoveKeepLeavesSkillOnDisk(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	src := t.TempDir()
	writeTestSkill(t, filepath.Join(src, "SKILL.md"), "demo", "Demo skill")
	if _, err := Add(AddRequest{Source: src}); err != nil {
		t.Fatal(err)
	}
	installDir := filepath.Join(installedSkillsRoot(), "demo")
	result, err := Remove(RemoveRequest{Name: "demo", Keep: true})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Removed || len(result.Deleted) != 0 {
		t.Fatalf("remove keep result = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(installDir, "SKILL.md")); err != nil {
		t.Fatalf("install dir should still exist: %v", err)
	}
	listed, _ := List(ListRequest{})
	if len(listed) != 0 {
		t.Fatalf("should not be listed in manifest: %#v", listed)
	}
}

func TestSearchLocalSource(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	src := t.TempDir()
	writeTestSkill(t, filepath.Join(src, "SKILL.md"), "search-demo", "A test skill for searching")
	result, err := Search(SearchRequest{Query: "search", Source: src})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Name != "search-demo" || result[0].Description != "A test skill for searching" {
		t.Fatalf("search result = %#v", result)
	}
}

func TestSearchLocalSourceNoMatch(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	src := t.TempDir()
	writeTestSkill(t, filepath.Join(src, "SKILL.md"), "demo", "A demo skill")
	result, err := Search(SearchRequest{Query: "zzz-no-match", Source: src})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected no results, got %#v", result)
	}
}

func TestInstallDuplicateOverwritesWithForce(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	src := t.TempDir()
	writeTestSkill(t, filepath.Join(src, "SKILL.md"), "demo", "First version")
	if _, err := Add(AddRequest{Source: src}); err != nil {
		t.Fatal(err)
	}
	writeTestSkill(t, filepath.Join(src, "SKILL.md"), "demo", "Second version")
	added, err := Add(AddRequest{Source: src, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if added.Entries[0].Description != "Second version" {
		t.Fatal("expected second version after force install")
	}
	listed, _ := List(ListRequest{})
	if len(listed) != 1 {
		t.Fatalf("expected one entry, got %d", len(listed))
	}
}

func TestRemoveNonExistentSkill(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	result, err := Remove(RemoveRequest{Name: "nope"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Removed {
		t.Fatal("expected Removed=false for non-existent skill")
	}
}

func TestListEmptyManifest(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	listed, err := List(ListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 0 {
		t.Fatalf("expected 0 skills, got %d", len(listed))
	}
}

func TestCheckCleanEnvironment(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	result, err := Check(DoctorRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Checks) > 1 || (len(result.Checks) == 1 && result.Checks[0].Code != "empty") {
		t.Fatalf("expected at most info about empty state, got %#v", result.Checks)
	}
}

func TestApplyManifestFromFile(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	cwd := t.TempDir()

	src1 := t.TempDir()
	writeTestSkill(t, filepath.Join(src1, "SKILL.md"), "alpha", "Alpha skill")
	src2 := t.TempDir()
	writeTestSkill(t, filepath.Join(src2, "SKILL.md"), "beta", "Beta skill")

	if _, err := Add(AddRequest{Source: src1}); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(AddRequest{Source: src2, Name: "beta"}); err != nil {
		t.Fatal(err)
	}
	export, err := ExportManifest(ExportManifestRequest{CWD: cwd})
	if err != nil {
		t.Fatal(err)
	}
	// Remove both skills so we can re-apply
	if _, err := Remove(RemoveRequest{Name: "alpha"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Remove(RemoveRequest{Name: "beta"}); err != nil {
		t.Fatal(err)
	}
	result, err := ApplyManifest(ApplyManifestRequest{Path: export.Path})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 2 {
		t.Fatalf("expected 2 entries from manifest, got %d", len(result.Entries))
	}
	listed, _ := List(ListRequest{})
	if len(listed) != 2 {
		t.Fatalf("expected 2 installed skills, got %d", len(listed))
	}
}

func TestCheckReportsMissingInstallDir(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	src := t.TempDir()
	writeTestSkill(t, filepath.Join(src, "SKILL.md"), "demo", "Demo skill")
	if _, err := Add(AddRequest{Source: src}); err != nil {
		t.Fatal(err)
	}
	installDir := filepath.Join(installedSkillsRoot(), "demo")
	if err := os.RemoveAll(installDir); err != nil {
		t.Fatal(err)
	}
	result, err := Check(DoctorRequest{})
	if err != nil {
		t.Fatal(err)
	}
	hasMissingDir := false
	for _, c := range result.Checks {
		if c.Code == "missing-skill" {
			hasMissingDir = true
		}
	}
	if !hasMissingDir {
		t.Fatalf("expected missing-install-dir check, got %#v", result.Checks)
	}
}
