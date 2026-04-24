package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/vlln/skit/internal/app"
	"github.com/vlln/skit/internal/lockfile"
)

const version = "0.1.0-dev"

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "-h", "--help", "help":
		printHelp(stdout)
		return 0
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "skit %s\n", version)
		return 0
	case "add":
		return runAdd(args[1:], stdout, stderr)
	case "install":
		return runInstall(args[1:], stdout, stderr)
	case "list", "ls":
		return runList(args[1:], stdout, stderr)
	case "remove", "rm":
		return runRemove(args[1:], stdout, stderr)
	case "update":
		return runUpdate(args[1:], stdout, stderr)
	case "inspect":
		return runInspect(args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "import-lock":
		return runImportLock(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "skit: unknown command %q\n\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "skit - Skill management CLI")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  skit <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  add        Add a local Skill to the skit store")
	fmt.Fprintln(w, "  install    Restore Skills from lock")
	fmt.Fprintln(w, "  list       List locked Skills")
	fmt.Fprintln(w, "  remove     Remove a Skill from lock")
	fmt.Fprintln(w, "  update     Refresh locked Skills from their sources")
	fmt.Fprintln(w, "  inspect    Inspect a locked Skill or source")
	fmt.Fprintln(w, "  doctor     Check lock, store, and declared requirements")
	fmt.Fprintln(w, "  init       Create a SKILL.md template")
	fmt.Fprintln(w, "  import-lock Import a compatible lock file")
	fmt.Fprintln(w, "  help       Show help")
	fmt.Fprintln(w, "  version    Show version")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Common flags:")
	fmt.Fprintln(w, "  --full-depth  Search recursively for Skills when adding a source")
}

func runAdd(args []string, stdout, stderr io.Writer) int {
	opts, rest, err := parseCommon(args)
	if err != nil {
		fmt.Fprintln(stderr, "skit add:", err)
		return 2
	}
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "skit add: expected exactly one source")
		return 2
	}
	result, err := app.Add(app.AddRequest{
		CWD:        cwd(),
		Scope:      opts.scope,
		Source:     rest[0],
		Skill:      opts.skill,
		All:        opts.all,
		IgnoreDeps: opts.ignoreDeps,
		FullDepth:  opts.fullDepth,
	})
	if err != nil {
		fmt.Fprintln(stderr, "skit add:", err)
		return 1
	}
	for i, entry := range result.DependencyEntries {
		fmt.Fprintf(stdout, "added dependency %s %s\n", entry.Name, entry.Hashes.Tree)
		if i < len(result.DependencyStorePaths) {
			fmt.Fprintf(stdout, "store %s\n", result.DependencyStorePaths[i])
		}
	}
	for i, entry := range result.Entries {
		fmt.Fprintf(stdout, "added %s %s\n", entry.Name, entry.Hashes.Tree)
		if i < len(result.StorePaths) {
			fmt.Fprintf(stdout, "store %s\n", result.StorePaths[i])
		}
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}
	fmt.Fprintln(stdout, "note: v1 stores and locks Skills only; Agent target sync is deferred")
	return 0
}

func runInstall(args []string, stdout, stderr io.Writer) int {
	opts, rest, err := parseCommon(args)
	if err != nil {
		fmt.Fprintln(stderr, "skit install:", err)
		return 2
	}
	if len(rest) != 0 {
		fmt.Fprintln(stderr, "skit install: unexpected arguments")
		return 2
	}
	result, err := app.Install(app.InstallRequest{CWD: cwd(), Scope: opts.scope})
	if err != nil {
		fmt.Fprintln(stderr, "skit install:", err)
		return 1
	}
	for _, entry := range result.Restored {
		fmt.Fprintf(stdout, "restored %s %s\n", entry.Name, entry.Hashes.Tree)
	}
	for _, entry := range result.Skipped {
		fmt.Fprintf(stderr, "skipped incomplete entry %s\n", entry.Name)
	}
	return 0
}

func runList(args []string, stdout, stderr io.Writer) int {
	opts, rest, err := parseCommon(args)
	if err != nil {
		fmt.Fprintln(stderr, "skit list:", err)
		return 2
	}
	if len(rest) != 0 {
		fmt.Fprintln(stderr, "skit list: unexpected arguments")
		return 2
	}
	entries, err := app.List(app.ListRequest{CWD: cwd(), Scope: opts.scope})
	if err != nil {
		fmt.Fprintln(stderr, "skit list:", err)
		return 1
	}
	if opts.json {
		return writeJSON(stdout, stderr, entries)
	}
	for _, entry := range entries {
		ref := entry.Source.ResolvedRef
		if ref == "" {
			ref = entry.Source.Ref
		}
		if ref == "" {
			ref = entry.Hashes.Tree
		}
		fmt.Fprintf(stdout, "%s\t%s\t%s\n", entry.Name, entry.Source.Locator, ref)
	}
	return 0
}

func runRemove(args []string, stdout, stderr io.Writer) int {
	opts, rest, err := parseCommon(args)
	if err != nil {
		fmt.Fprintln(stderr, "skit remove:", err)
		return 2
	}
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "skit remove: expected exactly one skill name")
		return 2
	}
	removed, err := app.Remove(app.RemoveRequest{CWD: cwd(), Scope: opts.scope, Name: rest[0]})
	if err != nil {
		fmt.Fprintln(stderr, "skit remove:", err)
		return 1
	}
	if !removed {
		fmt.Fprintf(stderr, "skit remove: %s is not installed\n", rest[0])
		return 1
	}
	fmt.Fprintf(stdout, "removed %s\n", rest[0])
	return 0
}

func runUpdate(args []string, stdout, stderr io.Writer) int {
	opts, rest, err := parseCommon(args)
	if err != nil {
		fmt.Fprintln(stderr, "skit update:", err)
		return 2
	}
	if len(rest) > 1 {
		fmt.Fprintln(stderr, "skit update: expected zero or one skill name")
		return 2
	}
	name := ""
	if len(rest) == 1 {
		name = rest[0]
	}
	result, err := app.Update(app.UpdateRequest{CWD: cwd(), Scope: opts.scope, Name: name, IgnoreDeps: opts.ignoreDeps})
	if err != nil {
		fmt.Fprintln(stderr, "skit update:", err)
		return 1
	}
	if opts.json {
		return writeJSON(stdout, stderr, result)
	}
	for i, entry := range result.DependencyEntries {
		fmt.Fprintf(stdout, "updated dependency %s %s\n", entry.Name, entry.Hashes.Tree)
		if i < len(result.DependencyStorePaths) {
			fmt.Fprintf(stdout, "store %s\n", result.DependencyStorePaths[i])
		}
	}
	for i, entry := range result.Entries {
		fmt.Fprintf(stdout, "updated %s %s\n", entry.Name, entry.Hashes.Tree)
		if i < len(result.StorePaths) {
			fmt.Fprintf(stdout, "store %s\n", result.StorePaths[i])
		}
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}
	return 0
}

func runInspect(args []string, stdout, stderr io.Writer) int {
	opts, rest, err := parseCommon(args)
	if err != nil {
		fmt.Fprintln(stderr, "skit inspect:", err)
		return 2
	}
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "skit inspect: expected exactly one skill name or source")
		return 2
	}
	result, err := app.Inspect(app.InspectRequest{CWD: cwd(), Scope: opts.scope, Target: rest[0], Skill: opts.skill})
	if err != nil {
		fmt.Fprintln(stderr, "skit inspect:", err)
		return 1
	}
	if opts.json {
		return writeJSON(stdout, stderr, result)
	}
	fmt.Fprintf(stdout, "name: %s\n", result.Name)
	fmt.Fprintf(stdout, "description: %s\n", result.Description)
	fmt.Fprintf(stdout, "source: %s %s\n", result.Source.Type, result.Source.Locator)
	if result.Source.Ref != "" {
		fmt.Fprintf(stdout, "ref: %s\n", result.Source.Ref)
	}
	if result.Source.ResolvedRef != "" {
		fmt.Fprintf(stdout, "resolvedRef: %s\n", result.Source.ResolvedRef)
	}
	if result.Source.Subpath != "" {
		fmt.Fprintf(stdout, "subpath: %s\n", result.Source.Subpath)
	}
	if result.Hashes.Tree != "" {
		fmt.Fprintf(stdout, "tree: %s\n", result.Hashes.Tree)
	}
	if result.Hashes.SkillMD != "" {
		fmt.Fprintf(stdout, "skillMd: %s\n", result.Hashes.SkillMD)
	}
	if result.StorePath != "" {
		fmt.Fprintf(stdout, "store: %s\n", result.StorePath)
	}
	printList(stdout, "bins", result.Requires.Bins)
	printList(stdout, "anyBins", result.Requires.AnyBins)
	printList(stdout, "env", result.Requires.Env)
	printList(stdout, "config", result.Requires.Config)
	printDependencies(stdout, result.Dependencies)
	printList(stdout, "files", result.Files)
	printList(stdout, "warnings", result.Warnings)
	return 0
}

func runDoctor(args []string, stdout, stderr io.Writer) int {
	opts, rest, err := parseCommon(args)
	if err != nil {
		fmt.Fprintln(stderr, "skit doctor:", err)
		return 2
	}
	if len(rest) != 0 {
		fmt.Fprintln(stderr, "skit doctor: unexpected arguments")
		return 2
	}
	result, err := app.Doctor(app.DoctorRequest{CWD: cwd(), Scope: opts.scope})
	if err != nil {
		fmt.Fprintln(stderr, "skit doctor:", err)
		return 1
	}
	if opts.json {
		if code := writeJSON(stdout, stderr, groupChecks(result.Checks)); code != 0 {
			return code
		}
		if hasErrorCheck(result.Checks) {
			return 1
		}
		return 0
	}
	if len(result.Checks) == 0 {
		fmt.Fprintln(stdout, "ok")
		return 0
	}
	exit := 0
	for _, check := range result.Checks {
		if check.Severity == "error" {
			exit = 1
		}
		if check.Skill != "" {
			fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", check.Severity, check.Code, check.Skill, check.Message)
		} else {
			fmt.Fprintf(stdout, "%s\t%s\t%s\n", check.Severity, check.Code, check.Message)
		}
	}
	return exit
}

func runInit(args []string, stdout, stderr io.Writer) int {
	opts, rest, err := parseCommon(args)
	if err != nil {
		fmt.Fprintln(stderr, "skit init:", err)
		return 2
	}
	if opts.scope == app.Global || opts.skill != "" || opts.all {
		fmt.Fprintln(stderr, "skit init: --global, --skill, and --all are not supported")
		return 2
	}
	if len(rest) > 1 {
		fmt.Fprintln(stderr, "skit init: expected zero or one skill name")
		return 2
	}
	name := ""
	if len(rest) == 1 {
		name = rest[0]
	}
	result, err := app.Init(app.InitRequest{CWD: cwd(), Name: name})
	if err != nil {
		fmt.Fprintln(stderr, "skit init:", err)
		return 1
	}
	if opts.json {
		return writeJSON(stdout, stderr, result)
	}
	fmt.Fprintf(stdout, "created %s\n", result.Path)
	return 0
}

func runImportLock(args []string, stdout, stderr io.Writer) int {
	opts, rest, err := parseCommon(args)
	if err != nil {
		fmt.Fprintln(stderr, "skit import-lock:", err)
		return 2
	}
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "skit import-lock: expected lock kind")
		return 2
	}
	result, err := app.ImportLock(app.ImportLockRequest{CWD: cwd(), Scope: opts.scope, Kind: rest[0]})
	if err != nil {
		fmt.Fprintln(stderr, "skit import-lock:", err)
		return 1
	}
	if opts.json {
		return writeJSON(stdout, stderr, result)
	}
	for _, entry := range result.Entries {
		fmt.Fprintf(stdout, "imported %s incomplete\n", entry.Name)
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}
	return 0
}

type doctorJSON struct {
	Errors   []app.DoctorCheck `json:"errors,omitempty"`
	Warnings []app.DoctorCheck `json:"warnings,omitempty"`
	Info     []app.DoctorCheck `json:"info,omitempty"`
}

func groupChecks(checks []app.DoctorCheck) doctorJSON {
	var out doctorJSON
	for _, check := range checks {
		switch check.Severity {
		case "error":
			out.Errors = append(out.Errors, check)
		case "warning":
			out.Warnings = append(out.Warnings, check)
		default:
			out.Info = append(out.Info, check)
		}
	}
	return out
}

func hasErrorCheck(checks []app.DoctorCheck) bool {
	for _, check := range checks {
		if check.Severity == "error" {
			return true
		}
	}
	return false
}

func writeJSON(stdout, stderr io.Writer, v any) int {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, "skit:", err)
		return 1
	}
	if _, err := stdout.Write(append(raw, '\n')); err != nil {
		fmt.Fprintln(stderr, "skit:", err)
		return 1
	}
	return 0
}

func printList(w io.Writer, label string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(w, "%s:\n", label)
	for _, item := range items {
		fmt.Fprintf(w, "  %s\n", item)
	}
}

func printDependencies(w io.Writer, deps []lockfile.Dependency) {
	if len(deps) == 0 {
		return
	}
	fmt.Fprintln(w, "dependencies:")
	for _, dep := range deps {
		optional := ""
		if dep.Optional {
			optional = " optional"
		}
		fmt.Fprintf(w, "  %s\t%s\t%s%s\n", dep.Name, dep.Source.Type, dep.Source.Locator, optional)
	}
}

type commonOptions struct {
	scope      app.Scope
	skill      string
	all        bool
	json       bool
	ignoreDeps bool
	fullDepth  bool
}

func parseCommon(args []string) (commonOptions, []string, error) {
	opts := commonOptions{scope: app.Project}
	var rest []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--project":
			if opts.scope == app.Global {
				return opts, nil, fmt.Errorf("--project and --global are mutually exclusive")
			}
			opts.scope = app.Project
		case "-g", "--global":
			if opts.scope == app.Project && hasFlag(args[:i], "--project") {
				return opts, nil, fmt.Errorf("--project and --global are mutually exclusive")
			}
			opts.scope = app.Global
		case "--all":
			if opts.skill != "" {
				return opts, nil, fmt.Errorf("--all and --skill are mutually exclusive")
			}
			opts.all = true
		case "-s", "--skill":
			if opts.all {
				return opts, nil, fmt.Errorf("--all and --skill are mutually exclusive")
			}
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("%s requires a value", arg)
			}
			opts.skill = args[i]
		case "-y", "--yes":
		case "--json":
			opts.json = true
		case "--ignore-deps":
			opts.ignoreDeps = true
		case "--full-depth":
			opts.fullDepth = true
		default:
			if len(arg) > 0 && arg[0] == '-' {
				return opts, nil, fmt.Errorf("unknown flag %s", arg)
			}
			rest = append(rest, arg)
		}
	}
	return opts, rest, nil
}

func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

func cwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}
