package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/vlln/skit/internal/app"
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
	case "sources", "source":
		if helpRequested(args[1:]) {
			return printCommandHelp("source", stdout, stderr)
		}
		return runSources(args[1:], stdout, stderr)
	case "export", "bundle":
		if helpRequested(args[1:]) {
			return printCommandHelp("export", stdout, stderr)
		}
		return runExport(args[1:], stdout, stderr)
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
	case "update":
		if helpRequested(args[1:]) {
			return printCommandHelp("update", stdout, stderr)
		}
		return runUpdate(args[1:], stdout, stderr)
	case "check", "doctor":
		return runCheck(args[1:], stdout, stderr)
	case "init":
		return runInit(args[1:], stdout, stderr)
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
		fmt.Fprintln(stdout, "  skit install <source>/<skill> [flags]")
		fmt.Fprintln(stdout, "  skit install <manifest.json> [flags]")
		fmt.Fprintln(stdout, "  skit install [flags]    # applies ./skit.json when present")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Install Skills into the local skit data directory and activate")
		fmt.Fprintln(stdout, "them by linking from configured agent skill directories.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "With <source>/<skill> syntax, the source is looked up from")
		fmt.Fprintln(stdout, "configured search sources and the matching skill is installed.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Useful flags:")
		fmt.Fprintln(stdout, "  -a, --agent     Also activate for one or more supported agents")
		fmt.Fprintln(stdout, "  -s, --skill     Select one or more Skills from one source")
		fmt.Fprintln(stdout, "  --name          Install a single Skill under this local folder name")
		fmt.Fprintln(stdout, "  --all           Install every discovered non-internal Skill")
		fmt.Fprintln(stdout, "  --full-depth    Search recursively when installing a source")
		fmt.Fprintln(stdout, "  --force         Replace an existing non-symlink active path")
		fmt.Fprintln(stdout, "  --dry-run       Preview manifest installation without changing state")
		fmt.Fprintln(stdout, "  --json          Print JSON")
	case "source", "sources":
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  skit source")
		fmt.Fprintln(stdout, "  skit source add [name] <url>")
		fmt.Fprintln(stdout, "  skit source remove <name>")
		fmt.Fprintln(stdout, "  skit source enable <name>")
		fmt.Fprintln(stdout, "  skit source disable <name>")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Manage search sources. The type is auto-detected from the URL.")
		fmt.Fprintln(stdout, "Supported URL forms: .json files, GitHub/GitLab repos, local")
		fmt.Fprintln(stdout, "paths, and registry URLs. Name is optional; when omitted it is")
		fmt.Fprintln(stdout, "derived from the URL.")
	case "export", "bundle":
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  skit export [path] [--json]")
		fmt.Fprintln(stdout, "  skit bundle [path] [--json]")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Write the current local manifest to a shareable file.")
		fmt.Fprintln(stdout, "The default path is ./skit.json.")
	case "list", "ls":
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  skit list [--all] [flags]")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "List Skills recorded in the local skit manifest and their")
		fmt.Fprintln(stdout, "active agent links. With --all, also scan supported agent")
		fmt.Fprintln(stdout, "skill directories and show externally installed Skills.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Useful flags:")
		fmt.Fprintln(stdout, "  --all           Include Skills visible in supported agent directories")
		fmt.Fprintln(stdout, "  --json          Print JSON")
	case "remove", "rm", "uninstall":
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  skit remove <name...> [flags]")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Remove installed Skills from the local manifest and active agent")
		fmt.Fprintln(stdout, "links. With --agent, only deactivate that agent.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Useful flags:")
		fmt.Fprintln(stdout, "  -a, --agent     Deactivate only the selected agent")
		fmt.Fprintln(stdout, "  --keep          Keep the local Skill directory")
	case "check", "doctor":
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  skit check [--json]")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Check that manifest entries have local Skill directories and")
		fmt.Fprintln(stdout, "active links pointing to them.")
	case "update":
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  skit update [name] [flags]")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Refresh installed Skills from their recorded sources. Without name,")
		fmt.Fprintln(stdout, "update all manifest entries.")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Useful flags:")
		fmt.Fprintln(stdout, "  -a, --agent     Also refresh active links for selected agents")
		fmt.Fprintln(stdout, "  --json          Print JSON")
	case "init":
		fmt.Fprintln(stdout, "Usage:")
		fmt.Fprintln(stdout, "  skit init <name>")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Create a Skill repository template named <name>-skill with")
		fmt.Fprintln(stdout, "README.md and skills/<name>/SKILL.md.")
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
	fmt.Fprintln(w, "  install      Install Skills and activate agent links")
	fmt.Fprintln(w, "  source        List or edit search sources")
	fmt.Fprintln(w, "  export       Write the local manifest to ./skit.json")
	fmt.Fprintln(w, "  list         List installed Skills")
	fmt.Fprintln(w, "  remove       Remove installed or active Skills")
	fmt.Fprintln(w, "  update       Refresh installed Skills from their sources")
	fmt.Fprintln(w, "  check        Check local Skills and active links")
	fmt.Fprintln(w, "  init         Create a Skill repository template")
	fmt.Fprintln(w, "  help         Show help")
	fmt.Fprintln(w, "  version      Show version; use --check to check for updates")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Concepts:")
	fmt.Fprintln(w, "  Installed Skills live under XDG_DATA_HOME/skit/skills.")
	fmt.Fprintln(w, "  XDG_DATA_HOME/skit/manifest.json records local Skills and sources.")
	fmt.Fprintln(w, "  Active Skills are symlinks from agent skill directories.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Common flags:")
	fmt.Fprintln(w, "  -a, --agent     Also activate for one or more agents")
	fmt.Fprintln(w, "  -s, --skill     Select one or more Skills from one source")
	fmt.Fprintln(w, "  --name          Install a single Skill under this local folder name")
	fmt.Fprintln(w, "  --all           Install every discovered non-internal Skill")
	fmt.Fprintln(w, "  --full-depth    Search recursively when installing a source")
	fmt.Fprintln(w, "  --force         Replace an existing non-symlink active path")
	fmt.Fprintln(w, "  --keep          With remove, keep the local Skill directory")
	fmt.Fprintln(w, "  --dry-run       Preview manifest installation")
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
	if opts.scope == app.Global || len(opts.skills) > 0 || len(opts.agents) > 0 || opts.all || opts.fullDepth {
		if opts.source == "" || !opts.fullDepth {
			fmt.Fprintln(stderr, "skit search: --global, --skill, --agent, and --all are not supported")
			return 2
		}
		if opts.scope == app.Global || len(opts.skills) > 0 || len(opts.agents) > 0 || opts.all {
			fmt.Fprintln(stderr, "skit search: --global, --skill, --agent, and --all are not supported")
			return 2
		}
	}
	if opts.force {
		fmt.Fprintln(stderr, "skit search: --force is not supported")
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
		target := formatSearchTarget(result)
		installs := formatInstalls(result.Installs)
		line := target
		if installs != "" {
			line += "  " + installs
		}
		fmt.Fprintln(stdout, line)
		if result.Description != "" {
			fmt.Fprintf(stdout, "  %s\n", result.Description)
		}
		if result.Slug != "" {
			fmt.Fprintf(stdout, "  https://skills.sh/%s\n", result.Slug)
		}
		fmt.Fprintln(stdout)
	}
	fmt.Fprintln(stdout, "use: skit install <source@skill>")
	return 0
}

func formatSearchTarget(result app.SearchResult) string {
	if result.Install != "" {
		return result.Install
	}
	source := result.Source
	if source == "" {
		source = result.Slug
	}
	if source != "" && result.Name != "" {
		return source + "@" + result.Name
	}
	return source
}

func formatInstalls(count int) string {
	if count <= 0 {
		return ""
	}
	if count >= 1_000_000 {
		return fmt.Sprintf("%.1fM installs", float64(count)/1_000_000)
	}
	if count >= 1_000 {
		return fmt.Sprintf("%.1fK installs", float64(count)/1_000)
	}
	if count == 1 {
		return "1 install"
	}
	return fmt.Sprintf("%d installs", count)
}

func runSources(args []string, stdout, stderr io.Writer) int {
	opts, rest, err := parseCommon(args)
	if err != nil {
		fmt.Fprintln(stderr, "skit source:", err)
		return 2
	}
	if opts.scope == app.Global || len(opts.skills) > 0 || len(opts.agents) > 0 || opts.all || opts.fullDepth || opts.force || opts.keep || opts.dryRun || opts.source != "" || opts.name != "" {
		fmt.Fprintln(stderr, "skit source: unsupported flag")
		return 2
	}
	if len(rest) == 0 {
		sources, err := app.ListSearchSources()
		if err != nil {
			fmt.Fprintln(stderr, "skit source:", err)
			return 1
		}
		if opts.json {
			return writeJSON(stdout, stderr, sources)
		}
		printSources(stdout, sources)
		return 0
	}
	switch rest[0] {
	case "add":
		return runSourceAdd(rest[1:], opts, stdout, stderr)
	case "remove", "rm":
		if len(rest) != 2 {
			fmt.Fprintln(stderr, "skit source remove: expected <name>")
			return 2
		}
		sources, err := app.RemoveSearchSource(rest[1])
		if err != nil {
			fmt.Fprintln(stderr, "skit source remove:", err)
			return 1
		}
		if opts.json {
			return writeJSON(stdout, stderr, sources)
		}
		fmt.Fprintf(stdout, "removed %s\n", rest[1])
		return 0
	case "enable":
		if len(rest) != 2 {
			fmt.Fprintln(stderr, "skit source enable: expected <name>")
			return 2
		}
		sources, err := app.EnableSearchSource(rest[1])
		if err != nil {
			fmt.Fprintln(stderr, "skit source enable:", err)
			return 1
		}
		if opts.json {
			return writeJSON(stdout, stderr, sources)
		}
		fmt.Fprintf(stdout, "enabled %s\n", rest[1])
		return 0
	case "disable":
		if len(rest) != 2 {
			fmt.Fprintln(stderr, "skit source disable: expected <name>")
			return 2
		}
		sources, err := app.DisableSearchSource(rest[1])
		if err != nil {
			fmt.Fprintln(stderr, "skit source disable:", err)
			return 1
		}
		if opts.json {
			return writeJSON(stdout, stderr, sources)
		}
		fmt.Fprintf(stdout, "disabled %s\n", rest[1])
		return 0
	default:
		fmt.Fprintf(stderr, "skit source: unknown action %q\n", rest[0])
		return 2
	}
}

func runSourceAdd(args []string, opts commonOptions, stdout, stderr io.Writer) int {
	var name, url string
	switch len(args) {
	case 3:
		if isSourceTypeName(args[1]) {
			return runSourceAddOld(args, opts, stdout, stderr)
		}
		fmt.Fprintln(stderr, "skit source add: expected [name] <url>")
		return 2
	case 2:
		name = args[0]
		url = args[1]
	case 1:
		url = args[0]
		name = deriveSourceName(url)
	default:
		fmt.Fprintln(stderr, "skit source add: expected [name] <url>")
		return 2
	}
	typ := detectSourceType(url)
	sources, err := app.AddSearchSource(app.SourceAddRequest{Name: name, Type: typ, Source: url})
	if err != nil {
		fmt.Fprintln(stderr, "skit source add:", err)
		return 1
	}
	if opts.json {
		return writeJSON(stdout, stderr, sources)
	}
	fmt.Fprintf(stdout, "added %s (%s)\n", name, typ)
	return 0
}

func runSourceAddOld(args []string, opts commonOptions, stdout, stderr io.Writer) int {
	sources, err := app.AddSearchSource(app.SourceAddRequest{Name: args[0], Type: args[1], Source: args[2]})
	if err != nil {
		fmt.Fprintln(stderr, "skit source add:", err)
		return 1
	}
	if opts.json {
		return writeJSON(stdout, stderr, sources)
	}
	fmt.Fprintf(stdout, "added %s\n", args[0])
	return 0
}

func isSourceTypeName(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "registry", "repo", "json", "local":
		return true
	}
	return false
}

func detectSourceType(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasSuffix(raw, ".json") {
		return "json"
	}
	if strings.HasPrefix(raw, "./") || strings.HasPrefix(raw, "../") || strings.HasPrefix(raw, "/") {
		return "local"
	}
	if strings.Contains(raw, "://") {
		if strings.Contains(raw, "github.com/") || strings.Contains(raw, "gitlab.com/") {
			return "repo"
		}
		return "registry"
	}
	return "repo"
}

func deriveSourceName(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, "://") {
		s := raw
		if idx := strings.Index(s, "?"); idx >= 0 {
			s = s[:idx]
		}
		if idx := strings.Index(s, "#"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimRight(s, "/")
		if idx := strings.LastIndex(s, "/"); idx >= 0 {
			name := s[idx+1:]
			name = strings.TrimSuffix(name, ".json")
			if name != "" {
				return name
			}
		}
		return "source"
	}
	if strings.HasPrefix(raw, ".") || strings.HasPrefix(raw, "/") || strings.HasSuffix(raw, ".json") {
		base := raw
		if idx := strings.LastIndex(base, "/"); idx >= 0 {
			base = base[idx+1:]
		}
		return strings.TrimSuffix(base, ".json")
	}
	parts := strings.SplitN(raw, "/", 3)
	if len(parts) >= 2 {
		repo := parts[1]
		if idx := strings.LastIndex(repo, "@"); idx >= 0 {
			repo = repo[:idx]
		}
		return repo
	}
	return raw
}

func parseNamespacedInstall(raw string) (string, string, bool) {
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, "://") || strings.HasPrefix(raw, ".") || strings.HasPrefix(raw, "/") {
		return "", "", false
	}
	if strings.Contains(raw, "@") {
		return "", "", false
	}
	idx := strings.LastIndex(raw, "/")
	if idx <= 0 || idx >= len(raw)-1 {
		return "", "", false
	}
	sourceName := raw[:idx]
	skillName := raw[idx+1:]
	sources, err := app.ListSearchSources()
	if err != nil {
		return "", "", false
	}
	for _, s := range sources {
		if s.Name == sourceName {
			return sourceName, skillName, true
		}
	}
	return "", "", false
}

func printSources(stdout io.Writer, sources []app.SearchSource) {
	if len(sources) == 0 {
		fmt.Fprintln(stdout, "no sources")
		return
	}
	var rows [][]string
	for _, source := range sources {
		locator := source.Source
		if locator == "" {
			locator = source.URL
		}
		rows = append(rows, []string{source.Name, source.Type, locator})
	}
	writeTable(stdout, rows)
}

func runInstallSource(args []string, stdout, stderr io.Writer) int {
	opts, rest, err := parseCommon(args)
	if err != nil {
		fmt.Fprintln(stderr, "skit install:", err)
		return 2
	}
	if len(rest) == 0 {
		defaultManifest := "skit.json"
		if _, err := os.Stat(defaultManifest); err != nil {
			fmt.Fprintln(stderr, "skit install: expected a source or ./skit.json")
			return 2
		}
		rest = []string{defaultManifest}
	}
	if len(rest) > 1 && len(opts.skills) > 0 {
		fmt.Fprintln(stderr, "skit install: --skill can only be used with one source; use source@skill for multiple sources")
		return 2
	}
	if len(rest) > 1 && opts.name != "" {
		fmt.Fprintln(stderr, "skit install: --name can only be used with one source")
		return 2
	}
	if len(rest) == 1 && opts.name == "" && len(opts.skills) == 0 && isManifestFile(rest[0]) {
		result, err := app.ApplyManifest(app.ApplyManifestRequest{Context: context.Background(), Path: rest[0], Agents: opts.agents, DryRun: opts.dryRun})
		if err != nil {
			fmt.Fprintln(stderr, "skit install:", err)
			return 1
		}
		if opts.json {
			return writeJSON(stdout, stderr, result)
		}
		if opts.dryRun {
			printDryRunResult(stdout, stderr, result)
		} else {
			printAddResult(stdout, stderr, result)
		}
		return 0
	}
	if opts.dryRun {
		fmt.Fprintln(stderr, "skit install: --dry-run is only supported for manifest installs")
		return 2
	}
	for _, src := range rest {
		if sourceName, skillName, ok := parseNamespacedInstall(src); ok {
			if opts.all {
				fmt.Fprintln(stderr, "skit install: --all cannot be used with named source install")
				return 2
			}
			result, err := app.AddFromNamedSource(app.AddFromNamedSourceRequest{
				CWD:        cwd(),
				Scope:      opts.scope,
				SourceName: sourceName,
				SkillName:  skillName,
				Name:       opts.name,
				Agents:     opts.agents,
				Force:      opts.force,
				Progress:   installProgress(stderr, opts.json),
			})
			if err != nil {
				fmt.Fprintln(stderr, "skit install:", err)
				return 1
			}
			printAddResult(stdout, stderr, result)
			continue
		}
		result, err := app.Add(app.AddRequest{
			CWD:       cwd(),
			Scope:     opts.scope,
			Source:    src,
			Name:      opts.name,
			Skills:    opts.skills,
			Agents:    opts.agents,
			All:       opts.all,
			FullDepth: opts.fullDepth,
			Force:     opts.force,
			Progress:  installProgress(stderr, opts.json),
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

func installProgress(stderr io.Writer, quiet bool) func(string) {
	if quiet {
		return nil
	}
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("SKIT_PROGRESS")))
	if mode == "off" || mode == "0" || mode == "false" {
		return nil
	}
	if mode == "ansi" || shouldUseANSI(stderr) {
		frames := []string{"-", "\\", "|", "/"}
		i := 0
		return func(message string) {
			if message == "" {
				return
			}
			fmt.Fprintf(stderr, "\r\033[2K[%s] %s\n", frames[i%len(frames)], message)
			i++
		}
	}
	return func(message string) {
		if message != "" {
			fmt.Fprintln(stderr, message)
		}
	}
}

func shouldUseANSI(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func printDryRunResult(stdout, stderr io.Writer, result app.AddResult) {
	for _, entry := range result.Entries {
		fmt.Fprintf(stdout, "would install %s\n", entry.Name)
	}
	for _, warning := range compactWarnings(result.Warnings) {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}
}

func isManifestFile(path string) bool {
	return strings.HasSuffix(path, ".json")
}

func printAddResult(stdout, stderr io.Writer, result app.AddResult) {
	for _, entry := range result.Entries {
		fmt.Fprintf(stdout, "installed %s\n", entry.Name)
	}
	for _, warning := range compactWarnings(result.Warnings) {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}
	for _, path := range result.ActivePaths {
		fmt.Fprintf(stdout, "active %s\n", path)
	}
}

func runExport(args []string, stdout, stderr io.Writer) int {
	opts, rest, err := parseCommon(args)
	if err != nil {
		fmt.Fprintln(stderr, "skit export:", err)
		return 2
	}
	if opts.scope == app.Global || len(opts.skills) > 0 || len(opts.agents) > 0 || opts.all || opts.fullDepth || opts.force || opts.keep || opts.dryRun || opts.source != "" || opts.name != "" {
		fmt.Fprintln(stderr, "skit export: unsupported flag")
		return 2
	}
	if len(rest) > 1 {
		fmt.Fprintln(stderr, "skit export: expected zero or one path")
		return 2
	}
	path := ""
	if len(rest) == 1 {
		path = rest[0]
	}
	result, err := app.ExportManifest(app.ExportManifestRequest{CWD: cwd(), Path: path})
	if err != nil {
		fmt.Fprintln(stderr, "skit export:", err)
		return 1
	}
	if opts.json {
		return writeJSON(stdout, stderr, result)
	}
	fmt.Fprintf(stdout, "exported %s\n", result.Path)
	return 0
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
	entries, err := app.List(app.ListRequest{CWD: cwd(), Scope: opts.scope, Agents: opts.agents, All: opts.all})
	if err != nil {
		fmt.Fprintln(stderr, "skit list:", err)
		return 1
	}
	if opts.json {
		return writeJSON(stdout, stderr, entries)
	}
	if len(entries) == 0 {
		if opts.all {
			fmt.Fprintln(stdout, "No skills found.")
		} else {
			fmt.Fprintln(stdout, "No skills installed.")
		}
		return 0
	}
	if opts.all {
		printListAll(stdout, entries)
		return 0
	}
	var rows [][]string
	for _, entry := range entries {
		rows = append(rows, []string{entry.Name, listState(entry), entry.Description})
	}
	writeTable(stdout, rows)
	return 0
}

func printListAll(stdout io.Writer, entries []app.ListEntry) {
	printedManaged := false
	var managedRows [][]string
	for _, entry := range entries {
		if !entry.Managed {
			continue
		}
		managedRows = append(managedRows, []string{entry.Name, listState(entry), entry.Description})
	}
	if len(managedRows) > 0 {
		fmt.Fprintln(stdout, "managed")
		writeTable(stdout, managedRows)
		printedManaged = true
	}
	var externalRows [][]string
	for _, entry := range entries {
		if entry.Managed {
			continue
		}
		externalRows = append(externalRows, []string{entry.Name, entry.Path, entry.Description})
	}
	if len(externalRows) > 0 {
		if printedManaged {
			fmt.Fprintln(stdout)
		}
		fmt.Fprintln(stdout, "external")
		writeTable(stdout, externalRows)
	}
}

func listState(entry app.ListEntry) string {
	if entry.Missing {
		return "missing"
	}
	if len(entry.Active) == 0 {
		return "inactive"
	}
	return "active"
}

func writeTable(stdout io.Writer, rows [][]string) {
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	for _, row := range rows {
		last := len(row)
		for last > 0 && row[last-1] == "" {
			last--
		}
		for i := 0; i < last; i++ {
			if i > 0 {
				fmt.Fprint(tw, "\t")
			}
			fmt.Fprint(tw, row[i])
		}
		fmt.Fprintln(tw)
	}
	_ = tw.Flush()
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
	if len(rest) == 0 {
		fmt.Fprintln(stderr, "skit remove: expected at least one skill name")
		return 2
	}
	names := rest
	exit := 0
	for _, name := range names {
		result, err := app.Remove(app.RemoveRequest{CWD: cwd(), Scope: opts.scope, Name: name, Agents: opts.agents, Keep: opts.keep})
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
		for _, path := range result.Unlinked {
			fmt.Fprintf(stdout, "unlinked %s\n", path)
		}
		for _, path := range result.Deleted {
			fmt.Fprintf(stdout, "deleted %s\n", path)
		}
		for _, path := range result.Skipped {
			fmt.Fprintf(stderr, "skipped %s\n", path)
		}
	}
	return exit
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
	result, err := app.Update(app.UpdateRequest{CWD: cwd(), Scope: opts.scope, Name: name, Agents: opts.agents})
	if err != nil {
		fmt.Fprintln(stderr, "skit update:", err)
		return 1
	}
	if opts.json {
		return writeJSON(stdout, stderr, result)
	}
	for _, entry := range result.Entries {
		fmt.Fprintf(stdout, "updated %s\n", entry.Name)
	}
	for _, warning := range compactWarnings(result.Warnings) {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}
	maybePrintUpdate(stderr)
	return 0
}

func runCheck(args []string, stdout, stderr io.Writer) int {
	opts, rest, err := parseCommon(args)
	if err != nil {
		fmt.Fprintln(stderr, "skit check:", err)
		return 2
	}
	if len(rest) != 0 {
		fmt.Fprintln(stderr, "skit check: unexpected arguments")
		return 2
	}
	result, err := app.Doctor(app.DoctorRequest{CWD: cwd(), Scope: opts.scope})
	if err != nil {
		fmt.Fprintln(stderr, "skit check:", err)
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
	if opts.scope == app.Global || len(opts.skills) > 0 || len(opts.agents) > 0 || opts.all {
		fmt.Fprintln(stderr, "skit init: --global, --skill, --agent, and --all are not supported")
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
	fmt.Fprintf(stdout, "created %s\n", result.RepoName)
	fmt.Fprintf(stdout, "readme %s\n", result.README)
	fmt.Fprintf(stdout, "skill %s\n", result.Path)
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

type commonOptions struct {
	scope     app.Scope
	skills    []string
	agents    []string
	source    string
	name      string
	all       bool
	json      bool
	fullDepth bool
	force     bool
	keep      bool
	dryRun    bool
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
		case "--name":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("%s requires a value", arg)
			}
			if opts.name != "" {
				return opts, nil, fmt.Errorf("--name may only be provided once")
			}
			opts.name = args[i]
		case "-y", "--yes":
		case "--json":
			opts.json = true
		case "--full-depth":
			opts.fullDepth = true
		case "--force":
			opts.force = true
		case "--keep":
			opts.keep = true
		case "--dry-run":
			opts.dryRun = true
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
