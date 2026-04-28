package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vlln/skit/internal/app"
	"github.com/vlln/skit/internal/lockfile"
	"github.com/vlln/skit/internal/updatecheck"
)

var version = "0.1.0-dev"

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "-h", "--help":
		if len(args) > 1 {
			return printCommandHelp(args[1], stdout, stderr)
		}
		printHelp(stdout)
		return 0
	case "help":
		if len(args) > 1 {
			return printCommandHelp(args[1], stdout, stderr)
		}
		printHelp(stdout)
		return 0
	case "version", "--version", "-v":
		if len(args) > 1 && args[1] == "--check" {
			return runVersionCheck(stdout, stderr)
		}
		if len(args) > 1 {
			fmt.Fprintf(stderr, "skit version: unknown argument %q\n", args[1])
			return 2
		}
		fmt.Fprintf(stdout, "skit %s\n", version)
		return 0
	case "search", "find":
		if helpRequested(args[1:]) {
			return printCommandHelp("search", stdout, stderr)
		}
		return runSearch(args[1:], stdout, stderr)
	case "install":
		if helpRequested(args[1:]) {
			return printCommandHelp("install", stdout, stderr)
		}
		return runInstall(args[1:], stdout, stderr)
	case "list", "ls":
		if helpRequested(args[1:]) {
			return printCommandHelp("list", stdout, stderr)
		}
		return runList(args[1:], stdout, stderr)
	case "remove", "rm", "uninstall":
		if helpRequested(args[1:]) {
			return printCommandHelp("remove", stdout, stderr)
		}
		return runRemove(args[1:], stdout, stderr)
	case "gc":
		if helpRequested(args[1:]) {
			return printCommandHelp("gc", stdout, stderr)
		}
		return runGC(args[1:], stdout, stderr)
	case "update":
		if helpRequested(args[1:]) {
			return printCommandHelp("update", stdout, stderr)
		}
		return runUpdate(args[1:], stdout, stderr)
	case "inspect":
		return runInspect(args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "import-lock":
		if helpRequested(args[1:]) {
			return printCommandHelp("import-lock", stdout, stderr)
		}
		return runImportLock(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "skit: unknown command %q\n\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func helpRequested(args []string) bool {
	return len(args) == 1 && (args[0] == "-h" || args[0] == "--help" || args[0] == "help")
}

func printCommandHelp(command string, stdout, stderr io.Writer) int {
	switch command {
	case "search", "find":
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  skit search <query> [flags]")
		fmt.Fprintln(stdout, "  skit search <query> --source <repo-or-path> [flags]")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Search for Skills. Without --source, query the configured remote")
		fmt.Fprintln(stdout, "search API. With --source, fetch or read one repository/path, discover")
		fmt.Fprintln(stdout, "Skills in it, and search locally without adding registry state.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Source examples:")
		fmt.Fprintln(stdout, "  skit search pdf --source github:owner/repo")
		fmt.Fprintln(stdout, "  skit search deploy --source ./awesome-skills")
		fmt.Fprintln(stdout, "  skit search lint --source https://github.com/owner/repo/tree/main/skills")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Useful flags:")
		fmt.Fprintln(stdout, "  --source        Search one source repository or local path")
		fmt.Fprintln(stdout, "  --full-depth    With --source, recursively search deeper Skill folders")
		fmt.Fprintln(stdout, "  --json          Print JSON")
	case "install":
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  skit install [source...] [flags]")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Install sources into the content-addressed store, write skit.lock,")
		fmt.Fprintln(stdout, "and create active symlinks unless --no-active is set. With no source,")
		fmt.Fprintln(stdout, "restore active symlinks from skit.lock.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Useful flags:")
		fmt.Fprintln(stdout, "  --project       Use .agents/skills in this project (default)")
		fmt.Fprintln(stdout, "  -g, --global    Use ~/.agents/skills")
		fmt.Fprintln(stdout, "  -a, --agent     Also activate for one or more supported agents")
		fmt.Fprintln(stdout, "  -s, --skill     Select one or more Skills from one source")
		fmt.Fprintln(stdout, "  --all           Install every discovered non-internal Skill")
		fmt.Fprintln(stdout, "  --full-depth    Search recursively when installing a source")
		fmt.Fprintln(stdout, "  --ignore-deps   Skip declared Skill dependencies")
		fmt.Fprintln(stdout, "  --no-active     Write store and lock only")
		fmt.Fprintln(stdout, "  --force         Replace an existing non-symlink active path")
		fmt.Fprintln(stdout, "  --json          Print JSON")
	case "list", "ls":
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  skit list [flags]")
		fmt.Fprintln(stdout, "  skit list --store [flags]")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "List Skills recorded in the selected skit.lock. Output shows")
		fmt.Fprintln(stdout, "name, source locator, and resolved ref or tree hash.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "With --store, list snapshots in the shared content-addressed store.")
		fmt.Fprintln(stdout, "Store output uses short tree IDs and a compact USE column:")
		fmt.Fprintln(stdout, "locked, active, active,locked, or orphan.")
		fmt.Fprintln(stdout, "Add --locks and optional names to show which locks reference snapshots.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Useful flags:")
		fmt.Fprintln(stdout, "  --project       Read project lock (default)")
		fmt.Fprintln(stdout, "  -g, --global    Read global lock")
		fmt.Fprintln(stdout, "  -a, --agent     Read skit.lock from one agent active root")
		fmt.Fprintln(stdout, "  --store         List shared store snapshots instead of lock entries")
		fmt.Fprintln(stdout, "  --locks         With --store, show lock owners")
		fmt.Fprintln(stdout, "  --json          Print JSON")
	case "remove", "rm", "uninstall":
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  skit remove <name...> [flags]")
		fmt.Fprintln(stdout, "  skit remove --all [flags]")
		fmt.Fprintln(stdout, "  skit remove --store <name> [tree-prefix]")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Remove Skills from the selected skit.lock and delete their active")
		fmt.Fprintln(stdout, "symlinks. Store snapshots are kept by default because another project")
		fmt.Fprintln(stdout, "or the global lock may still reference the same immutable content.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "--prune deletes the removed snapshot only when no known project or")
		fmt.Fprintln(stdout, "global lock still references it. Use skit gc for bulk store cleanup.")
		fmt.Fprintln(stdout, "--store removes an orphan store snapshot directly. It refuses locked")
		fmt.Fprintln(stdout, "or active snapshots. tree-prefix may include or omit sha256-.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Useful flags:")
		fmt.Fprintln(stdout, "  --project       Remove from project lock and active root (default)")
		fmt.Fprintln(stdout, "  -g, --global    Remove from global lock and active root")
		fmt.Fprintln(stdout, "  -a, --agent     Remove from one agent active root skit.lock")
		fmt.Fprintln(stdout, "  --all           Remove every locked Skill in the selected scope")
		fmt.Fprintln(stdout, "  --prune         Also delete unreferenced store snapshots")
		fmt.Fprintln(stdout, "  --store         Remove an orphan store snapshot by name and optional tree prefix")
	case "gc":
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  skit gc [--json]")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Garbage collect the shared content-addressed store. skit scans known")
		fmt.Fprintln(stdout, "project and global locks, keeps snapshots referenced by non-incomplete")
		fmt.Fprintln(stdout, "lock entries, and removes unreferenced store directories.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "This is broader than remove --prune: remove --prune only considers")
		fmt.Fprintln(stdout, "the snapshot just removed, while gc scans the whole store.")
	case "update":
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  skit update [name] [flags]")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Refresh locked Skills from their recorded sources. Without name, update")
		fmt.Fprintln(stdout, "all complete lock entries. Local sources are re-read and re-hashed;")
		fmt.Fprintln(stdout, "mutable git refs such as branches resolve to their current commit.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Incomplete imported entries are skipped because they do not contain")
		fmt.Fprintln(stdout, "enough information for reproducible restore; reinstall them with an")
		fmt.Fprintln(stdout, "explicit source to make them complete.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Useful flags:")
		fmt.Fprintln(stdout, "  --project       Update project lock (default)")
		fmt.Fprintln(stdout, "  -g, --global    Update global lock")
		fmt.Fprintln(stdout, "  -a, --agent     Also refresh active links for selected agents")
		fmt.Fprintln(stdout, "  --ignore-deps   Skip declared Skill dependencies")
		fmt.Fprintln(stdout, "  --json          Print JSON")
	case "import-lock":
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  skit import-lock <skills|clawhub> [flags]")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Import lock files from compatible Skill ecosystems into skit.lock.")
		fmt.Fprintln(stdout, "Imports are intentionally conservative: when the source lock lacks")
		fmt.Fprintln(stdout, "skit tree hashes or source archives, entries are marked incomplete.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Supported kinds:")
		fmt.Fprintln(stdout, "  skills   Read ./skills-lock.json")
		fmt.Fprintln(stdout, "  clawhub  Read ./.clawhub/lock.json or legacy ./.clawdhub/lock.json")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Incomplete entries preserve source clues and warnings, but are not")
		fmt.Fprintln(stdout, "automatically restored by skit install. Reinstall with an explicit")
		fmt.Fprintln(stdout, "source to make the Skill fully restorable.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Useful flags:")
		fmt.Fprintln(stdout, "  --project       Write project lock (default)")
		fmt.Fprintln(stdout, "  -g, --global    Write global lock")
		fmt.Fprintln(stdout, "  --json          Print JSON")
	default:
		fmt.Fprintf(stderr, "skit help: unknown command %q\n", command)
		return 2
	}
	return 0
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "Skill Kit (Skill management CLI)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  skit <command> [flags]")
	fmt.Fprintln(w, "  skit help <command>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  search       Search for Skills")
	fmt.Fprintln(w, "  install      Install sources or restore from lock")
	fmt.Fprintln(w, "  list         List locked Skills")
	fmt.Fprintln(w, "  remove       Remove locked and active Skills")
	fmt.Fprintln(w, "  gc           Garbage collect unreferenced store snapshots")
	fmt.Fprintln(w, "  update       Refresh locked Skills from their sources")
	fmt.Fprintln(w, "  inspect      Inspect a locked Skill or source")
	fmt.Fprintln(w, "  doctor       Check lock, store, and declared requirements")
	fmt.Fprintln(w, "  init         Create a SKILL.md template")
	fmt.Fprintln(w, "  import-lock  Import a compatible lock file")
	fmt.Fprintln(w, "  help         Show help")
	fmt.Fprintln(w, "  version      Show version; use --check to check for updates")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Concepts:")
	fmt.Fprintln(w, "  skit.lock records reproducible Skill entries for project or global scope.")
	fmt.Fprintln(w, "  Active Skills are symlinks from .agents/skills or ~/.agents/skills.")
	fmt.Fprintln(w, "  Store snapshots live under XDG_DATA_HOME/skit/store and are shared.")
	fmt.Fprintln(w, "  Incomplete imported entries preserve source clues but are not restorable.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Common flags:")
	fmt.Fprintln(w, "  --project       Use project scope (default)")
	fmt.Fprintln(w, "  -g, --global    Use global scope")
	fmt.Fprintln(w, "  -a, --agent     Also activate for one or more agents")
	fmt.Fprintln(w, "  -s, --skill     Select one or more Skills from one source")
	fmt.Fprintln(w, "  --all           Install every discovered non-internal Skill")
	fmt.Fprintln(w, "  --full-depth    Search recursively when installing a source")
	fmt.Fprintln(w, "  --ignore-deps   Skip declared Skill dependencies")
	fmt.Fprintln(w, "  --no-active     Write store and lock only")
	fmt.Fprintln(w, "  --force         Replace an existing non-symlink active path")
	fmt.Fprintln(w, "  --prune         With remove, delete unreferenced store snapshots")
	fmt.Fprintln(w, "  --store         With list/remove, operate on shared store snapshots")
	fmt.Fprintln(w, "  --locks         With list --store, show lock owners")
	fmt.Fprintln(w, "  --json          Print JSON for supported commands")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'skit help <command>' for command-specific details.")
}

func runSearch(args []string, stdout, stderr io.Writer) int {
	opts, rest, err := parseCommon(args)
	if err != nil {
		fmt.Fprintln(stderr, "skit search:", err)
		return 2
	}
	if opts.scope == app.Global || len(opts.skills) > 0 || len(opts.agents) > 0 || opts.all || opts.ignoreDeps || opts.fullDepth || opts.prune || opts.store || opts.locks {
		if opts.source == "" || !opts.fullDepth {
			fmt.Fprintln(stderr, "skit search: --global, --skill, --agent, --all, --ignore-deps, --prune, --store, and --locks are not supported")
			return 2
		}
		if opts.scope == app.Global || len(opts.skills) > 0 || len(opts.agents) > 0 || opts.all || opts.ignoreDeps || opts.prune || opts.store || opts.locks {
			fmt.Fprintln(stderr, "skit search: --global, --skill, --agent, --all, --ignore-deps, --prune, --store, and --locks are not supported")
			return 2
		}
	}
	if opts.noActive || opts.force {
		fmt.Fprintln(stderr, "skit search: --no-active and --force are not supported")
		return 2
	}
	query := strings.TrimSpace(strings.Join(rest, " "))
	if query == "" {
		fmt.Fprintln(stderr, "skit search: expected a query")
		return 2
	}
	results, err := app.Search(app.SearchRequest{Context: context.Background(), CWD: cwd(), Query: query, Limit: 10, Source: opts.source, FullDepth: opts.fullDepth})
	if err != nil {
		fmt.Fprintln(stderr, "skit search:", err)
		return 1
	}
	if opts.json {
		return writeJSON(stdout, stderr, results)
	}
	if len(results) == 0 {
		fmt.Fprintf(stdout, "no skills found for %q\n", query)
		return 0
	}
	for _, result := range results {
		source := result.Source
		if source == "" {
			source = result.Slug
		}
		target := source
		if result.Install != "" {
			target = result.Install
		}
		if source != "" && result.Name != "" {
			target = source + "@" + result.Name
		}
		if result.Install != "" {
			target = result.Install
		}
		fmt.Fprint(stdout, target)
		if result.Description != "" {
			fmt.Fprintf(stdout, "\t%s", result.Description)
		}
		if result.Path != "" {
			fmt.Fprintf(stdout, "\t%s", result.Path)
		}
		if result.Installs > 0 {
			fmt.Fprintf(stdout, "\t%s", formatInstalls(result.Installs))
		}
		if result.Slug != "" {
			fmt.Fprintf(stdout, "\thttps://skills.sh/%s", result.Slug)
		}
		fmt.Fprintln(stdout)
	}
	fmt.Fprintln(stdout)
	if opts.source != "" {
		fmt.Fprintln(stdout, "Install with: skit install <printed-install-argument>")
	} else {
		fmt.Fprintln(stdout, "Install with: skit install <source@skill>")
	}
	return 0
}

func formatInstalls(count int) string {
	switch {
	case count >= 1_000_000:
		return fmt.Sprintf("%.1fM installs", trimFloat(float64(count)/1_000_000))
	case count >= 1_000:
		return fmt.Sprintf("%.1fK installs", trimFloat(float64(count)/1_000))
	case count == 1:
		return "1 install"
	default:
		return fmt.Sprintf("%d installs", count)
	}
}

func trimFloat(v float64) float64 {
	return float64(int(v*10)) / 10
}

func runInstallSource(args []string, stdout, stderr io.Writer) int {
	opts, rest, err := parseCommon(args)
	if err != nil {
		fmt.Fprintln(stderr, "skit install:", err)
		return 2
	}
	if len(rest) == 0 {
		return runRestore(opts, stdout, stderr)
	}
	if opts.store {
		fmt.Fprintln(stderr, "skit install: --store is not supported")
		return 2
	}
	if opts.locks {
		fmt.Fprintln(stderr, "skit install: --locks is not supported")
		return 2
	}
	if len(rest) > 1 && len(opts.skills) > 0 {
		fmt.Fprintln(stderr, "skit install: --skill can only be used with one source; use source@skill for multiple sources")
		return 2
	}
	for _, src := range rest {
		result, err := app.Add(app.AddRequest{
			CWD:        cwd(),
			Scope:      opts.scope,
			Source:     src,
			Skills:     opts.skills,
			Agents:     opts.agents,
			All:        opts.all,
			IgnoreDeps: opts.ignoreDeps,
			FullDepth:  opts.fullDepth,
			NoActive:   opts.noActive,
			Force:      opts.force,
		})
		if err != nil {
			fmt.Fprintln(stderr, "skit install:", err)
			return 1
		}
		printAddResult(stdout, stderr, result)
	}
	maybePrintUpdate(stderr)
	return 0
}

func printAddResult(stdout, stderr io.Writer, result app.AddResult) {
	for _, entry := range result.DependencyEntries {
		fmt.Fprintf(stdout, "added dependency %s %s\n", entry.Name, entry.Hashes.Tree)
	}
	for _, entry := range result.Entries {
		fmt.Fprintf(stdout, "added %s %s\n", entry.Name, entry.Hashes.Tree)
	}
	for _, warning := range compactWarnings(result.Warnings) {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}
	for _, path := range result.ActivePaths {
		fmt.Fprintf(stdout, "active %s\n", path)
	}
}

func compactWarnings(warnings []string) []string {
	const executablePrefix = "executable file in Skill directory: "
	var out []string
	var executable []string
	seen := map[string]bool{}
	for _, warning := range warnings {
		if strings.HasPrefix(warning, executablePrefix) {
			executable = append(executable, strings.TrimPrefix(warning, executablePrefix))
			continue
		}
		if !seen[warning] {
			out = append(out, warning)
			seen[warning] = true
		}
	}
	if len(executable) == 1 {
		out = append(out, executablePrefix+executable[0])
	} else if len(executable) > 1 {
		out = append(out, fmt.Sprintf("%d executable files in Skill directory: %s", len(executable), strings.Join(executable, ", ")))
	}
	return out
}

func runVersionCheck(stdout, stderr io.Writer) int {
	fmt.Fprintf(stdout, "skit %s\n", version)
	result, err := updatecheck.Check(context.Background(), updatecheck.Request{Current: version, Force: true})
	if err != nil {
		fmt.Fprintln(stderr, "skit version:", err)
		return 1
	}
	if result.Available {
		fmt.Fprintln(stderr, updatecheck.Message(result))
	} else {
		fmt.Fprintln(stdout, "skit is up to date")
	}
	return 0
}

func maybePrintUpdate(stderr io.Writer) {
	result, err := updatecheck.Check(context.Background(), updatecheck.Request{Current: version})
	if err != nil || !result.Available {
		return
	}
	fmt.Fprintln(stderr, updatecheck.Message(result))
}

func runInstall(args []string, stdout, stderr io.Writer) int {
	return runInstallSource(args, stdout, stderr)
}

func runRestore(opts commonOptions, stdout, stderr io.Writer) int {
	if opts.all || len(opts.skills) > 0 || opts.ignoreDeps || opts.fullDepth || opts.noActive || opts.force || opts.prune || opts.store || opts.locks {
		fmt.Fprintln(stderr, "skit install: flags require at least one source")
		return 2
	}
	result, err := app.Install(app.InstallRequest{CWD: cwd(), Scope: opts.scope, Agents: opts.agents})
	if err != nil {
		fmt.Fprintln(stderr, "skit install:", err)
		return 1
	}
	for _, entry := range result.Restored {
		fmt.Fprintf(stdout, "restored %s %s\n", entry.Name, entry.Hashes.Tree)
	}
	for _, path := range result.ActivePaths {
		fmt.Fprintf(stdout, "active %s\n", path)
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
	if len(rest) != 0 && !opts.store {
		fmt.Fprintln(stderr, "skit list: unexpected arguments")
		return 2
	}
	if opts.prune {
		fmt.Fprintln(stderr, "skit list: --prune is not supported")
		return 2
	}
	if opts.locks && !opts.store {
		fmt.Fprintln(stderr, "skit list: --locks requires --store")
		return 2
	}
	if opts.store {
		if len(opts.agents) > 0 {
			fmt.Fprintln(stderr, "skit list: --agent is not supported with --store")
			return 2
		}
		if opts.scope == app.Global {
			fmt.Fprintln(stderr, "skit list: --global is not supported with --store")
			return 2
		}
		entries, err := app.ListStore(app.ListStoreRequest{CWD: cwd(), Names: rest, IncludeLocks: opts.locks})
		if err != nil {
			fmt.Fprintln(stderr, "skit list:", err)
			return 1
		}
		if opts.json {
			return writeJSON(stdout, stderr, entries)
		}
		printStoreList(stdout, entries, opts.locks)
		return 0
	}
	entries, err := app.List(app.ListRequest{CWD: cwd(), Scope: opts.scope, Agents: opts.agents})
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

func printStoreList(stdout io.Writer, entries []app.StoreListEntry, showLocks bool) {
	nameWidth := len("NAME")
	treeWidth := len("TREE")
	useWidth := len("USE")
	locksWidth := len("LOCKS")
	for _, entry := range entries {
		nameWidth = maxInt(nameWidth, len(entry.Name))
		treeWidth = maxInt(treeWidth, len(entry.Tree))
		useWidth = maxInt(useWidth, len(strings.Join(entry.Use, ",")))
		locksWidth = maxInt(locksWidth, len(strings.Join(entry.Locks, ",")))
	}
	if showLocks {
		fmt.Fprintf(stdout, "%-*s  %-*s  %-*s  %-*s\n", nameWidth, "NAME", treeWidth, "TREE", useWidth, "USE", locksWidth, "LOCKS")
	} else {
		fmt.Fprintf(stdout, "%-*s  %-*s  %-*s\n", nameWidth, "NAME", treeWidth, "TREE", useWidth, "USE")
	}
	for _, entry := range entries {
		if showLocks {
			fmt.Fprintf(stdout, "%-*s  %-*s  %-*s  %-*s\n", nameWidth, entry.Name, treeWidth, entry.Tree, useWidth, strings.Join(entry.Use, ","), locksWidth, strings.Join(entry.Locks, ","))
		} else {
			fmt.Fprintf(stdout, "%-*s  %-*s  %-*s\n", nameWidth, entry.Name, treeWidth, entry.Tree, useWidth, strings.Join(entry.Use, ","))
		}
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func runRemove(args []string, stdout, stderr io.Writer) int {
	opts, rest, err := parseCommon(args)
	if err != nil {
		fmt.Fprintln(stderr, "skit remove:", err)
		return 2
	}
	if opts.store {
		return runRemoveStore(opts, rest, stdout, stderr)
	}
	if opts.all && len(rest) > 0 {
		fmt.Fprintln(stderr, "skit remove: --all cannot be combined with skill names")
		return 2
	}
	if !opts.all && len(rest) == 0 {
		fmt.Fprintln(stderr, "skit remove: expected at least one skill name or --all")
		return 2
	}
	if opts.locks {
		fmt.Fprintln(stderr, "skit remove: --locks is not supported")
		return 2
	}
	names := rest
	if opts.all {
		entries, err := app.List(app.ListRequest{CWD: cwd(), Scope: opts.scope, Agents: opts.agents})
		if err != nil {
			fmt.Fprintln(stderr, "skit remove:", err)
			return 1
		}
		for _, entry := range entries {
			names = append(names, entry.Name)
		}
	}
	exit := 0
	for _, name := range names {
		result, err := app.Remove(app.RemoveRequest{CWD: cwd(), Scope: opts.scope, Name: name, Prune: opts.prune, Agents: opts.agents})
		if err != nil {
			fmt.Fprintln(stderr, "skit remove:", err)
			return 1
		}
		if !result.Removed {
			fmt.Fprintf(stderr, "skit remove: %s is not installed\n", name)
			exit = 1
			continue
		}
		fmt.Fprintf(stdout, "removed %s\n", name)
		for _, path := range result.Pruned {
			fmt.Fprintf(stdout, "pruned %s\n", path)
		}
		for _, path := range result.Skipped {
			fmt.Fprintf(stderr, "kept referenced store %s\n", path)
		}
	}
	return exit
}

func runRemoveStore(opts commonOptions, rest []string, stdout, stderr io.Writer) int {
	if opts.scope == app.Global || opts.all || opts.prune || opts.force || opts.noActive || opts.ignoreDeps || opts.fullDepth || opts.locks || len(opts.skills) > 0 || len(opts.agents) > 0 {
		fmt.Fprintln(stderr, "skit remove: with --store, --global, --all, --prune, --force, --no-active, --ignore-deps, --full-depth, --locks, --agent, and --skill are not supported")
		return 2
	}
	if len(rest) == 0 || len(rest) > 2 {
		fmt.Fprintln(stderr, "skit remove: --store expected skill name and optional tree prefix")
		return 2
	}
	result, err := app.RemoveStore(app.RemoveStoreRequest{CWD: cwd(), Name: rest[0], TreePrefix: optionalArg(rest, 1)})
	if err != nil {
		fmt.Fprintln(stderr, "skit remove:", err)
		return 1
	}
	if opts.json {
		return writeJSON(stdout, stderr, result)
	}
	fmt.Fprintf(stdout, "removed store %s %s\n", result.Name, result.Tree)
	return 0
}

func optionalArg(args []string, index int) string {
	if index >= len(args) {
		return ""
	}
	return args[index]
}

func runGC(args []string, stdout, stderr io.Writer) int {
	opts, rest, err := parseCommon(args)
	if err != nil {
		fmt.Fprintln(stderr, "skit gc:", err)
		return 2
	}
	if len(rest) != 0 {
		fmt.Fprintln(stderr, "skit gc: unexpected arguments")
		return 2
	}
	if opts.scope == app.Global || len(opts.skills) > 0 || len(opts.agents) > 0 || opts.all || opts.ignoreDeps || opts.fullDepth || opts.noActive || opts.force || opts.prune || opts.store || opts.locks {
		fmt.Fprintln(stderr, "skit gc: --global, --skill, --agent, --all, --ignore-deps, --full-depth, --no-active, --force, --prune, --store, and --locks are not supported")
		return 2
	}
	result, err := app.GC(app.GCRequest{CWD: cwd()})
	if err != nil {
		fmt.Fprintln(stderr, "skit gc:", err)
		return 1
	}
	if opts.json {
		return writeJSON(stdout, stderr, result)
	}
	for _, path := range result.Pruned {
		fmt.Fprintf(stdout, "pruned %s\n", path)
	}
	if len(result.Pruned) == 0 {
		fmt.Fprintln(stdout, "nothing to prune")
	}
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
	if opts.prune {
		fmt.Fprintln(stderr, "skit update: --prune is not supported")
		return 2
	}
	if opts.store {
		fmt.Fprintln(stderr, "skit update: --store is not supported")
		return 2
	}
	if opts.locks {
		fmt.Fprintln(stderr, "skit update: --locks is not supported")
		return 2
	}
	name := ""
	if len(rest) == 1 {
		name = rest[0]
	}
	result, err := app.Update(app.UpdateRequest{CWD: cwd(), Scope: opts.scope, Name: name, Agents: opts.agents, IgnoreDeps: opts.ignoreDeps})
	if err != nil {
		fmt.Fprintln(stderr, "skit update:", err)
		return 1
	}
	if opts.json {
		return writeJSON(stdout, stderr, result)
	}
	for _, entry := range result.DependencyEntries {
		fmt.Fprintf(stdout, "updated dependency %s %s\n", entry.Name, entry.Hashes.Tree)
	}
	for _, entry := range result.Entries {
		fmt.Fprintf(stdout, "updated %s %s\n", entry.Name, entry.Hashes.Tree)
	}
	for _, warning := range compactWarnings(result.Warnings) {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}
	maybePrintUpdate(stderr)
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
	if opts.prune {
		fmt.Fprintln(stderr, "skit inspect: --prune is not supported")
		return 2
	}
	if opts.store {
		fmt.Fprintln(stderr, "skit inspect: --store is not supported")
		return 2
	}
	if opts.locks {
		fmt.Fprintln(stderr, "skit inspect: --locks is not supported")
		return 2
	}
	if len(opts.agents) > 0 {
		fmt.Fprintln(stderr, "skit inspect: --agent is not supported")
		return 2
	}
	if len(opts.skills) > 1 {
		fmt.Fprintln(stderr, "skit inspect: expected at most one --skill value")
		return 2
	}
	skillName := ""
	if len(opts.skills) == 1 {
		skillName = opts.skills[0]
	}
	result, err := app.Inspect(app.InspectRequest{CWD: cwd(), Scope: opts.scope, Target: rest[0], Skill: skillName})
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
	if opts.prune {
		fmt.Fprintln(stderr, "skit doctor: --prune is not supported")
		return 2
	}
	if opts.store {
		fmt.Fprintln(stderr, "skit doctor: --store is not supported")
		return 2
	}
	if opts.locks {
		fmt.Fprintln(stderr, "skit doctor: --locks is not supported")
		return 2
	}
	if len(opts.agents) > 0 {
		fmt.Fprintln(stderr, "skit doctor: --agent is not supported")
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
	if opts.scope == app.Global || len(opts.skills) > 0 || len(opts.agents) > 0 || opts.all || opts.prune || opts.store || opts.locks {
		fmt.Fprintln(stderr, "skit init: --global, --skill, --agent, --all, --prune, --store, and --locks are not supported")
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
	if opts.prune {
		fmt.Fprintln(stderr, "skit import-lock: --prune is not supported")
		return 2
	}
	if opts.store {
		fmt.Fprintln(stderr, "skit import-lock: --store is not supported")
		return 2
	}
	if opts.locks {
		fmt.Fprintln(stderr, "skit import-lock: --locks is not supported")
		return 2
	}
	if len(opts.agents) > 0 {
		fmt.Fprintln(stderr, "skit import-lock: --agent is not supported")
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
	skills     []string
	agents     []string
	source     string
	all        bool
	json       bool
	ignoreDeps bool
	fullDepth  bool
	noActive   bool
	force      bool
	prune      bool
	store      bool
	locks      bool
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
			if len(opts.skills) > 0 {
				return opts, nil, fmt.Errorf("--all and --skill are mutually exclusive")
			}
			opts.all = true
		case "-s", "--skill":
			if opts.all {
				return opts, nil, fmt.Errorf("--all and --skill are mutually exclusive")
			}
			if len(opts.skills) > 0 {
				return opts, nil, fmt.Errorf("--skill may only be provided once")
			}
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("%s requires a value", arg)
			}
			for ; i < len(args); i++ {
				if strings.HasPrefix(args[i], "-") {
					i--
					break
				}
				opts.skills = append(opts.skills, args[i])
			}
			if len(opts.skills) == 0 {
				return opts, nil, fmt.Errorf("%s requires a value", arg)
			}
		case "-a", "--agent":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("%s requires a value", arg)
			}
			for ; i < len(args); i++ {
				if strings.HasPrefix(args[i], "-") {
					i--
					break
				}
				opts.agents = append(opts.agents, args[i])
			}
			if len(opts.agents) == 0 {
				return opts, nil, fmt.Errorf("%s requires a value", arg)
			}
		case "--source":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("%s requires a value", arg)
			}
			if opts.source != "" {
				return opts, nil, fmt.Errorf("--source may only be provided once")
			}
			opts.source = args[i]
		case "-y", "--yes":
		case "--json":
			opts.json = true
		case "--ignore-deps":
			opts.ignoreDeps = true
		case "--full-depth":
			opts.fullDepth = true
		case "--no-active":
			opts.noActive = true
		case "--force":
			opts.force = true
		case "--prune":
			opts.prune = true
		case "--store":
			opts.store = true
		case "--locks":
			opts.locks = true
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
