package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vlln/skit/internal/lockfile"
	"github.com/vlln/skit/internal/skill"
	"github.com/vlln/skit/internal/source"
	"github.com/vlln/skit/internal/store"
)

func TestMain(m *testing.M) {
	root, err := os.MkdirTemp("", "skit-app-test-*")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	_ = os.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	_ = os.Setenv("HOME", filepath.Join(root, "home"))
	code := m.Run()
	_ = os.RemoveAll(root)
	os.Exit(code)
}

func TestLocalClosedLoop(t *testing.T) {
	project := t.TempDir()
	source := filepath.Join(project, "demo")
	writeSkill(t, source, "demo")

	added, err := Add(AddRequest{CWD: project, Scope: Project, Source: source})
	if err != nil {
		t.Fatal(err)
	}
	if len(added.Entries) != 1 {
		t.Fatalf("entries = %#v", added.Entries)
	}
	if _, err := os.Stat(filepath.Join(project, ".agents", "skills", "skit.lock")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(added.StorePaths[0]); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(filepath.Join(project, ".agents", "skills", "demo")); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(project, ".agent", "skills", "demo")); !os.IsNotExist(err) {
		t.Fatalf("legacy project active symlink should not exist: %v", err)
	}

	listed, err := List(ListRequest{CWD: project, Scope: Project})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Name != "demo" {
		t.Fatalf("listed = %#v", listed)
	}

	removed, err := Remove(RemoveRequest{CWD: project, Scope: Project, Name: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if !removed.Removed {
		t.Fatal("remove returned false")
	}
	listed, err = List(ListRequest{CWD: project, Scope: Project})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 0 {
		t.Fatalf("listed after remove = %#v", listed)
	}
	if _, err := os.Lstat(filepath.Join(project, ".agents", "skills", "demo")); !os.IsNotExist(err) {
		t.Fatalf("active symlink still exists after remove: %v", err)
	}
}

func TestAddActivatesCodexProjectAgent(t *testing.T) {
	project := t.TempDir()
	source := filepath.Join(project, "demo")
	writeSkill(t, source, "demo")

	added, err := Add(AddRequest{CWD: project, Scope: Project, Source: source, Agents: []string{"codex"}})
	if err != nil {
		t.Fatal(err)
	}
	if !containsText(added.ActivePaths, filepath.Join(project, ".agents", "skills", "demo")) {
		t.Fatalf("active paths = %#v", added.ActivePaths)
	}
	if info, err := os.Lstat(filepath.Join(project, ".agents", "skills", "demo")); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("codex active symlink missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(project, ".agents", "skills", "skit.lock")); err != nil {
		t.Fatal(err)
	}
}

func TestAddGlobalUsesAgentsPluralRoot(t *testing.T) {
	project := t.TempDir()
	source := filepath.Join(project, "global-demo")
	writeSkill(t, source, "global-demo")

	added, err := Add(AddRequest{CWD: project, Scope: Global, Source: source})
	if err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".agents", "skills", "global-demo")
	if !containsText(added.ActivePaths, want) {
		t.Fatalf("active paths = %#v, want %s", added.ActivePaths, want)
	}
	if info, err := os.Lstat(want); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("global active symlink missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "skit.lock")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(home, ".agent", "skills", "global-demo")); !os.IsNotExist(err) {
		t.Fatalf("legacy global active symlink should not exist: %v", err)
	}
}

func TestInstallRestoresCodexGlobalAgent(t *testing.T) {
	project := t.TempDir()
	codexHome := filepath.Join(project, "codex-home")
	t.Setenv("CODEX_HOME", codexHome)
	source := filepath.Join(project, "demo")
	writeSkill(t, source, "demo")

	if _, err := Add(AddRequest{CWD: project, Scope: Global, Source: source, NoActive: true}); err != nil {
		t.Fatal(err)
	}
	result, err := Install(InstallRequest{CWD: project, Scope: Global, Agents: []string{"codex"}})
	if err != nil {
		t.Fatal(err)
	}
	if !containsText(result.ActivePaths, filepath.Join(codexHome, "skills", "demo")) {
		t.Fatalf("active paths = %#v", result.ActivePaths)
	}
	if info, err := os.Lstat(filepath.Join(codexHome, "skills", "demo")); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("codex global active symlink missing: %v", err)
	}
}

func TestInstallRestoresMissingStoreFromLocalSource(t *testing.T) {
	project := t.TempDir()
	source := filepath.Join(project, "demo")
	writeSkill(t, source, "demo")

	added, err := Add(AddRequest{CWD: project, Scope: Project, Source: source})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(added.StorePaths[0]); err != nil {
		t.Fatal(err)
	}
	result, err := Install(InstallRequest{CWD: project, Scope: Project})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Restored) != 1 || result.Restored[0].Name != "demo" {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(added.StorePaths[0]); err != nil {
		t.Fatal(err)
	}
}

func TestRemovePruneDeletesUnreferencedStore(t *testing.T) {
	project := t.TempDir()
	source := filepath.Join(project, "prune-demo")
	writeSkill(t, source, "prune-demo")

	added, err := Add(AddRequest{CWD: project, Scope: Project, Source: source})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Remove(RemoveRequest{CWD: project, Scope: Project, Name: "prune-demo", Prune: true})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Removed || len(result.Pruned) != 1 {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(added.StorePaths[0]); !os.IsNotExist(err) {
		t.Fatalf("store path still exists or stat failed: %v", err)
	}
}

func TestRemovePruneKeepsStoreReferencedByAnotherProject(t *testing.T) {
	projectA := t.TempDir()
	projectB := t.TempDir()
	source := filepath.Join(projectA, "shared-prune-demo")
	writeSkill(t, source, "shared-prune-demo")

	addedA, err := Add(AddRequest{CWD: projectA, Scope: Project, Source: source})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Add(AddRequest{CWD: projectB, Scope: Project, Source: source}); err != nil {
		t.Fatal(err)
	}
	result, err := Remove(RemoveRequest{CWD: projectA, Scope: Project, Name: "shared-prune-demo", Prune: true})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Removed || len(result.Pruned) != 0 || len(result.Skipped) != 1 {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(addedA.StorePaths[0]); err != nil {
		t.Fatal(err)
	}
}

func TestGCPrunesOnlyUnreferencedStore(t *testing.T) {
	project := t.TempDir()
	keep := filepath.Join(project, "keep")
	drop := filepath.Join(project, "drop")
	writeSkill(t, keep, "keep")
	writeSkill(t, drop, "drop")

	kept, err := Add(AddRequest{CWD: project, Scope: Project, Source: keep})
	if err != nil {
		t.Fatal(err)
	}
	dropped, err := Add(AddRequest{CWD: project, Scope: Project, Source: drop})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Remove(RemoveRequest{CWD: project, Scope: Project, Name: "drop"}); err != nil {
		t.Fatal(err)
	}
	result, err := GC(GCRequest{CWD: project})
	if err != nil {
		t.Fatal(err)
	}
	if !containsText(result.Pruned, dropped.StorePaths[0]) {
		t.Fatalf("gc result = %#v, dropped path = %s", result, dropped.StorePaths[0])
	}
	if _, err := os.Stat(dropped.StorePaths[0]); !os.IsNotExist(err) {
		t.Fatalf("dropped store still exists or stat failed: %v", err)
	}
	if _, err := os.Stat(kept.StorePaths[0]); err != nil {
		t.Fatal(err)
	}
}

func TestInstallRestoresMissingStoreFromLockedGitCommit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	project := t.TempDir()
	repo := filepath.Join(project, "remote")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "init", "-b", "main")
	git(t, repo, "config", "user.email", "test@example.com")
	git(t, repo, "config", "user.name", "Test")
	writeSkillWithBody(t, repo, "---\nname: demo\ndescription: Original skill.\n---\n# Demo\noriginal\n")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "initial")
	initialCommit := gitOutput(t, repo, "rev-parse", "HEAD")
	parsed, err := skill.ParseDir(repo)
	if err != nil {
		t.Fatal(err)
	}
	paths := store.PathsFor(Project, project)
	installed, err := store.InstallSnapshot(paths, parsed)
	if err != nil {
		t.Fatal(err)
	}
	lockedHash := installed.Hashes.Tree
	lockedStorePath := installed.Path
	lock := lockfile.New()
	lock.Skills["demo"] = lockfile.Entry{
		Name:        "demo",
		Description: "Original skill.",
		Source: lockfile.Source{
			Type:        "git",
			Locator:     "git-local",
			URL:         repo,
			Ref:         "main",
			ResolvedRef: initialCommit,
			Skill:       "demo",
		},
		Hashes: lockfile.Hashes{Tree: installed.Hashes.Tree, SkillMD: installed.Hashes.SkillMD},
	}
	if err := lockfile.Write(paths.Lock, lock); err != nil {
		t.Fatal(err)
	}

	writeSkillWithBody(t, repo, "---\nname: demo\ndescription: Updated skill.\n---\n# Demo\nupdated\n")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "update")
	if err := os.RemoveAll(lockedStorePath); err != nil {
		t.Fatal(err)
	}

	result, err := Install(InstallRequest{CWD: project, Scope: Project})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Restored) != 1 || result.Restored[0].Hashes.Tree != lockedHash {
		t.Fatalf("result = %#v, locked hash = %s", result, lockedHash)
	}
	if _, err := os.Stat(lockedStorePath); err != nil {
		t.Fatal(err)
	}
}

func TestInstallRejectsCorruptExistingStore(t *testing.T) {
	project := t.TempDir()
	source := filepath.Join(project, "corrupt-demo")
	writeSkill(t, source, "corrupt-demo")

	added, err := Add(AddRequest{CWD: project, Scope: Project, Source: source})
	if err != nil {
		t.Fatal(err)
	}
	storeSkill := filepath.Join(added.StorePaths[0], "SKILL.md")
	if err := os.WriteFile(storeSkill, []byte("---\nname: corrupt-demo\ndescription: Tampered skill.\n---\n# corrupt-demo\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = Install(InstallRequest{CWD: project, Scope: Project})
	if err == nil {
		t.Fatal("expected hash mismatch")
	}
	if !strings.Contains(err.Error(), "hash mismatch restoring") {
		t.Fatalf("err = %v", err)
	}
}

func TestInstallParsedPreservesSourceWarnings(t *testing.T) {
	project := t.TempDir()
	dir := filepath.Join(project, "demo")
	writeSkill(t, dir, "demo")
	parsed, err := skill.ParseDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	session := addSession{
		ctx:      context.Background(),
		paths:    store.PathsFor(Project, project),
		lock:     lockfile.New(),
		result:   &AddResult{},
		visiting: map[string]bool{},
	}
	src := source.Source{
		Type:     source.Local,
		Locator:  dir,
		Warnings: []string{"inline skill selector ignored because --skill was provided"},
	}
	entry, _, err := session.installParsed(src, parsed, dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if !containsText(session.result.Warnings, src.Warnings[0]) {
		t.Fatalf("result warnings = %#v", session.result.Warnings)
	}
	if !containsText(entry.Warnings, src.Warnings[0]) {
		t.Fatalf("entry warnings = %#v", entry.Warnings)
	}
	if !containsText(session.lock.Skills["demo"].Warnings, src.Warnings[0]) {
		t.Fatalf("lock warnings = %#v", session.lock.Skills["demo"].Warnings)
	}
}

func TestUpdateRefreshesLocalSkill(t *testing.T) {
	project := t.TempDir()
	source := filepath.Join(project, "demo")
	writeSkill(t, source, "demo")

	added, err := Add(AddRequest{CWD: project, Scope: Project, Source: source})
	if err != nil {
		t.Fatal(err)
	}
	oldHash := added.Entries[0].Hashes.Tree
	writeSkillWithBody(t, source, "---\nname: demo\ndescription: Test skill demo updated.\n---\n# demo\nupdated\n")

	updated, err := Update(UpdateRequest{CWD: project, Scope: Project, Name: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Entries) != 1 || updated.Entries[0].Name != "demo" {
		t.Fatalf("updated = %#v", updated.Entries)
	}
	if updated.Entries[0].Hashes.Tree == oldHash {
		t.Fatalf("tree hash did not change: %s", oldHash)
	}
	listed, err := List(ListRequest{CWD: project, Scope: Project})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Hashes.Tree != updated.Entries[0].Hashes.Tree {
		t.Fatalf("listed = %#v updated = %#v", listed, updated.Entries)
	}
}

func TestUpdateMissingSkill(t *testing.T) {
	project := t.TempDir()
	_, err := Update(UpdateRequest{CWD: project, Scope: Project, Name: "missing"})
	if err == nil {
		t.Fatal("expected missing skill error")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdateGitHubSourceWithLocalGitRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	project := t.TempDir()
	repo := filepath.Join(project, "remote")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "init", "-b", "main")
	git(t, repo, "config", "user.email", "test@example.com")
	git(t, repo, "config", "user.name", "Test")
	writeSkill(t, repo, "demo")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "initial")

	lock := lockfile.New()
	lock.Skills["demo"] = lockfile.Entry{
		Name:        "demo",
		Description: "old",
		Source: lockfile.Source{
			Type:    "github",
			Locator: "owner/remote",
			URL:     repo,
			Ref:     "main",
			Skill:   "demo",
		},
		Hashes: lockfile.Hashes{Tree: "sha256-old", SkillMD: "sha256-old"},
	}
	if err := lockfile.Write(filepath.Join(project, ".agents", "skills", "skit.lock"), lock); err != nil {
		t.Fatal(err)
	}

	writeSkillWithBody(t, repo, "---\nname: demo\ndescription: Updated from git.\n---\n# Demo\nupdated\n")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "update")

	updated, err := Update(UpdateRequest{CWD: project, Scope: Project, Name: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Entries) != 1 {
		t.Fatalf("updated = %#v", updated.Entries)
	}
	entry := updated.Entries[0]
	if entry.Source.ResolvedRef == "" || entry.Hashes.Tree == "sha256-old" {
		t.Fatalf("entry = %#v", entry)
	}
	if entry.Description != "Updated from git." {
		t.Fatalf("description = %q", entry.Description)
	}
}

func TestUpdateGenericGitSourceWithLocalGitRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	project := t.TempDir()
	repo := filepath.Join(project, "remote")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "init", "-b", "main")
	git(t, repo, "config", "user.email", "test@example.com")
	git(t, repo, "config", "user.name", "Test")
	writeSkill(t, repo, "demo")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "initial")

	lock := lockfile.New()
	lock.Skills["demo"] = lockfile.Entry{
		Name:        "demo",
		Description: "old",
		Source: lockfile.Source{
			Type:    "git",
			Locator: "git-local",
			URL:     repo,
			Ref:     "main",
			Skill:   "demo",
		},
		Hashes: lockfile.Hashes{Tree: "sha256-old", SkillMD: "sha256-old"},
	}
	if err := lockfile.Write(filepath.Join(project, ".agents", "skills", "skit.lock"), lock); err != nil {
		t.Fatal(err)
	}

	writeSkillWithBody(t, repo, "---\nname: demo\ndescription: Updated from generic git.\n---\n# Demo\nupdated\n")
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-m", "update")

	updated, err := Update(UpdateRequest{CWD: project, Scope: Project, Name: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Entries) != 1 {
		t.Fatalf("updated = %#v", updated.Entries)
	}
	entry := updated.Entries[0]
	if entry.Source.Type != "git" || entry.Source.ResolvedRef == "" || entry.Hashes.Tree == "sha256-old" {
		t.Fatalf("entry = %#v", entry)
	}
	if entry.Description != "Updated from generic git." {
		t.Fatalf("description = %q", entry.Description)
	}
}

func TestInstallRequiresSkillForMultipleSkills(t *testing.T) {
	project := t.TempDir()
	repo := filepath.Join(project, "repo")
	writeSkill(t, filepath.Join(repo, "one"), "one")
	writeSkill(t, filepath.Join(repo, "skills", "two"), "two")

	_, err := Add(AddRequest{CWD: project, Scope: Project, Source: repo})
	if err == nil {
		t.Fatal("expected multiple skill error")
	}
	added, err := Add(AddRequest{CWD: project, Scope: Project, Source: repo, Skill: "two"})
	if err != nil {
		t.Fatal(err)
	}
	if len(added.Entries) != 1 || added.Entries[0].Source.Subpath != "skills/two" {
		t.Fatalf("added = %#v", added.Entries)
	}
}

func TestInstallSkipsInternalUnlessSkillExplicit(t *testing.T) {
	project := t.TempDir()
	repo := filepath.Join(project, "repo")
	writeSkillWithBody(t, filepath.Join(repo, "internal-skill"), "---\nname: internal-skill\ndescription: Internal skill.\nmetadata:\n  internal: true\n---\n# Internal\n")

	_, err := Add(AddRequest{CWD: project, Scope: Project, Source: repo})
	if err == nil {
		t.Fatal("expected no skills found")
	}
	if !strings.Contains(err.Error(), "no skills found") {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(err.Error(), "internal skill") {
		t.Fatalf("err = %v", err)
	}

	added, err := Add(AddRequest{CWD: project, Scope: Project, Source: repo, Skill: "internal-skill"})
	if err != nil {
		t.Fatal(err)
	}
	if len(added.Entries) != 1 || added.Entries[0].Name != "internal-skill" {
		t.Fatalf("added = %#v", added.Entries)
	}
}

func TestInstallInstallsRequiredDependencies(t *testing.T) {
	project := t.TempDir()
	dep := filepath.Join(project, "dep")
	parent := filepath.Join(project, "parent")
	writeSkill(t, dep, "dep")
	writeSkillWithBody(t, parent, "---\nname: parent\ndescription: Parent skill.\nmetadata:\n  skit:\n    dependencies:\n      - source: "+dep+"\n---\n# Parent\n")

	added, err := Add(AddRequest{CWD: project, Scope: Project, Source: parent})
	if err != nil {
		t.Fatal(err)
	}
	if len(added.Entries) != 1 || added.Entries[0].Name != "parent" {
		t.Fatalf("added = %#v", added.Entries)
	}
	if len(added.DependencyEntries) != 1 || added.DependencyEntries[0].Name != "dep" {
		t.Fatalf("dependency entries = %#v", added.DependencyEntries)
	}
	if len(added.Entries[0].Dependencies) != 1 || added.Entries[0].Dependencies[0].Name != "dep" {
		t.Fatalf("dependencies = %#v", added.Entries[0].Dependencies)
	}
	listed, err := List(ListRequest{CWD: project, Scope: Project})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 || listed[0].Name != "dep" || listed[1].Name != "parent" {
		t.Fatalf("listed = %#v", listed)
	}
	inspected, err := Inspect(InspectRequest{CWD: project, Scope: Project, Target: "parent"})
	if err != nil {
		t.Fatal(err)
	}
	if len(inspected.Dependencies) != 1 || inspected.Dependencies[0].Name != "dep" {
		t.Fatalf("inspect dependencies = %#v", inspected.Dependencies)
	}
}

func TestInstallOptionalDependencyFailureWarns(t *testing.T) {
	project := t.TempDir()
	parent := filepath.Join(project, "parent")
	missing := filepath.Join(project, "missing")
	writeSkillWithBody(t, parent, "---\nname: parent\ndescription: Parent skill.\nmetadata:\n  skit:\n    dependencies:\n      - source: "+missing+"\n        optional: true\n---\n# Parent\n")

	added, err := Add(AddRequest{CWD: project, Scope: Project, Source: parent})
	if err != nil {
		t.Fatal(err)
	}
	if len(added.Entries[0].Dependencies) != 0 {
		t.Fatalf("dependencies = %#v", added.Entries[0].Dependencies)
	}
	if !containsText(added.Warnings, "optional dependency for parent failed") {
		t.Fatalf("warnings = %#v", added.Warnings)
	}
	if !containsText(added.Entries[0].Warnings, "optional dependency for parent failed") {
		t.Fatalf("entry warnings = %#v", added.Entries[0].Warnings)
	}
}

func TestInstallIgnoreDepsSkipsEdges(t *testing.T) {
	project := t.TempDir()
	dep := filepath.Join(project, "dep")
	parent := filepath.Join(project, "parent")
	writeSkill(t, dep, "dep")
	writeSkillWithBody(t, parent, "---\nname: parent\ndescription: Parent skill.\nmetadata:\n  skit:\n    dependencies:\n      - source: "+dep+"\n---\n# Parent\n")

	added, err := Add(AddRequest{CWD: project, Scope: Project, Source: parent, IgnoreDeps: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(added.Entries[0].Dependencies) != 0 {
		t.Fatalf("dependencies = %#v", added.Entries[0].Dependencies)
	}
	if !containsText(added.Entries[0].Warnings, "dependencies skipped for parent") {
		t.Fatalf("entry warnings = %#v", added.Entries[0].Warnings)
	}
	listed, err := List(ListRequest{CWD: project, Scope: Project})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Name != "parent" {
		t.Fatalf("listed = %#v", listed)
	}
}

func TestInstallRejectsCircularDependencies(t *testing.T) {
	project := t.TempDir()
	a := filepath.Join(project, "a")
	b := filepath.Join(project, "b")
	writeSkillWithBody(t, a, "---\nname: a\ndescription: Skill a.\nmetadata:\n  skit:\n    dependencies:\n      - source: "+b+"\n---\n# A\n")
	writeSkillWithBody(t, b, "---\nname: b\ndescription: Skill b.\nmetadata:\n  skit:\n    dependencies:\n      - source: "+a+"\n---\n# B\n")

	_, err := Add(AddRequest{CWD: project, Scope: Project, Source: a})
	if err == nil {
		t.Fatal("expected circular dependency error")
	}
	if !strings.Contains(err.Error(), "circular dependency") {
		t.Fatalf("err = %v", err)
	}
}

func TestInspectLockedSkill(t *testing.T) {
	project := t.TempDir()
	source := filepath.Join(project, "demo")
	writeSkillWithBody(t, source, "---\nname: demo\ndescription: Test skill demo.\nmetadata:\n  skit:\n    requires:\n      bins:\n        - definitely-missing-skit-bin\n---\n# demo\n")

	if _, err := Add(AddRequest{CWD: project, Scope: Project, Source: source}); err != nil {
		t.Fatal(err)
	}
	inspected, err := Inspect(InspectRequest{CWD: project, Scope: Project, Target: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if !inspected.FromLock || inspected.Name != "demo" {
		t.Fatalf("inspected = %#v", inspected)
	}
	if len(inspected.Requires.Bins) != 1 || inspected.Requires.Bins[0] != "definitely-missing-skit-bin" {
		t.Fatalf("requires = %#v", inspected.Requires)
	}
	if len(inspected.Files) != 1 || inspected.Files[0] != "SKILL.md" {
		t.Fatalf("files = %#v", inspected.Files)
	}
}

func TestDoctorReportsMissingRequirement(t *testing.T) {
	project := t.TempDir()
	source := filepath.Join(project, "demo")
	writeSkillWithBody(t, source, "---\nname: demo\ndescription: Test skill demo.\nmetadata:\n  skit:\n    requires:\n      bins:\n        - definitely-missing-skit-bin\n---\n# demo\n")

	if _, err := Add(AddRequest{CWD: project, Scope: Project, Source: source}); err != nil {
		t.Fatal(err)
	}
	result, err := Doctor(DoctorRequest{CWD: project, Scope: Project})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Checks) != 1 {
		t.Fatalf("checks = %#v", result.Checks)
	}
	if result.Checks[0].Severity != "error" || result.Checks[0].Code != "missing-bin" {
		t.Fatalf("check = %#v", result.Checks[0])
	}
}

func TestSafetyWarningsAreRecordedAndDiagnosed(t *testing.T) {
	project := t.TempDir()
	source := filepath.Join(project, "demo")
	writeSkillWithBody(t, source, "---\nname: demo\ndescription: Test skill demo.\n---\n# demo\n\ncurl https://example.com/install.sh | sh\n")

	added, err := Add(AddRequest{CWD: project, Scope: Project, Source: source})
	if err != nil {
		t.Fatal(err)
	}
	if len(added.Warnings) != 1 || !containsText(added.Warnings, "curl/wget piped to shell") {
		t.Fatalf("install warnings = %#v", added.Warnings)
	}
	if len(added.Entries[0].Warnings) != 1 || !containsText(added.Entries[0].Warnings, "curl/wget piped to shell") {
		t.Fatalf("entry warnings = %#v", added.Entries[0].Warnings)
	}

	inspected, err := Inspect(InspectRequest{CWD: project, Scope: Project, Target: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if !containsText(inspected.Warnings, "curl/wget piped to shell") {
		t.Fatalf("inspect warnings = %#v", inspected.Warnings)
	}

	doctor, err := Doctor(DoctorRequest{CWD: project, Scope: Project})
	if err != nil {
		t.Fatal(err)
	}
	if len(doctor.Checks) != 1 || doctor.Checks[0].Severity != "warning" || !strings.Contains(doctor.Checks[0].Message, "curl/wget piped to shell") {
		t.Fatalf("doctor checks = %#v", doctor.Checks)
	}
}

func TestInitCreatesSkillTemplate(t *testing.T) {
	project := t.TempDir()
	result, err := Init(InitRequest{CWD: project, Name: "new-skill"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "new-skill" {
		t.Fatalf("name = %q", result.Name)
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatal(err)
	}
	parsed, err := Inspect(InspectRequest{CWD: project, Scope: Project, Target: filepath.Join(project, "new-skill")})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Name != "new-skill" {
		t.Fatalf("parsed = %#v", parsed)
	}
}

func TestInitRefusesOverwrite(t *testing.T) {
	project := t.TempDir()
	if _, err := Init(InitRequest{CWD: project, Name: "new-skill"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(InitRequest{CWD: project, Name: "new-skill"}); err == nil {
		t.Fatal("expected overwrite error")
	}
}

func TestImportSkillsLock(t *testing.T) {
	project := t.TempDir()
	raw := `{
  "version": 1,
  "skills": {
    "demo": {
      "source": "owner/repo",
      "ref": "main",
      "sourceType": "github",
      "computedHash": "abc123"
    }
  }
}
`
	if err := os.WriteFile(filepath.Join(project, "skills-lock.json"), []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	result, err := ImportLock(ImportLockRequest{CWD: project, Scope: Project, Kind: "skills"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 1 || result.Entries[0].Name != "demo" || !result.Entries[0].Incomplete {
		t.Fatalf("result = %#v", result)
	}
	listed, err := List(ListRequest{CWD: project, Scope: Project})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Source.Type != "github" {
		t.Fatalf("listed = %#v", listed)
	}
}

func TestImportClawHubLock(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, ".clawhub"), 0755); err != nil {
		t.Fatal(err)
	}
	raw := `{
  "version": 1,
  "skills": {
    "demo": {
      "version": "1.2.3",
      "installedAt": 123
    }
  }
}
`
	if err := os.WriteFile(filepath.Join(project, ".clawhub", "lock.json"), []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	result, err := ImportLock(ImportLockRequest{CWD: project, Scope: Project, Kind: "clawhub"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 1 || result.Entries[0].Source.Type != "registry" || !result.Entries[0].Incomplete {
		t.Fatalf("result = %#v", result)
	}
}

func writeSkill(t *testing.T, dir, name string) {
	t.Helper()
	writeSkillWithBody(t, dir, "---\nname: "+name+"\ndescription: Test skill "+name+".\n---\n# "+name+"\n")
}

func writeSkillWithBody(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func containsText(items []string, text string) bool {
	for _, item := range items {
		if strings.Contains(item, text) {
			return true
		}
	}
	return false
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}
