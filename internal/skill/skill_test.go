package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDirMinimal(t *testing.T) {
	s, err := ParseDir("../../testdata/skills/minimal")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "minimal" || s.Description == "" {
		t.Fatalf("unexpected skill: %+v", s)
	}
}

func TestParseDirLowercaseWarning(t *testing.T) {
	s, err := ParseDir("../../testdata/skills/lowercase")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Warnings) != 1 || !strings.Contains(s.Warnings[0], "lowercase") {
		t.Fatalf("warnings = %#v", s.Warnings)
	}
}

func TestParseDirCanonicalWins(t *testing.T) {
	s, err := ParseDir("../../testdata/skills/both")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "both" {
		t.Fatalf("name = %q, want both", s.Name)
	}
	if len(s.Warnings) != 1 {
		t.Fatalf("warnings = %#v, want one warning", s.Warnings)
	}
}

func TestParseDirWarnsNameMismatch(t *testing.T) {
	s, err := ParseDir("../../testdata/skills/bad-name")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "wrong-name" {
		t.Fatalf("name = %q, want wrong-name", s.Name)
	}
	if len(s.Warnings) != 1 || !strings.Contains(s.Warnings[0], "does not match directory basename") {
		t.Fatalf("warnings = %#v", s.Warnings)
	}
}

func TestParseDirSkitMetadata(t *testing.T) {
	s, err := ParseDir("../../testdata/skills/with-skit")
	if err != nil {
		t.Fatal(err)
	}
	if s.Skit.Carrier != "metadata.skit" {
		t.Fatalf("carrier = %q", s.Skit.Carrier)
	}
	if s.Skit.Version != "1.2.0" {
		t.Fatalf("version = %q", s.Skit.Version)
	}
	if len(s.Skit.Dependencies) != 1 || !s.Skit.Dependencies[0].Optional {
		t.Fatalf("deps = %#v", s.Skit.Dependencies)
	}
	if got := s.Skit.Requires.AnyBins; len(got) != 2 || got[1] != "mutool" {
		t.Fatalf("anyBins = %#v", got)
	}
}

func TestParseDirManifest(t *testing.T) {
	s, err := ParseDir("../../testdata/skills/with-manifest")
	if err != nil {
		t.Fatal(err)
	}
	if s.Skit.Carrier != "skill.yaml" {
		t.Fatalf("carrier = %q", s.Skit.Carrier)
	}
	if s.Skit.Registry.Slug != "with-manifest" {
		t.Fatalf("registry slug = %q", s.Skit.Registry.Slug)
	}
}

func TestDiscoverBounded(t *testing.T) {
	skills, _, err := Discover("../../testdata/repos/discovery")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, s := range skills {
		names[s.Name] = true
	}
	for _, want := range []string{"root-skill", "nested-skill", "system-skill", "agent-skill"} {
		if !names[want] {
			t.Fatalf("missing discovered skill %q in %#v", want, names)
		}
	}
}

func TestDiscoverFallbackFindsDeepSkill(t *testing.T) {
	root := t.TempDir()
	writeTestSkill(t, filepath.Join(root, "packages", "tools", "skills", "deep-skill"), "deep-skill")

	skills, _, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 || skills[0].Name != "deep-skill" {
		t.Fatalf("skills = %#v", skills)
	}
}

func TestDiscoverFullDepthAddsDeepSkillWhenRootSkillExists(t *testing.T) {
	root := t.TempDir()
	writeTestSkill(t, root, "root-skill")
	writeTestSkill(t, filepath.Join(root, "packages", "tools", "skills", "deep-skill"), "deep-skill")

	skills, _, err := DiscoverWithOptions(root, ParseOptions{FullDepth: true})
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, s := range skills {
		names[s.Name] = true
	}
	if !names["root-skill"] || !names["deep-skill"] {
		t.Fatalf("names = %#v", names)
	}
}

func TestDiscoverChildSkillsUseChildBasename(t *testing.T) {
	root := t.TempDir()
	writeTestSkill(t, filepath.Join(root, "skills", "find-skills"), "find-skills")

	skills, warnings, err := DiscoverWithOptions(root, ParseOptions{ExpectedBasename: "skills"})
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 || skills[0].Name != "find-skills" {
		t.Fatalf("skills = %#v", skills)
	}
	if containsWarning(warnings, "does not match directory basename") || containsWarning(skills[0].Warnings, "does not match directory basename") {
		t.Fatalf("warnings = %#v skill warnings = %#v", warnings, skills[0].Warnings)
	}
}

func TestDiscoverPriorityAgentDirs(t *testing.T) {
	root := t.TempDir()
	writeTestSkill(t, filepath.Join(root, ".opencode", "skills", "opencode-skill"), "opencode-skill")
	writeTestSkill(t, filepath.Join(root, ".windsurf", "skills", "windsurf-skill"), "windsurf-skill")

	skills, _, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, s := range skills {
		names[s.Name] = true
	}
	if !names["opencode-skill"] || !names["windsurf-skill"] {
		t.Fatalf("names = %#v", names)
	}
}

func TestDiscoverSkipsInternalByDefault(t *testing.T) {
	root := t.TempDir()
	writeTestSkillWithBody(t, filepath.Join(root, "internal-skill"), "---\nname: internal-skill\ndescription: Test skill.\nmetadata:\n  internal: true\n---\n# Test\n")

	skills, warnings, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 0 {
		t.Fatalf("skills = %#v", skills)
	}
	if !containsWarning(warnings, "internal skill") {
		t.Fatalf("warnings = %#v", warnings)
	}
}

func TestDiscoverIncludesInternalWhenExplicit(t *testing.T) {
	root := t.TempDir()
	writeTestSkillWithBody(t, filepath.Join(root, "internal-skill"), "---\nname: internal-skill\ndescription: Test skill.\nmetadata:\n  internal: true\n---\n# Test\n")

	skills, _, err := DiscoverWithOptions(root, ParseOptions{IncludeInternal: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 || skills[0].Name != "internal-skill" || !skills[0].Internal {
		t.Fatalf("skills = %#v", skills)
	}
}

func TestDiscoverRejectsDuplicateRuntimeName(t *testing.T) {
	root := t.TempDir()
	writeTestSkill(t, filepath.Join(root, "one"), "same-name")
	writeTestSkill(t, filepath.Join(root, "two"), "same-name")

	_, _, err := Discover(root)
	if err == nil {
		t.Fatal("expected duplicate name error")
	}
	if !strings.Contains(err.Error(), "duplicate skill name") {
		t.Fatalf("err = %v", err)
	}
}

func writeTestSkill(t *testing.T, dir, name string) {
	t.Helper()
	body := "---\nname: " + name + "\ndescription: Test skill.\n---\n# Test\n"
	writeTestSkillWithBody(t, dir, body)
}

func writeTestSkillWithBody(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func containsWarning(warnings []string, text string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, text) {
			return true
		}
	}
	return false
}
