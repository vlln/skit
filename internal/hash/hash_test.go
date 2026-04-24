package hash

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTreeStable(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "SKILL.md"), "---\nname: demo\ndescription: Demo.\n---\n")
	write(t, filepath.Join(dir, "scripts", "run.sh"), "#!/bin/sh\n")
	if err := os.Chmod(filepath.Join(dir, "scripts", "run.sh"), 0755); err != nil {
		t.Fatal(err)
	}
	first, err := Tree(dir, filepath.Join(dir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Tree(dir, filepath.Join(dir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if first.Tree == "" || first.SkillMD == "" || first != second {
		t.Fatalf("hashes not stable: %#v %#v", first, second)
	}
}

func TestTreeRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "SKILL.md"), "---\nname: demo\ndescription: Demo.\n---\n")
	if err := os.Symlink("SKILL.md", filepath.Join(dir, "link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err := Tree(dir, filepath.Join(dir, "SKILL.md"))
	if err == nil {
		t.Fatal("expected symlink rejection")
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
