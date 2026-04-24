package lockfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteReadStable(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".skit", "lock.json")
	lock := New()
	var err error
	lock, err = Add(lock, Entry{
		Name:        "demo",
		Description: "Demo skill",
		Source:      Source{Type: "local", Locator: "/tmp/demo"},
		Hashes:      Hashes{Tree: "sha256-tree", SkillMD: "sha256-md"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(path, lock); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(raw), "\n") {
		t.Fatal("lock does not end with newline")
	}
	read, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if read.Skills["demo"].Hashes.Tree != "sha256-tree" {
		t.Fatalf("read lock = %#v", read)
	}
}

func TestAddRejectsSameNameDifferentHash(t *testing.T) {
	lock := New()
	lock, err := Add(lock, Entry{Name: "demo", Source: Source{Type: "local", Locator: "/a"}, Hashes: Hashes{Tree: "one"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Add(lock, Entry{Name: "demo", Source: Source{Type: "local", Locator: "/a"}, Hashes: Hashes{Tree: "two"}})
	if err == nil {
		t.Fatal("expected conflict")
	}
}
