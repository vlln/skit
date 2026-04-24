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
