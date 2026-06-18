package wordlist

import (
	"strings"

	"github.com/digitalohara/webhound/pkg/utils"
)

// Generator expands base wordlist entries by appending extensions.
type Generator struct {
	mgr        *Manager
	extensions []string
	noExt      bool
}

// NewGenerator creates a Generator.
// If noExt is true, no extension variants are added.
func NewGenerator(mgr *Manager, extensions []string, noExt bool) *Generator {
	return &Generator{
		mgr:        mgr,
		extensions: extensions,
		noExt:      noExt,
	}
}

// ExpandedPaths returns all paths after extension expansion, deduplicated.
// Paths that already contain a "." are not extended (they have their own ext).
// Directory-style paths (no dot) get both their bare form and ext variants.
func (g *Generator) ExpandedPaths() []string {
	entries := g.mgr.Entries()
	var all []string

	for _, entry := range entries {
		// Always include the bare path.
		all = append(all, entry)

		// Skip extension expansion if the entry already has an extension
		// or looks like an absolute path with a dot in a component.
		if g.noExt || looksExtended(entry) {
			continue
		}

		for _, ext := range g.extensions {
			ext = strings.TrimPrefix(ext, ".")
			if ext == "" {
				continue
			}
			all = append(all, entry+"."+ext)
		}
	}

	return utils.DeduplicateStrings(all)
}

// looksExtended returns true if the entry already carries an extension.
func looksExtended(entry string) bool {
	base := entry
	if idx := strings.LastIndex(entry, "/"); idx >= 0 {
		base = entry[idx+1:]
	}
	if base == "" {
		return true
	}
	dotIdx := strings.LastIndex(base, ".")
	if dotIdx < 0 {
		return false
	}
	// If the dot is at the start (hidden file like .env), treat as extended.
	if dotIdx == 0 {
		return true
	}
	// Heuristic: extension must be <= 10 chars.
	ext := base[dotIdx+1:]
	return len(ext) > 0 && len(ext) <= 10
}
