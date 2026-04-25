package app

import (
	"context"
	"os"
	"sort"
)

func sortedMapKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func firstExisting(paths ...string) string {
	for _, path := range paths {
		if exists(path) {
			return path
		}
	}
	return ""
}

func appendUnique(dst []string, items ...string) []string {
	seen := make(map[string]bool, len(dst)+len(items))
	for _, item := range dst {
		seen[item] = true
	}
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		dst = append(dst, item)
		seen[item] = true
	}
	return dst
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
func ctx(c context.Context) context.Context {
	if c != nil {
		return c
	}
	return context.Background()
}
