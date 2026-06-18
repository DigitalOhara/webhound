package utils

import (
	"net/url"
	"os"
	"strings"
)

// DeduplicateStrings returns a deduplicated slice preserving order.
func DeduplicateStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

// FileExists returns true if path exists and is a regular file.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// DirExists returns true if path exists and is a directory.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// ContainsString reports whether needle is in haystack (case-insensitive).
func ContainsString(haystack []string, needle string) bool {
	lower := strings.ToLower(needle)
	for _, s := range haystack {
		if strings.ToLower(s) == lower {
			return true
		}
	}
	return false
}

// JoinURL joins a base URL and a path segment, avoiding double slashes.
func JoinURL(base, path string) string {
	base = strings.TrimRight(base, "/")
	path = strings.TrimLeft(path, "/")
	if path == "" {
		return base + "/"
	}
	return base + "/" + path
}

// ExtractHost returns just the host from a URL string.
func ExtractHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Host
}

// ExtractBasePath extracts the path component, stripping the last segment.
func ExtractBasePath(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "/"
	}
	p := u.Path
	idx := strings.LastIndex(p, "/")
	if idx <= 0 {
		return "/"
	}
	return p[:idx+1]
}

// CountWords counts whitespace-delimited words in s.
func CountWords(s string) int {
	return len(strings.Fields(s))
}

// CountLines counts newline-delimited lines in s.
func CountLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// TruncateString truncates s to max runes, appending "…" if truncated.
func TruncateString(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

// EnsureDir creates a directory and all parents if they don't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0750)
}
