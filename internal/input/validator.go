package input

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateAndNormalize parses, validates, and normalises a raw URL string.
// It enforces http/https schemes, trims trailing slashes, and returns a
// canonical form suitable for use as a scan base URL.
func ValidateAndNormalize(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty URL")
	}

	// Prepend scheme only when there is no scheme at all (no "://").
	// This avoids prepending to URLs that already carry a non-http scheme.
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", raw, err)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("unsupported scheme %q in URL %q (use http or https)", u.Scheme, raw)
	}

	if u.Host == "" {
		return "", fmt.Errorf("URL %q has no host", raw)
	}

	// Lowercase scheme and host for consistency.
	u.Scheme = scheme
	u.Host = strings.ToLower(u.Host)

	// Remove default ports.
	switch {
	case scheme == "http" && strings.HasSuffix(u.Host, ":80"):
		u.Host = u.Host[:len(u.Host)-3]
	case scheme == "https" && strings.HasSuffix(u.Host, ":443"):
		u.Host = u.Host[:len(u.Host)-4]
	}

	// Normalise path: strip trailing slash so JoinURL works predictably.
	u.Path = strings.TrimRight(u.Path, "/")

	// Remove fragment — never useful for server-side requests.
	u.Fragment = ""

	return u.String(), nil
}

// ValidateURLList validates each entry in the slice, returning deduplicated
// valid URLs and a slice of per-entry errors.
func ValidateURLList(raw []string) (valid []string, errs []error) {
	seen := make(map[string]struct{})
	for _, r := range raw {
		norm, err := ValidateAndNormalize(r)
		if err != nil {
			errs = append(errs, fmt.Errorf("skipping %q: %w", r, err))
			continue
		}
		if _, dup := seen[norm]; dup {
			continue
		}
		seen[norm] = struct{}{}
		valid = append(valid, norm)
	}
	return
}
