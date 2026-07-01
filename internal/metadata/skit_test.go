package metadata

import "testing"

func TestRequiresTopLevel(t *testing.T) {
	fm, err := ParseYAML(`
name: demo
description: Demo.
requires:
  bins:
    - git
    - jq
  any-bins:
    - pdftotext
    - mutool
  env:
    - API_KEY
  config:
    - ~/.config/demo
  skills:
    - github:owner/repo@dep-skill
  platforms:
    os:
      - linux
      - darwin
    arch:
      - amd64
`)
	if err != nil {
		t.Fatal(err)
	}
	got := FromCarriers(fm)
	if len(got.Requires.Bins) != 2 || got.Requires.Bins[0] != "git" || got.Requires.Bins[1] != "jq" {
		t.Fatalf("bins = %#v", got.Requires.Bins)
	}
	if len(got.Requires.AnyBins) != 2 || got.Requires.AnyBins[1] != "mutool" {
		t.Fatalf("anyBins = %#v", got.Requires.AnyBins)
	}
	if len(got.Requires.Env) != 1 || got.Requires.Env[0] != "API_KEY" {
		t.Fatalf("env = %#v", got.Requires.Env)
	}
	if len(got.Requires.Config) != 1 || got.Requires.Config[0] != "~/.config/demo" {
		t.Fatalf("config = %#v", got.Requires.Config)
	}
	if len(got.Requires.Skills) != 1 || got.Requires.Skills[0] != "github:owner/repo@dep-skill" {
		t.Fatalf("skills = %#v", got.Requires.Skills)
	}
	if len(got.Requires.Platforms.OS) != 2 || got.Requires.Platforms.OS[0] != "linux" {
		t.Fatalf("os = %#v", got.Requires.Platforms.OS)
	}
	if len(got.Requires.Platforms.Arch) != 1 || got.Requires.Platforms.Arch[0] != "amd64" {
		t.Fatalf("arch = %#v", got.Requires.Platforms.Arch)
	}
}

func TestRequiresMetadataSkitFallback(t *testing.T) {
	fm, err := ParseYAML(`
name: demo
description: Demo.
metadata:
  skit:
    requires:
      bins:
        - qpdf
      env:
        - PDF_API_KEY
`)
	if err != nil {
		t.Fatal(err)
	}
	got := FromCarriers(fm)
	if len(got.Requires.Bins) != 1 || got.Requires.Bins[0] != "qpdf" {
		t.Fatalf("bins = %#v", got.Requires.Bins)
	}
	if len(got.Requires.Env) != 1 || got.Requires.Env[0] != "PDF_API_KEY" {
		t.Fatalf("env = %#v", got.Requires.Env)
	}
}

func TestRequiresTopLevelWinsOverMetadataSkit(t *testing.T) {
	fm, err := ParseYAML(`
name: demo
description: Demo.
requires:
  bins:
    - git
metadata:
  skit:
    requires:
      bins:
        - qpdf
`)
	if err != nil {
		t.Fatal(err)
	}
	got := FromCarriers(fm)
	if len(got.Requires.Bins) != 1 || got.Requires.Bins[0] != "git" {
		t.Fatalf("bins = %#v (top-level should win)", got.Requires.Bins)
	}
}

func TestRequiresEmpty(t *testing.T) {
	fm, err := ParseYAML(`
name: demo
description: Demo.
`)
	if err != nil {
		t.Fatal(err)
	}
	got := FromCarriers(fm)
	if len(got.Requires.Bins) != 0 {
		t.Fatalf("expected empty requires, got %#v", got.Requires)
	}
}
