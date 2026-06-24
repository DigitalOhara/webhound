package jsextract

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Finding is one extracted item from a JS file.
type Finding struct {
	Type  string // url | ws | path | env
	Label string // env var name (env type only)
	Value string
}

// JSFile groups all findings extracted from one script URL.
type JSFile struct {
	URL      string
	Findings []Finding
}

// Extractor crawls script tags from the target HTML and analyses each JS file.
type Extractor struct {
	client    *http.Client
	userAgent string
	targetURL string
	base      *url.URL
}

var (
	reScriptSrc = regexp.MustCompile(`(?i)<script[^>]+\bsrc=["']([^"']+)["']`)

	reAbsURL  = regexp.MustCompile("[\"`'](https?://[^\\s\"`'<>{}\\[\\]\\\\|^]{8,})[\"`']")
	reWSURL   = regexp.MustCompile("[\"`'](wss?://[^\\s\"`'<>{}\\[\\]\\\\|^]{4,})[\"`']")
	rePath    = regexp.MustCompile("[\"`'](/(?:api|v[0-9]+|graphql|auth|admin|swagger|health|metrics|internal|webhook|ws|socket|rpc)[^\\s\"`'<>{}\\\\]{2,})[\"`']")
	reEnvURL  = regexp.MustCompile("([A-Z][A-Z0-9_]{3,})\\s*:\\s*[\"`'](https?://[^\"`'\\s]{4,})[\"`']")
	reFetch   = regexp.MustCompile("\\bfetch\\([\"`']([^\"`'\\s)]{4,})[\"`']")
	reAxios   = regexp.MustCompile("\\baxios\\.\\w+\\([\"`']([^\"`'\\s)]{4,})[\"`']")
	reBaseURL = regexp.MustCompile("\\bbaseURL\\s*[=:]\\s*[\"`']([^\"`'\\s]{4,})[\"`']")
)

// noiseDomains are filtered out of URL findings to avoid framework internals.
var noiseDomains = []string{
	"w3.org", "whatwg.org", "ecma-international.org",
	"reactjs.org", "webpack.js.org", "babeljs.io",
	"mozilla.org", "tc39.es", "npmjs.com", "nodejs.org",
	"fonts.googleapis.com", "use.typekit.net",
	"cdnjs.cloudflare.com", "ajax.googleapis.com",
	"cdn.jsdelivr.net", "unpkg.com", "schemas.xmlsoap.org",
}

func isNoise(v string) bool {
	if len(v) < 10 {
		return true
	}
	low := strings.ToLower(v)
	for _, d := range noiseDomains {
		if strings.Contains(low, d) {
			return true
		}
	}
	return false
}

// New creates an Extractor that reuses the scan's HTTP client.
func New(client *http.Client, userAgent, targetURL string) *Extractor {
	base, _ := url.Parse(targetURL)
	return &Extractor{client: client, userAgent: userAgent, targetURL: targetURL, base: base}
}

// Run fetches the target root HTML, discovers <script src> URLs, fetches and
// extracts from each JS file. Returns one JSFile per script with findings.
func (e *Extractor) Run(ctx context.Context) []JSFile {
	body, err := e.get(ctx, e.targetURL)
	if err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var files []JSFile

	for _, src := range e.scriptSrcs(string(body)) {
		resolved := e.resolve(src)
		if resolved == "" || seen[resolved] {
			continue
		}
		seen[resolved] = true

		jsBody, err := e.get(ctx, resolved)
		if err != nil {
			continue
		}
		f := ExtractFromBody(resolved, jsBody)
		if len(f.Findings) > 0 {
			files = append(files, f)
		}
	}
	return files
}

// ExtractFromBody analyses an already-fetched JS body (e.g. a .js file found
// during the wordlist scan). Returns a JSFile — caller checks len(Findings).
func ExtractFromBody(jsURL string, body []byte) JSFile {
	text := string(body)
	seen := make(map[string]bool)
	var findings []Finding

	add := func(typ, label, val string) {
		val = strings.TrimRight(val, "/.,;)")
		key := typ + "|" + val
		if seen[key] || isNoise(val) {
			return
		}
		seen[key] = true
		findings = append(findings, Finding{Type: typ, Label: label, Value: val})
	}

	for _, m := range reAbsURL.FindAllStringSubmatch(text, -1) {
		add("url", "", m[1])
	}
	for _, m := range reWSURL.FindAllStringSubmatch(text, -1) {
		add("ws", "", m[1])
	}
	for _, m := range rePath.FindAllStringSubmatch(text, -1) {
		add("path", "", m[1])
	}
	for _, m := range reEnvURL.FindAllStringSubmatch(text, -1) {
		add("env", m[1], m[2])
	}
	for _, m := range reFetch.FindAllStringSubmatch(text, -1) {
		v := m[1]
		if strings.HasPrefix(v, "/") {
			add("path", "fetch", v)
		} else if strings.HasPrefix(v, "http") {
			add("url", "fetch", v)
		}
	}
	for _, m := range reAxios.FindAllStringSubmatch(text, -1) {
		v := m[1]
		if strings.HasPrefix(v, "/") {
			add("path", "axios", v)
		} else if strings.HasPrefix(v, "http") {
			add("url", "axios", v)
		}
	}
	for _, m := range reBaseURL.FindAllStringSubmatch(text, -1) {
		add("url", "baseURL", m[1])
	}

	return JSFile{URL: jsURL, Findings: findings}
}

// WriteReport writes a sidecar findings file to outPath.
func WriteReport(outPath string, target string, files []JSFile, startedAt time.Time) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	const div = "────────────────────────────────────────────────────────────────────────"

	total := 0
	for _, jf := range files {
		total += len(jf.Findings)
	}

	fmt.Fprintf(w, "WebHound — JS Endpoint Extraction\n")
	fmt.Fprintf(w, "Started : %s\n", startedAt.UTC().Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(w, "Target  : %s\n", target)
	fmt.Fprintf(w, "JS Files: %d analysed\n", len(files))
	fmt.Fprintf(w, "Findings: %d\n", total)
	fmt.Fprintf(w, "%s\n\n", div)

	for _, jf := range files {
		fmt.Fprintf(w, "[js] %s (%d finding", jf.URL, len(jf.Findings))
		if len(jf.Findings) != 1 {
			fmt.Fprint(w, "s")
		}
		fmt.Fprintln(w, ")")
		for _, fi := range jf.Findings {
			switch fi.Type {
			case "env":
				fmt.Fprintf(w, "  [ENV]   %s → %s\n", fi.Label, fi.Value)
			case "path":
				if fi.Label != "" {
					fmt.Fprintf(w, "  [PATH]  %s (%s)\n", fi.Value, fi.Label)
				} else {
					fmt.Fprintf(w, "  [PATH]  %s\n", fi.Value)
				}
			case "ws":
				fmt.Fprintf(w, "  [WS]    %s\n", fi.Value)
			default:
				if fi.Label != "" {
					fmt.Fprintf(w, "  [URL]   %s (%s)\n", fi.Value, fi.Label)
				} else {
					fmt.Fprintf(w, "  [URL]   %s\n", fi.Value)
				}
			}
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "%s\n", div)
	fmt.Fprintf(w, "Duration: %s\n", time.Since(startedAt).Round(time.Millisecond))

	return w.Flush()
}

func (e *Extractor) scriptSrcs(html string) []string {
	var srcs []string
	for _, m := range reScriptSrc.FindAllStringSubmatch(html, -1) {
		srcs = append(srcs, m[1])
	}
	return srcs
}

func (e *Extractor) resolve(src string) string {
	if e.base == nil {
		return src
	}
	ref, err := url.Parse(src)
	if err != nil {
		return ""
	}
	return e.base.ResolveReference(ref).String()
}

func (e *Extractor) get(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", e.userAgent)
	req.Header.Set("Accept", "*/*")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 5<<20))
}
