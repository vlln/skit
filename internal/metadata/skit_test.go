package metadata

import "testing"

func TestCompatibilityClawdbotRequires(t *testing.T) {
	fm, err := ParseYAML(`
name: demo
description: Demo.
metadata:
  clawdbot:
    requires:
      bins:
        - qpdf
      anyBins:
        - rg
      env:
        - PDF_API_KEY
      config:
        - ~/.config/demo
      skills:
        - github:owner/repo@dep-skill
    primaryEnv: PRIMARY_KEY
    os:
      - macos
    homepage: https://example.com/demo
`)
	if err != nil {
		t.Fatal(err)
	}
	got, err := FromCarriers(fm, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Requires.Bins) != 1 || got.Requires.Bins[0] != "qpdf" {
		t.Fatalf("bins = %#v", got.Requires.Bins)
	}
	if len(got.Requires.AnyBins) != 1 || got.Requires.AnyBins[0] != "rg" {
		t.Fatalf("anyBins = %#v", got.Requires.AnyBins)
	}
	if len(got.Requires.Env) != 2 || got.Requires.Env[0] != "PDF_API_KEY" || got.Requires.Env[1] != "PRIMARY_KEY" {
		t.Fatalf("env = %#v", got.Requires.Env)
	}
	if len(got.Requires.Config) != 1 || got.Requires.Config[0] != "~/.config/demo" {
		t.Fatalf("config = %#v", got.Requires.Config)
	}
	if len(got.Requires.Skills) != 1 || got.Requires.Skills[0] != "github:owner/repo@dep-skill" {
		t.Fatalf("skills = %#v", got.Requires.Skills)
	}
	if len(got.Platforms.OS) != 1 || got.Platforms.OS[0] != "darwin" {
		t.Fatalf("os = %#v", got.Platforms.OS)
	}
	if got.Registry.Homepage != "https://example.com/demo" {
		t.Fatalf("homepage = %q", got.Registry.Homepage)
	}
}

func TestExplicitSkitWinsOverCompatibility(t *testing.T) {
	fm, err := ParseYAML(`
name: demo
description: Demo.
metadata:
  skit:
    requires:
      bins:
        - git
      skills:
        - github:owner/repo@explicit
  clawdbot:
    requires:
      bins:
        - qpdf
      env:
        - API_KEY
      skills:
        - github:owner/repo@compat
`)
	if err != nil {
		t.Fatal(err)
	}
	got, err := FromCarriers(fm, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Requires.Bins) != 1 || got.Requires.Bins[0] != "git" {
		t.Fatalf("bins = %#v", got.Requires.Bins)
	}
	if len(got.Requires.Env) != 1 || got.Requires.Env[0] != "API_KEY" {
		t.Fatalf("env = %#v", got.Requires.Env)
	}
	if len(got.Requires.Skills) != 1 || got.Requires.Skills[0] != "github:owner/repo@explicit" {
		t.Fatalf("skills = %#v", got.Requires.Skills)
	}
}
