package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vlln/skit/internal/lockfile"
)

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

func TestUnknownFlagErrors(t *testing.T) {
	var out, err bytes.Buffer
	code := Run([]string{"add", "--skil", "demo", "./demo"}, &out, &err)
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
	if code := Run([]string{"add", "./demo"}, &out, &errOut); code != 0 {
		t.Fatalf("add code = %d, stderr = %q", code, errOut.String())
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
	if code := Run([]string{"add", "./demo"}, &out, &errOut); code != 0 {
		t.Fatalf("add code = %d, stderr = %q", code, errOut.String())
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

func TestDoctorJSONReturnsErrorStatus(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkillWithBody(t, filepath.Join(project, "demo"), "---\nname: demo\ndescription: Test skill.\nmetadata:\n  skit:\n    requires:\n      bins:\n        - definitely-missing-skit-bin\n---\n# Demo\n")

	var out, errOut bytes.Buffer
	if code := Run([]string{"add", "./demo"}, &out, &errOut); code != 0 {
		t.Fatalf("add code = %d, stderr = %q", code, errOut.String())
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

func TestAddPrintsDependencyAndInspectJSONIncludesEdge(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkill(t, filepath.Join(project, "dep"), "dep")
	writeCLITestSkillWithBody(t, filepath.Join(project, "parent"), "---\nname: parent\ndescription: Parent skill.\nmetadata:\n  skit:\n    dependencies:\n      - source: "+filepath.Join(project, "dep")+"\n---\n# Parent\n")

	var out, errOut bytes.Buffer
	if code := Run([]string{"add", "./parent"}, &out, &errOut); code != 0 {
		t.Fatalf("add code = %d, stderr = %q", code, errOut.String())
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

func TestAddIgnoreDepsCLI(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkill(t, filepath.Join(project, "dep"), "dep")
	writeCLITestSkillWithBody(t, filepath.Join(project, "parent"), "---\nname: parent\ndescription: Parent skill.\nmetadata:\n  skit:\n    dependencies:\n      - source: "+filepath.Join(project, "dep")+"\n---\n# Parent\n")

	var out, errOut bytes.Buffer
	if code := Run([]string{"add", "--ignore-deps", "./parent"}, &out, &errOut); code != 0 {
		t.Fatalf("add code = %d, stderr = %q", code, errOut.String())
	}
	if strings.Contains(out.String(), "added dependency") {
		t.Fatalf("stdout = %q", out.String())
	}
	if !strings.Contains(errOut.String(), "dependencies skipped for parent") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestAddOptionalDependencyWarningCLI(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkillWithBody(t, filepath.Join(project, "parent"), "---\nname: parent\ndescription: Parent skill.\nmetadata:\n  skit:\n    dependencies:\n      - source: "+filepath.Join(project, "missing")+"\n        optional: true\n---\n# Parent\n")

	var out, errOut bytes.Buffer
	if code := Run([]string{"add", "./parent"}, &out, &errOut); code != 0 {
		t.Fatalf("add code = %d, stderr = %q", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "optional dependency for parent failed") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestAddFullDepthCLI(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkill(t, filepath.Join(project, "root-skill"), "root-skill")
	writeCLITestSkill(t, filepath.Join(project, "packages", "tools", "skills", "deep-skill"), "deep-skill")

	var out, errOut bytes.Buffer
	if code := Run([]string{"add", "--full-depth", "--all", "."}, &out, &errOut); code != 0 {
		t.Fatalf("add code = %d, stdout = %q stderr = %q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "added root-skill ") || !strings.Contains(out.String(), "added deep-skill ") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestAddInternalSkipPrintsReasonCLI(t *testing.T) {
	project := t.TempDir()
	chdir(t, project)
	writeCLITestSkillWithBody(t, filepath.Join(project, "repo", "internal-skill"), "---\nname: internal-skill\ndescription: Internal skill.\nmetadata:\n  internal: true\n---\n# Internal\n")

	var out, errOut bytes.Buffer
	if code := Run([]string{"add", "./repo"}, &out, &errOut); code != 1 {
		t.Fatalf("add code = %d, stdout = %q stderr = %q", code, out.String(), errOut.String())
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
	if code := Run([]string{"add", "./demo"}, &out, &errOut); code != 0 {
		t.Fatalf("add code = %d, stderr = %q", code, errOut.String())
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
	if code := Run([]string{"add", "./demo"}, &out, &errOut); code != 0 {
		t.Fatalf("add code = %d, stderr = %q", code, errOut.String())
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
