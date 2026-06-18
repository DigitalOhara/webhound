package unit_test

import (
	"strings"
	"testing"

	"github.com/digitalohara/webhound/internal/config"
	"github.com/digitalohara/webhound/internal/wordlist"
)

func TestWordlistManager_LoadBuiltin(t *testing.T) {
	cfg := config.DefaultConfig()
	// No wordlists specified → use built-in common list.
	mgr := wordlist.NewManager(cfg)
	if err := mgr.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if mgr.Count() == 0 {
		t.Error("expected non-empty built-in wordlist")
	}
}

func TestWordlistManager_LoadSpecificBuiltins(t *testing.T) {
	for _, name := range []string{"common", "directories", "files"} {
		t.Run(name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Wordlists = []string{name}
			mgr := wordlist.NewManager(cfg)
			if err := mgr.Load(); err != nil {
				t.Fatalf("Load(%q) error: %v", name, err)
			}
			if mgr.Count() == 0 {
				t.Errorf("built-in wordlist %q is empty", name)
			}
		})
	}
}

func TestWordlistManager_Deduplication(t *testing.T) {
	// Loading two overlapping built-in lists should deduplicate entries.
	cfg := config.DefaultConfig()
	cfg.Wordlists = []string{"common", "common"} // deliberate duplicate
	mgr := wordlist.NewManager(cfg)
	if err := mgr.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	entries := mgr.Entries()
	seen := make(map[string]int)
	for _, e := range entries {
		seen[e]++
		if seen[e] > 1 {
			t.Errorf("duplicate entry found: %q", e)
		}
	}
}

func TestGenerator_ExpandedPaths(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Wordlists = []string{"directories"}
	cfg.Extensions = []string{"php", "html"}
	mgr := wordlist.NewManager(cfg)
	_ = mgr.Load()

	gen := wordlist.NewGenerator(mgr, cfg.Extensions, false)
	paths := gen.ExpandedPaths()

	if len(paths) <= mgr.Count() {
		t.Errorf("expanded paths (%d) should exceed base count (%d)", len(paths), mgr.Count())
	}

	// Every base entry should appear without extension.
	entries := mgr.Entries()
	entrySet := make(map[string]bool, len(entries))
	for _, e := range entries {
		entrySet[e] = true
	}
	for _, p := range paths {
		// paths like "admin.php", "admin.html" should be present.
		_ = p
	}

	// Verify "admin" appears and "admin.php" appears.
	pathSet := make(map[string]bool, len(paths))
	for _, p := range paths {
		pathSet[p] = true
	}
	if entrySet["admin"] && !pathSet["admin"] {
		t.Error("bare 'admin' should be in expanded paths")
	}
	if entrySet["admin"] && !pathSet["admin.php"] {
		t.Error("'admin.php' should be in expanded paths")
	}
}

func TestGenerator_NoExtension(t *testing.T) {
	// Use directories wordlist — no entries have extensions already.
	cfg := config.DefaultConfig()
	cfg.Wordlists = []string{"directories"}
	cfg.Extensions = []string{"php", "html"}
	mgr := wordlist.NewManager(cfg)
	_ = mgr.Load()

	// Capture which entries are bare (no extension) in the base wordlist.
	baseEntries := make(map[string]bool, len(mgr.Entries()))
	for _, e := range mgr.Entries() {
		baseEntries[e] = true
	}

	gen := wordlist.NewGenerator(mgr, cfg.Extensions, true) // noExt = true
	paths := gen.ExpandedPaths()

	for _, p := range paths {
		// If this path IS a base entry (no ext in wordlist), it should not
		// have had a .php or .html suffix appended.
		if !baseEntries[p] {
			for _, ext := range []string{".php", ".html"} {
				if strings.HasSuffix(p, ext) {
					// Only a problem if the BASE entry (without ext) exists in wordlist.
					base := p[:len(p)-len(ext)]
					if baseEntries[base] {
						t.Errorf("path %q has appended extension when noExt=true", p)
					}
				}
			}
		}
	}
}

func TestBuiltinWordlists_NotEmpty(t *testing.T) {
	tests := []struct {
		name     string
		entries  []string
	}{
		{"BuiltinCommon", wordlist.BuiltinCommon()},
		{"BuiltinDirectories", wordlist.BuiltinDirectories()},
		{"BuiltinFiles", wordlist.BuiltinFiles()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.entries) == 0 {
				t.Errorf("%s returned empty slice", tt.name)
			}
		})
	}
}
