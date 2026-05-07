package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallListRemoveCLI(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	src := t.TempDir()
	writeCLITestSkill(t, filepath.Join(src, "SKILL.md"), "demo")

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"install", src}, &stdout, &stderr); code != 0 {
		t.Fatalf("install code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "installed demo") {
		t.Fatalf("install stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "discover") || !strings.Contains(stderr.String(), "copy demo") {
		t.Fatalf("install stderr should show progress, got %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"list"}, &stdout, &stderr); code != 0 {
		t.Fatalf("list code = %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "demo  active") {
		t.Fatalf("list stdout = %q", stdout.String())
	}
	if strings.Contains(stdout.String(), src) {
		t.Fatalf("list should not print source path by default: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"remove", "demo"}, &stdout, &stderr); code != 0 {
		t.Fatalf("remove code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "removed demo") {
		t.Fatalf("remove stdout = %q", stdout.String())
	}
}

func TestInstallNameCLI(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	src := t.TempDir()
	writeCLITestSkill(t, filepath.Join(src, "SKILL.md"), "upstream")

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"install", src, "--name", "local"}, &stdout, &stderr); code != 0 {
		t.Fatalf("install code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "installed local") {
		t.Fatalf("install stdout = %q", stdout.String())
	}
}

func TestListAllCLI(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	cwd := t.TempDir()
	t.Chdir(cwd)

	writeCLITestSkill(t, filepath.Join(cwd, ".agents", "skills", "external-skill", "SKILL.md"), "external-skill")

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"list"}, &stdout, &stderr); code != 0 {
		t.Fatalf("list code = %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No skills installed.") {
		t.Fatalf("default list stdout = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"list", "--all"}, &stdout, &stderr); code != 0 {
		t.Fatalf("list --all code = %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "external\nexternal-skill  ") {
		t.Fatalf("list --all stdout = %q", stdout.String())
	}
}

func TestSearchOutputIsCompact(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	src := t.TempDir()
	writeCLITestSkill(t, filepath.Join(src, "SKILL.md"), "demo")

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"search", "demo", "--source", src}, &stdout, &stderr); code != 0 {
		t.Fatalf("search code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), src) {
		t.Fatalf("search stdout = %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "Install with") || strings.Contains(stdout.String(), "install: skit install") {
		t.Fatalf("search output should not include repeated install hints: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "use: skit install <source@skill>") {
		t.Fatalf("search output should include one compact install hint: %q", stdout.String())
	}
}

func TestSourcesAddListRemoveCLI(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	src := t.TempDir()
	writeCLITestSkill(t, filepath.Join(src, "SKILL.md"), "demo")

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"sources"}, &stdout, &stderr); code != 0 {
		t.Fatalf("sources code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "skills-sh") {
		t.Fatalf("default sources stdout = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"sources", "add", "local-test", "repo", src}, &stdout, &stderr); code != 0 {
		t.Fatalf("sources add code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "added local-test") {
		t.Fatalf("sources add stdout = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"search", "demo", "--source", "local-test"}, &stdout, &stderr); code != 0 {
		t.Fatalf("search named source code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "test skill") {
		t.Fatalf("search named source stdout = %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"sources", "remove", "local-test"}, &stdout, &stderr); code != 0 {
		t.Fatalf("sources remove code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "removed local-test") {
		t.Fatalf("sources remove stdout = %q", stdout.String())
	}
}

func TestSearchJSONSourceCLI(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	catalog := filepath.Join(t.TempDir(), "catalog.json")
	raw := `{
  "schema": "skit.catalog/v1",
  "skills": [
    {
      "name": "image-review",
      "target": "github:org/team-skills@image-review",
      "description": "Review generated images.",
      "keywords": ["vision"]
    }
  ]
}
`
	if err := os.WriteFile(catalog, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"sources", "add", "catalog", "json", catalog}, &stdout, &stderr); code != 0 {
		t.Fatalf("sources add json code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"search", "image", "--source", "catalog"}, &stdout, &stderr); code != 0 {
		t.Fatalf("search json source code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "github:org/team-skills@image-review") || !strings.Contains(stdout.String(), "Review generated images.") {
		t.Fatalf("search json stdout = %q", stdout.String())
	}
}

func TestCheckCLI(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)

	src := t.TempDir()
	writeCLITestSkill(t, filepath.Join(src, "SKILL.md"), "demo")

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"install", src}, &stdout, &stderr); code != 0 {
		t.Fatalf("install code = %d stderr=%q", code, stderr.String())
	}
	if err := os.Remove(filepath.Join(home, ".agents", "skills", "demo")); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"check"}, &stdout, &stderr); code != 1 {
		t.Fatalf("check code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "missing-active-link") {
		t.Fatalf("check stdout = %q", stdout.String())
	}
}

func TestInstallManifestCLI(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)

	src := t.TempDir()
	writeCLITestSkill(t, filepath.Join(src, "SKILL.md"), "demo")

	firstData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", firstData)
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"install", src, "--name", "shared-demo"}, &stdout, &stderr); code != 0 {
		t.Fatalf("install code = %d stderr=%q", code, stderr.String())
	}
	manifest := filepath.Join(firstData, "skit", "manifest.json")

	secondData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", secondData)
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"install", manifest}, &stdout, &stderr); code != 0 {
		t.Fatalf("install manifest code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "installed shared-demo") {
		t.Fatalf("install manifest stdout = %q", stdout.String())
	}
}

func TestExportAndInstallDefaultManifestCLI(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	cwd := t.TempDir()
	t.Chdir(cwd)

	src := t.TempDir()
	writeCLITestSkill(t, filepath.Join(src, "SKILL.md"), "demo")

	firstData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", firstData)
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"install", src}, &stdout, &stderr); code != 0 {
		t.Fatalf("install code = %d stderr=%q", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"export"}, &stdout, &stderr); code != 0 {
		t.Fatalf("export code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(cwd, "skit.json")); err != nil {
		t.Fatalf("skit.json missing: %v", err)
	}

	secondData := t.TempDir()
	t.Setenv("XDG_DATA_HOME", secondData)
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"install", "--dry-run"}, &stdout, &stderr); code != 0 {
		t.Fatalf("dry-run code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "would install demo") {
		t.Fatalf("dry-run stdout = %q", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"install"}, &stdout, &stderr); code != 0 {
		t.Fatalf("install default manifest code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "installed demo") {
		t.Fatalf("install default manifest stdout = %q", stdout.String())
	}
}

func TestInitCLI(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	cwd := t.TempDir()
	t.Chdir(cwd)

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"init", "demo"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "created demo-skill") {
		t.Fatalf("init stdout = %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(cwd, "demo-skill", "README.md")); err != nil {
		t.Fatalf("README missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cwd, "demo-skill", "skills", "demo", "SKILL.md")); err != nil {
		t.Fatalf("SKILL.md missing: %v", err)
	}
}

func writeCLITestSkill(t *testing.T, path, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: test skill\n---\n# " + name + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
