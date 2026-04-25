package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vlln/skit/internal/skill"
)

type InitRequest struct {
	CWD  string
	Name string
}

type InitResult struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func Init(req InitRequest) (InitResult, error) {
	cwd := cleanCWD(req.CWD)
	name := req.Name
	dir := cwd
	if name == "" {
		name = filepath.Base(cwd)
	} else {
		dir = filepath.Join(cwd, name)
	}
	if !skill.ValidName(name) {
		return InitResult{}, fmt.Errorf("invalid skill name %q", name)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return InitResult{}, err
	}
	path := filepath.Join(dir, "SKILL.md")
	if exists(path) {
		return InitResult{}, fmt.Errorf("SKILL.md already exists in %s", dir)
	}
	body := skillTemplate(name)
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		return InitResult{}, err
	}
	return InitResult{Name: name, Path: path}, nil
}
func skillTemplate(name string) string {
	return "---\n" +
		"name: " + name + "\n" +
		"description: \"TODO: describe " + name + ".\"\n" +
		"metadata:\n" +
		"  skit:\n" +
		"    version: 0.1.0\n" +
		"---\n" +
		"# " + name + "\n" +
		"\n" +
		"Describe when to use this skill and what workflow it should follow.\n"
}
