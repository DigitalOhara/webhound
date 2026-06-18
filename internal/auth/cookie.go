package auth

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ParseCookieString parses "key=value; key2=value2" into a map.
func ParseCookieString(s string) map[string]string {
	result := make(map[string]string)
	for _, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			result[strings.TrimSpace(kv[0])] = ""
			continue
		}
		result[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return result
}

// CookiesToHeader merges multiple cookie strings into a single Cookie header value.
func CookiesToHeader(cookies []string) string {
	merged := make(map[string]string)
	for _, c := range cookies {
		for k, v := range ParseCookieString(c) {
			merged[k] = v
		}
	}
	var parts []string
	for k, v := range merged {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, "; ")
}

// LoadCookieFile loads cookies from a file.
// Supports both Netscape format and simple "key=value" per line.
func LoadCookieFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening cookie file: %w", err)
	}
	defer f.Close()

	var cookies []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Netscape format: domain \t flag \t path \t secure \t expiry \t name \t value
		if strings.Count(line, "\t") >= 6 {
			fields := strings.Split(line, "\t")
			if len(fields) >= 7 {
				cookies = append(cookies, fields[5]+"="+fields[6])
			}
			continue
		}
		// Simple key=value or key=value; key2=value2
		cookies = append(cookies, line)
	}
	return cookies, scanner.Err()
}
