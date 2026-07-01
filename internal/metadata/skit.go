package metadata

// Skit holds skit-specific metadata: structured runtime requirements.
type Skit struct {
	Raw      YAMLMap
	Requires Requires
}

type Requires struct {
	Bins      []string
	AnyBins   []string
	Env       []string
	Config    []string
	Skills    []string
	Platforms Platforms
}

type Platforms struct {
	OS   []string
	Arch []string
}

// FromCarriers extracts skit-specific metadata from the frontmatter's
// top-level requires block, or from metadata.skit.requires (fallback).
func FromCarriers(frontmatter YAMLMap) Skit {
	var out Skit

	// Try top-level requires first (new schema)
	reqs, ok := AsMap(frontmatter["requires"])
	if ok {
		out.Raw = reqs
		out.Requires = decodeRequires(reqs)
		return out
	}

	// Fallback: metadata.skit.requires (old schema)
	meta, _ := AsMap(frontmatter["metadata"])
	if skitMeta, ok := AsMap(meta["skit"]); ok {
		if reqs, ok := AsMap(skitMeta["requires"]); ok {
			out.Raw = reqs
			out.Requires = decodeRequires(reqs)
		}
	}
	return out
}

func decodeRequires(raw YAMLMap) Requires {
	var r Requires
	r.Bins = stringList(raw["bins"])
	r.AnyBins = stringList(raw["any-bins"])
	r.Env = stringList(raw["env"])
	r.Config = stringList(raw["config"])
	r.Skills = stringList(raw["skills"])
	if platforms, ok := AsMap(raw["platforms"]); ok {
		r.Platforms = Platforms{
			OS:   stringList(platforms["os"]),
			Arch: stringList(platforms["arch"]),
		}
	}
	return r
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
