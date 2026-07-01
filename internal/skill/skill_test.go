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

func TestParseDirSkitRequires(t *testing.T) {
	s, err := ParseDir("../../testdata/skills/with-skit")
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Requires.AnyBins; len(got) != 2 || got[1] != "mutool" {
		t.Fatalf("anyBins = %#v", got)
	}
	if len(s.Requires.Bins) != 1 || s.Requires.Bins[0] != "qpdf" {
		t.Fatalf("bins = %#v", s.Requires.Bins)
	}
}

func TestParseDirLenientColonDescription(t *testing.T) {
	root := t.TempDir()
	writeTestSkillWithBody(t, root, "---\nname: my-skill\ndescription: Use when: the user asks about PDFs\n---\n# Test\n")
	s, err := ParseDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if s.Description != "Use when: the user asks about PDFs" {
		t.Fatalf("description = %q", s.Description)
	}
	hasRepairWarning := false
	for _, w := range s.Warnings {
		if strings.Contains(w, "repaired") {
			hasRepairWarning = true
			break
		}
	}
	if !hasRepairWarning {
		t.Fatalf("expected repair warning, got %#v", s.Warnings)
	}
}

func TestParseDirUnicodeName(t *testing.T) {
	root := t.TempDir()
	writeTestSkillWithBody(t, root, "---\nname: café\ndescription: Test skill.\n---\n# Test\n")
	s, err := ParseDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "café" {
		t.Fatalf("name = %q", s.Name)
	}
}

func TestParseDirChineseName(t *testing.T) {
	root := t.TempDir()
	writeTestSkillWithBody(t, root, "---\nname: 技能\ndescription: Test skill.\n---\n# Test\n")
	s, err := ParseDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "技能" {
		t.Fatalf("name = %q", s.Name)
	}
}

func TestParseDirNFKCNormalization(t *testing.T) {
	root := t.TempDir()
	decomposed := "cafe\u0301"
	composed := "café"
	writeTestSkillWithBody(t, root, "---\nname: "+decomposed+"\ndescription: Test.\n---\n# Test\n")
	s, err := ParseDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != composed {
		t.Fatalf("name = %q (expected NFKC normalized %q)", s.Name, composed)
	}
}

func TestParseDirInvalidNameWarning(t *testing.T) {
	root := t.TempDir()
	writeTestSkillWithBody(t, root, "---\nname: MySkill\ndescription: Test.\n---\n# Test\n")
	s, err := ParseDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "MySkill" {
		t.Fatalf("name = %q", s.Name)
	}
	hasWarning := false
	for _, w := range s.Warnings {
		if strings.Contains(w, "invalid") {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Fatalf("expected invalid name warning, got %#v", s.Warnings)
	}
}

func TestParseDirDescriptionTooLongWarning(t *testing.T) {
	root := t.TempDir()
	longDesc := strings.Repeat("x", 1100)
	writeTestSkillWithBody(t, root, "---\nname: my-skill\ndescription: "+longDesc+"\n---\n# Test\n")
	s, err := ParseDir(root)
	if err != nil {
		t.Fatal(err)
	}
	hasWarning := false
	for _, w := range s.Warnings {
		if strings.Contains(w, "description exceeds") {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Fatalf("expected description too long warning, got %#v", s.Warnings)
	}
}

func TestParseDirNameDefaultsToDir(t *testing.T) {
	root := t.TempDir()
	writeTestSkillWithBody(t, root, "---\ndescription: Test.\n---\n# Test\n")
	s, err := ParseDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name == "" {
		t.Fatal("name should not be empty")
	}
}

func TestParseDirLicense(t *testing.T) {
	root := t.TempDir()
	writeTestSkillWithBody(t, root, "---\nname: my-skill\ndescription: Test.\nlicense: MIT\n---\n# Test\n")
	s, err := ParseDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if s.License != "MIT" {
		t.Fatalf("license = %q", s.License)
	}
}

func TestParseDirMetadata(t *testing.T) {
	root := t.TempDir()
	writeTestSkillWithBody(t, root, "---\nname: my-skill\ndescription: Test.\nmetadata:\n  author: example-org\n  version: \"1.0\"\n---\n# Test\n")
	s, err := ParseDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if s.Metadata["author"] != "example-org" {
		t.Fatalf("metadata.author = %q", s.Metadata["author"])
	}
	if s.Metadata["version"] != "1.0" {
		t.Fatalf("metadata.version = %q", s.Metadata["version"])
	}
}

func TestParseDirNoDescription(t *testing.T) {
	root := t.TempDir()
	writeTestSkillWithBody(t, root, "---\nname: my-skill\n---\n# Test\n")
	s, err := ParseDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if s.Description != "" {
		t.Fatalf("description should be empty, got %q", s.Description)
	}
}

func TestParseDirFrontmatterPreserved(t *testing.T) {
	root := t.TempDir()
	writeTestSkillWithBody(t, root, "---\nname: my-skill\ndescription: Test.\nallowed-tools:\n  - Bash(git:*)\nwhen-to-use: For PDF work\n---\n# Test\n")
	s, err := ParseDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Frontmatter["allowed-tools"]; !ok {
		t.Fatal("allowed-tools not in frontmatter")
	}
	if _, ok := s.Frontmatter["when-to-use"]; !ok {
		t.Fatal("when-to-use not in frontmatter")
	}
}

func TestValidateSkill(t *testing.T) {
	s := Skill{
		Name:        "my-skill",
		Description: "A valid skill.",
		Root:        "/tmp/my-skill",
	}
	errs := ValidateSkill(s)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %#v", errs)
	}
}

func TestValidateSkillNameMismatch(t *testing.T) {
	s := Skill{
		Name:        "wrong-name",
		Description: "Description.",
		Root:        "/tmp/my-skill",
	}
	errs := ValidateSkill(s)
	if len(errs) == 0 {
		t.Fatal("expected name mismatch error")
	}
}

func TestValidateSkillEmptyDescription(t *testing.T) {
	s := Skill{
		Name:        "my-skill",
		Description: "",
		Root:        "/tmp/my-skill",
	}
	errs := ValidateSkill(s)
	hasWarning := false
	for _, e := range errs {
		if strings.Contains(e, "description is recommended") {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Fatalf("expected description recommendation, got %#v", errs)
	}
}

func TestValidName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"my-skill", true},
		{"pdf-processing", true},
		{"data-analysis", true},
		{"a", true},
		{"café", true},
		{"技能", true},
		{"навык", true},
		{"мой-навык", true},
		{"MySkill", false},
		{"-leading", false},
		{"trailing-", false},
		{"double--hyphen", false},
		{"", false},
		{strings.Repeat("a", 65), false},
	}
	for _, tt := range tests {
		got := ValidName(tt.name)
		if got != tt.valid {
			t.Errorf("ValidName(%q) = %v, want %v", tt.name, got, tt.valid)
		}
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
	if len(skills) != 1 || skills[0].Name != "internal-skill" || !isInternal(skills[0]) {
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