package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vlln/skit/internal/lockfile"
)

func TestMain(m *testing.M) {
	root, err := os.MkdirTemp("", "skit-cli-test-*")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	_ = os.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	code := m.Run()
	_ = os.RemoveAll(root)
	os.Exit(code)
}

func TestRunHelp(t *testing.T) {
	var out, err bytes.Buffer
	code := Run([]string{"--help"}, &out, &err)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("help output missing usage: %q", out.String())
	}
	if err.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", err.String())
	}
}

func TestRunVersion(t *testing.T) {
	var out, err bytes.Buffer
	code := Run([]string{"version"}, &out, &err)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.HasPrefix(out.String(), "skit ") {
		t.Fatalf("version output = %q", out.String())
	}
}

func TestRunVersionCheckReportsUpdate(t *testing.T) {
	oldVersion := version
	version = "0.1.0"
	t.Cleanup(func() { version = oldVersion })
	t.Setenv("SKIT_UPDATE_CHECK_CACHE", filepath.Join(t.TempDir(), "update-check.json"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v0.2.0","html_url":"https://example.com/skit/releases/v0.2.0"}`))
	}))
	defer server.Close()
	t.Setenv("SKIT_UPDATE_CHECK_URL", server.URL)

	var out, err bytes.Buffer
	code := Run([]string{"version", "--check"}, &out, &err)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, err.String())
	}
	if !strings.Contains(out.String(), "skit 0.1.0") {
		t.Fatalf("stdout = %q", out.String())
	}
	if !strings.Contains(err.String(), "update available: skit v0.2.0 is available") {
		t.Fatalf("stderr = %q", err.String())
	}
}

func TestUnknownFlagErrors(t *testing.T) {
	var out, err bytes.Buffer
	code := Run([]string{"install", "--skil", "demo", "./demo"}, &out, &err)
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(err.String(), "unknown flag --skil") {
		t.Fatalf("stderr = %q", err.String())
	}
}

func TestListJSON(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkill(t, filepath.Join(project, "demo"), "demo")

	var out, errOut bytes.Buffer
	if code := Run([]string{"install", "./demo"}, &out, &errOut); code != 0 {
		t.Fatalf("install code = %d, stderr = %q", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := Run([]string{"list", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("list code = %d, stderr = %q", code, errOut.String())
	}
	var entries []lockfile.Entry
	if err := json.Unmarshal(out.Bytes(), &entries); err != nil {
		t.Fatalf("json = %q: %v", out.String(), err)
	}
	if len(entries) != 1 || entries[0].Name != "demo" {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestInspectJSON(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkill(t, filepath.Join(project, "demo"), "demo")

	var out, errOut bytes.Buffer
	if code := Run([]string{"install", "./demo"}, &out, &errOut); code != 0 {
		t.Fatalf("install code = %d, stderr = %q", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := Run([]string{"inspect", "demo", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("inspect code = %d, stderr = %q", code, errOut.String())
	}
	var result struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("json = %q: %v", out.String(), err)
	}
	if result.Name != "demo" {
		t.Fatalf("result = %#v", result)
	}
}

func TestSearchJSON(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "skills": [
		    {"id":"skill-creator","name":"skill-creator","source":"openclaw/openclaw","installs":42}
		  ]
		}`))
	}))
	defer server.Close()
	t.Setenv("SKIT_SEARCH_API_URL", server.URL)

	var out, errOut bytes.Buffer
	if code := Run([]string{"search", "skill", "create", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("search code = %d, stderr = %q", code, errOut.String())
	}
	var results []struct {
		Name   string `json:"name"`
		Source string `json:"source"`
	}
	if err := json.Unmarshal(out.Bytes(), &results); err != nil {
		t.Fatalf("json = %q: %v", out.String(), err)
	}
	if len(results) != 1 || results[0].Name != "skill-creator" || results[0].Source != "openclaw/openclaw" {
		t.Fatalf("results = %#v", results)
	}
}

func TestSearchTextIsCompact(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "skills": [
		    {"id":"skill-creator","name":"skill-creator","source":"openclaw/openclaw","installs":42},
		    {"id":"no-source","name":"fallback-skill","source":"","installs":0}
		  ]
		}`))
	}))
	defer server.Close()
	t.Setenv("SKIT_SEARCH_API_URL", server.URL)

	var out, errOut bytes.Buffer
	if code := Run([]string{"search", "skill", "create"}, &out, &errOut); code != 0 {
		t.Fatalf("search code = %d, stderr = %q", code, errOut.String())
	}
	got := out.String()
	if strings.Count(got, "Install with: skit install <source@skill>") != 1 {
		t.Fatalf("install hint count wrong: %q", got)
	}
	if strings.Contains(got, "  skit install ") {
		t.Fatalf("per-result install command still present: %q", got)
	}
	if !strings.Contains(got, "openclaw/openclaw@skill-creator\t42 installs\thttps://skills.sh/skill-creator") {
		t.Fatalf("missing compact result line: %q", got)
	}
	if !strings.Contains(got, "no-source@fallback-skill\thttps://skills.sh/no-source") {
		t.Fatalf("missing slug fallback result line: %q", got)
	}
}

func TestDoctorJSONReturnsErrorStatus(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkillWithBody(t, filepath.Join(project, "demo"), "---\nname: demo\ndescription: Test skill.\nmetadata:\n  skit:\n    requires:\n      bins:\n        - definitely-missing-skit-bin\n---\n# Demo\n")

	var out, errOut bytes.Buffer
	if code := Run([]string{"install", "./demo"}, &out, &errOut); code != 0 {
		t.Fatalf("install code = %d, stderr = %q", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := Run([]string{"doctor", "--json"}, &out, &errOut); code != 1 {
		t.Fatalf("doctor code = %d, want 1; stdout = %q stderr = %q", code, out.String(), errOut.String())
	}
	var result struct {
		Errors []struct {
			Code string `json:"code"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("json = %q: %v", out.String(), err)
	}
	if len(result.Errors) != 1 || result.Errors[0].Code != "missing-bin" {
		t.Fatalf("result = %#v", result)
	}
}

func TestInstallPrintsDependencyAndInspectJSONIncludesEdge(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkill(t, filepath.Join(project, "dep"), "dep")
	writeCLITestSkillWithBody(t, filepath.Join(project, "parent"), "---\nname: parent\ndescription: Parent skill.\nmetadata:\n  skit:\n    dependencies:\n      - source: "+filepath.Join(project, "dep")+"\n---\n# Parent\n")

	var out, errOut bytes.Buffer
	if code := Run([]string{"install", "./parent"}, &out, &errOut); code != 0 {
		t.Fatalf("install code = %d, stderr = %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "added dependency dep ") {
		t.Fatalf("stdout = %q", out.String())
	}
	out.Reset()
	errOut.Reset()
	if code := Run([]string{"inspect", "parent", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("inspect code = %d, stderr = %q", code, errOut.String())
	}
	var inspected struct {
		Dependencies []struct {
			Name string `json:"name"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(out.Bytes(), &inspected); err != nil {
		t.Fatalf("json = %q: %v", out.String(), err)
	}
	if len(inspected.Dependencies) != 1 || inspected.Dependencies[0].Name != "dep" {
		t.Fatalf("inspected = %#v", inspected)
	}
}

func TestInstallIgnoreDepsCLI(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkill(t, filepath.Join(project, "dep"), "dep")
	writeCLITestSkillWithBody(t, filepath.Join(project, "parent"), "---\nname: parent\ndescription: Parent skill.\nmetadata:\n  skit:\n    dependencies:\n      - source: "+filepath.Join(project, "dep")+"\n---\n# Parent\n")

	var out, errOut bytes.Buffer
	if code := Run([]string{"install", "--ignore-deps", "./parent"}, &out, &errOut); code != 0 {
		t.Fatalf("install code = %d, stderr = %q", code, errOut.String())
	}
	if strings.Contains(out.String(), "added dependency") {
		t.Fatalf("stdout = %q", out.String())
	}
	if !strings.Contains(errOut.String(), "dependencies skipped for parent") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestInstallTextOutputIsCompact(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	skillDir := filepath.Join(project, "demo")
	writeCLITestSkill(t, skillDir, "demo")
	scripts := filepath.Join(skillDir, "scripts")
	if err := os.MkdirAll(scripts, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"one.sh", "two.sh"} {
		path := filepath.Join(scripts, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	var out, errOut bytes.Buffer
	if code := Run([]string{"install", "./demo"}, &out, &errOut); code != 0 {
		t.Fatalf("install code = %d, stdout = %q stderr = %q", code, out.String(), errOut.String())
	}
	if strings.Contains(out.String(), "\nstore ") {
		t.Fatalf("stdout should not print store path by default: %q", out.String())
	}
	if strings.Count(errOut.String(), "executable file") != 1 || !strings.Contains(errOut.String(), "2 executable files in Skill directory") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestInstallAutomaticallyChecksForUpdate(t *testing.T) {
	oldVersion := version
	version = "0.1.0"
	t.Cleanup(func() { version = oldVersion })
	project := t.TempDir()
	chdir(t, project)
	t.Setenv("SKIT_UPDATE_CHECK_CACHE", filepath.Join(t.TempDir(), "update-check.json"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v0.2.0","html_url":"https://example.com/skit/releases/v0.2.0"}`))
	}))
	defer server.Close()
	t.Setenv("SKIT_UPDATE_CHECK_URL", server.URL)
	writeCLITestSkill(t, filepath.Join(project, "demo"), "demo")

	var out, errOut bytes.Buffer
	if code := Run([]string{"install", "./demo"}, &out, &errOut); code != 0 {
		t.Fatalf("install code = %d, stderr = %q", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "update available: skit v0.2.0 is available") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestRemoveMultipleAndUninstallAliasCLI(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkill(t, filepath.Join(project, "one"), "one")
	writeCLITestSkill(t, filepath.Join(project, "two"), "two")

	var out, errOut bytes.Buffer
	if code := Run([]string{"install", "./one", "./two"}, &out, &errOut); code != 0 {
		t.Fatalf("install code = %d, stderr = %q", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := Run([]string{"uninstall", "one", "two"}, &out, &errOut); code != 0 {
		t.Fatalf("remove code = %d, stdout = %q stderr = %q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "removed one") || !strings.Contains(out.String(), "removed two") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestRemovePruneCLI(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkill(t, filepath.Join(project, "prune-cli"), "prune-cli")

	var out, errOut bytes.Buffer
	if code := Run([]string{"install", "./prune-cli"}, &out, &errOut); code != 0 {
		t.Fatalf("install code = %d, stderr = %q", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := Run([]string{"remove", "--prune", "prune-cli"}, &out, &errOut); code != 0 {
		t.Fatalf("remove code = %d, stdout = %q stderr = %q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "removed prune-cli") || !strings.Contains(out.String(), "pruned ") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestGCCLI(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkill(t, filepath.Join(project, "gc-cli"), "gc-cli")

	var out, errOut bytes.Buffer
	if code := Run([]string{"install", "./gc-cli"}, &out, &errOut); code != 0 {
		t.Fatalf("install code = %d, stderr = %q", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := Run([]string{"remove", "gc-cli"}, &out, &errOut); code != 0 {
		t.Fatalf("remove code = %d, stdout = %q stderr = %q", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := Run([]string{"gc"}, &out, &errOut); code != 0 {
		t.Fatalf("gc code = %d, stdout = %q stderr = %q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "pruned ") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestInstallOptionalDependencyWarningCLI(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkillWithBody(t, filepath.Join(project, "parent"), "---\nname: parent\ndescription: Parent skill.\nmetadata:\n  skit:\n    dependencies:\n      - source: "+filepath.Join(project, "missing")+"\n        optional: true\n---\n# Parent\n")

	var out, errOut bytes.Buffer
	if code := Run([]string{"install", "./parent"}, &out, &errOut); code != 0 {
		t.Fatalf("install code = %d, stderr = %q", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "optional dependency for parent failed") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestInstallFullDepthCLI(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkill(t, filepath.Join(project, "root-skill"), "root-skill")
	writeCLITestSkill(t, filepath.Join(project, "packages", "tools", "skills", "deep-skill"), "deep-skill")

	var out, errOut bytes.Buffer
	if code := Run([]string{"install", "--full-depth", "--all", "."}, &out, &errOut); code != 0 {
		t.Fatalf("install code = %d, stdout = %q stderr = %q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "added root-skill ") || !strings.Contains(out.String(), "added deep-skill ") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestInstallSkillFlagAcceptsMultipleValuesOnce(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkill(t, filepath.Join(project, "repo", "one"), "one")
	writeCLITestSkill(t, filepath.Join(project, "repo", "two"), "two")

	var out, errOut bytes.Buffer
	if code := Run([]string{"install", "./repo", "--skill", "one", "two"}, &out, &errOut); code != 0 {
		t.Fatalf("install code = %d, stdout = %q stderr = %q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "added one ") || !strings.Contains(out.String(), "added two ") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestInstallRejectsRepeatedSkillFlag(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkill(t, filepath.Join(project, "repo", "one"), "one")
	writeCLITestSkill(t, filepath.Join(project, "repo", "two"), "two")

	var out, errOut bytes.Buffer
	if code := Run([]string{"install", "./repo", "--skill", "one", "--skill", "two"}, &out, &errOut); code != 2 {
		t.Fatalf("install code = %d, stdout = %q stderr = %q", code, out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "--skill may only be provided once") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestInstallInternalSkipPrintsReasonCLI(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkillWithBody(t, filepath.Join(project, "repo", "internal-skill"), "---\nname: internal-skill\ndescription: Internal skill.\nmetadata:\n  internal: true\n---\n# Internal\n")

	var out, errOut bytes.Buffer
	if code := Run([]string{"install", "./repo"}, &out, &errOut); code != 1 {
		t.Fatalf("install code = %d, stdout = %q stderr = %q", code, out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "internal skill") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestUpdateCommand(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	path := filepath.Join(project, "demo")
	writeCLITestSkill(t, path, "demo")

	var out, errOut bytes.Buffer
	if code := Run([]string{"install", "./demo"}, &out, &errOut); code != 0 {
		t.Fatalf("install code = %d, stderr = %q", code, errOut.String())
	}
	writeCLITestSkillWithBody(t, path, "---\nname: demo\ndescription: Updated skill.\n---\n# Updated\n")
	out.Reset()
	errOut.Reset()
	if code := Run([]string{"update", "demo"}, &out, &errOut); code != 0 {
		t.Fatalf("update code = %d, stderr = %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "updated demo ") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestUpdateJSON(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	path := filepath.Join(project, "demo")
	writeCLITestSkill(t, path, "demo")

	var out, errOut bytes.Buffer
	if code := Run([]string{"install", "./demo"}, &out, &errOut); code != 0 {
		t.Fatalf("install code = %d, stderr = %q", code, errOut.String())
	}
	writeCLITestSkillWithBody(t, path, "---\nname: demo\ndescription: Updated skill.\n---\n# Updated\n")
	out.Reset()
	errOut.Reset()
	if code := Run([]string{"update", "demo", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("update code = %d, stderr = %q", code, errOut.String())
	}
	var result struct {
		Entries []struct {
			Name string `json:"name"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("json = %q: %v", out.String(), err)
	}
	if len(result.Entries) != 1 || result.Entries[0].Name != "demo" {
		t.Fatalf("result = %#v", result)
	}
}

func TestImportLockJSON(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	raw := `{
  "version": 1,
  "skills": {
    "demo": {
      "source": "owner/repo",
      "sourceType": "github",
      "computedHash": "abc123"
    }
  }
}
`
	if err := os.WriteFile(filepath.Join(project, "skills-lock.json"), []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if code := Run([]string{"import-lock", "skills", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("import code = %d, stderr = %q", code, errOut.String())
	}
	var result struct {
		Entries []struct {
			Name       string `json:"name"`
			Incomplete bool   `json:"incomplete"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("json = %q: %v", out.String(), err)
	}
	if len(result.Entries) != 1 || result.Entries[0].Name != "demo" || !result.Entries[0].Incomplete {
		t.Fatalf("result = %#v", result)
	}
}

func TestInitCommand(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)

	var out, errOut bytes.Buffer
	if code := Run([]string{"init", "new-skill"}, &out, &errOut); code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, errOut.String())
	}
	if _, err := os.Stat(filepath.Join(project, "new-skill", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

func writeCLITestSkill(t *testing.T, dir, name string) {
	t.Helper()
	writeCLITestSkillWithBody(t, dir, "---\nname: "+name+"\ndescription: Test skill.\n---\n# Test\n")
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Fatal(err)
		}
	})
}

func writeCLITestSkillWithBody(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}
