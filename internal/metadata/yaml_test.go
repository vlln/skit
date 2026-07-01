package metadata

import "testing"

func TestParseBlockScalar(t *testing.T) {
	got, err := ParseYAML("name: demo\ndescription: |\n  first\n  second\n")
	if err != nil {
		t.Fatal(err)
	}
	if got["description"] != "first\nsecond\n" {
		t.Fatalf("description = %#v", got["description"])
	}
}

func TestParseFoldedBlockScalar(t *testing.T) {
	got, err := ParseYAML("description: >\n  first\n  second\n")
	if err != nil {
		t.Fatal(err)
	}
	if got["description"] != "first second\n" {
		t.Fatalf("description = %#v", got["description"])
	}
}

func TestParseInlineList(t *testing.T) {
	got, err := ParseYAML("requires:\n  bins: [git, rg]\n")
	if err != nil {
		t.Fatal(err)
	}
	requires, ok := AsMap(got["requires"])
	if !ok {
		t.Fatalf("requires = %#v", got["requires"])
	}
	items, ok := requires["bins"].([]any)
	if !ok || len(items) != 2 || items[0] != "git" || items[1] != "rg" {
		t.Fatalf("bins = %#v", requires["bins"])
	}
}

func TestParseYAMLLenientColon(t *testing.T) {
	// Unquoted values with colons should be repaired
	src := "name: demo\ndescription: Use this skill when: the user asks about PDFs\n"
	got, err := ParseYAMLLenient(src)
	if err != nil {
		t.Fatal(err)
	}
	if got["description"] != "Use this skill when: the user asks about PDFs" {
		t.Fatalf("description = %#v", got["description"])
	}
}

func TestParseYAMLLenientPreservesValid(t *testing.T) {
	// Valid YAML should pass through unchanged
	src := "name: demo\ndescription: A simple description\n"
	got, err := ParseYAMLLenient(src)
	if err != nil {
		t.Fatal(err)
	}
	if got["description"] != "A simple description" {
		t.Fatalf("description = %#v", got["description"])
	}
}

func TestParseYAMLLenientQuotedValue(t *testing.T) {
	// Already-quoted values should not be double-quoted
	src := "name: demo\ndescription: \"Use when: user asks\"\n"
	got, err := ParseYAMLLenient(src)
	if err != nil {
		t.Fatal(err)
	}
	if got["description"] != "Use when: user asks" {
		t.Fatalf("description = %#v", got["description"])
	}
}

func TestParseYAMLLenientBlockScalar(t *testing.T) {
	// Block scalars should not be quoted
	src := "name: demo\ndescription: |\n  Use when: user asks\n"
	got, err := ParseYAMLLenient(src)
	if err != nil {
		t.Fatal(err)
	}
	if got["description"] != "Use when: user asks\n" {
		t.Fatalf("description = %#v", got["description"])
	}
}
