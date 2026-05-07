package metadata

import "fmt"

type Skit struct {
	Version      string
	Raw          YAMLMap
	Carrier      string
	Dependencies []Dependency
	Requires     Requires
	Platforms    Platforms
	Keywords     []string
	Registry     Registry
}

type Dependency struct {
	Source   string
	Ref      string
	Skill    string
	Optional bool
}

type Requires struct {
	Bins    []string
	AnyBins []string
	Env     []string
	Config  []string
	Skills  []string
}

type Platforms struct {
	OS   []string
	Arch []string
}

type Registry struct {
	Slug     string
	Homepage string
}

func FromCarriers(frontmatter YAMLMap, manifest YAMLMap, hasManifest bool) (Skit, error) {
	var out Skit
	meta, _ := AsMap(frontmatter["metadata"])
	skitMeta, hasSkit := AsMap(meta["skit"])
	compat := compatibilitySkit(frontmatter)
	if hasSkit && hasManifest {
		return out, fmt.Errorf("metadata.skit and skill.yaml are mutually exclusive")
	}
	if hasSkit {
		out.Carrier = "metadata.skit"
		out.Raw = skitMeta
		decoded, err := decodeSkit(out)
		if err != nil {
			return decoded, err
		}
		return mergeCompatibility(decoded, compat), nil
	}
	if hasManifest {
		if schema, ok := AsString(manifest["schema"]); ok && schema != "skit.skill/v1" {
			return out, fmt.Errorf("unsupported skill.yaml schema %q", schema)
		}
		if _, ok := manifest["name"]; ok {
			return out, fmt.Errorf("skill.yaml must not define name")
		}
		if _, ok := manifest["description"]; ok {
			return out, fmt.Errorf("skill.yaml must not define description")
		}
		if _, ok := manifest["license"]; ok {
			return out, fmt.Errorf("skill.yaml must not define license")
		}
		out.Carrier = "skill.yaml"
		out.Raw = manifest
		decoded, err := decodeSkit(out)
		if err != nil {
			return decoded, err
		}
		return mergeCompatibility(decoded, compat), nil
	}
	compat.Raw = YAMLMap{}
	return compat, nil
}

func decodeSkit(in Skit) (Skit, error) {
	raw := in.Raw
	if v, ok := AsString(raw["version"]); ok {
		in.Version = v
	}
	if deps, ok := raw["dependencies"].([]any); ok {
		for _, item := range deps {
			m, ok := AsMap(item)
			if !ok {
				return in, fmt.Errorf("dependency must be a mapping")
			}
			source, _ := AsString(m["source"])
			if source == "" {
				return in, fmt.Errorf("dependency source is required")
			}
			dep := Dependency{Source: source}
			dep.Ref, _ = AsString(m["ref"])
			dep.Skill, _ = AsString(m["skill"])
			if optional, ok := m["optional"].(bool); ok {
				dep.Optional = optional
			}
			in.Dependencies = append(in.Dependencies, dep)
		}
	}
	if requires, ok := AsMap(raw["requires"]); ok {
		in.Requires = Requires{
			Bins:    stringList(requires["bins"]),
			AnyBins: stringList(requires["anyBins"]),
			Env:     stringList(requires["env"]),
			Config:  stringList(requires["config"]),
			Skills:  stringList(requires["skills"]),
		}
	}
	if platforms, ok := AsMap(raw["platforms"]); ok {
		in.Platforms = Platforms{
			OS:   stringList(platforms["os"]),
			Arch: stringList(platforms["arch"]),
		}
	}
	in.Keywords = stringList(raw["keywords"])
	if registry, ok := AsMap(raw["registry"]); ok {
		in.Registry.Slug, _ = AsString(registry["slug"])
		in.Registry.Homepage, _ = AsString(registry["homepage"])
	}
	return in, nil
}

func stringList(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := AsString(item); ok {
			out = append(out, s)
		}
	}
	return out
}

func compatibilitySkit(frontmatter YAMLMap) Skit {
	block, ok := compatibilityBlock(frontmatter)
	if !ok {
		return Skit{}
	}
	var out Skit
	out.Requires = requiresFromMap(block)
	if primaryEnv, ok := AsString(block["primaryEnv"]); ok && primaryEnv != "" {
		out.Requires.Env = appendStringUnique(out.Requires.Env, primaryEnv)
	}
	if oses := stringList(block["os"]); len(oses) > 0 {
		for _, os := range oses {
			out.Platforms.OS = appendStringUnique(out.Platforms.OS, normalizeOS(os))
		}
	} else if osName, ok := AsString(block["os"]); ok && osName != "" {
		out.Platforms.OS = appendStringUnique(out.Platforms.OS, normalizeOS(osName))
	}
	if homepage, ok := AsString(block["homepage"]); ok {
		out.Registry.Homepage = homepage
	}
	return out
}

func compatibilityBlock(frontmatter YAMLMap) (YAMLMap, bool) {
	if meta, ok := AsMap(frontmatter["metadata"]); ok {
		for _, key := range []string{"clawdbot", "clawdis", "openclaw"} {
			if block, ok := AsMap(meta[key]); ok && len(block) > 0 {
				return block, true
			}
		}
	}
	if block, ok := AsMap(frontmatter["clawdis"]); ok && len(block) > 0 {
		return block, true
	}
	fallback := YAMLMap{}
	for _, key := range []string{"requires", "primaryEnv", "homepage", "os"} {
		if v, ok := frontmatter[key]; ok {
			fallback[key] = v
		}
	}
	if len(fallback) > 0 {
		return fallback, true
	}
	return nil, false
}

func requiresFromMap(block YAMLMap) Requires {
	requires, _ := AsMap(block["requires"])
	return Requires{
		Bins:    stringList(requires["bins"]),
		AnyBins: stringList(requires["anyBins"]),
		Env:     stringList(requires["env"]),
		Config:  stringList(requires["config"]),
		Skills:  stringList(requires["skills"]),
	}
}

func mergeCompatibility(explicit, compat Skit) Skit {
	if len(explicit.Requires.Bins) == 0 {
		explicit.Requires.Bins = compat.Requires.Bins
	}
	if len(explicit.Requires.AnyBins) == 0 {
		explicit.Requires.AnyBins = compat.Requires.AnyBins
	}
	if len(explicit.Requires.Env) == 0 {
		explicit.Requires.Env = compat.Requires.Env
	}
	if len(explicit.Requires.Config) == 0 {
		explicit.Requires.Config = compat.Requires.Config
	}
	if len(explicit.Requires.Skills) == 0 {
		explicit.Requires.Skills = compat.Requires.Skills
	}
	if len(explicit.Platforms.OS) == 0 {
		explicit.Platforms.OS = compat.Platforms.OS
	}
	if len(explicit.Platforms.Arch) == 0 {
		explicit.Platforms.Arch = compat.Platforms.Arch
	}
	if explicit.Registry.Homepage == "" {
		explicit.Registry.Homepage = compat.Registry.Homepage
	}
	return explicit
}

func normalizeOS(os string) string {
	switch os {
	case "macos":
		return "darwin"
	case "win32":
		return "windows"
	default:
		return os
	}
}

func appendStringUnique(items []string, item string) []string {
	if item == "" {
		return items
	}
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}
