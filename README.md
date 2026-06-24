# WebHound

**WebHound** is a fast, flexible web content discovery tool built for security professionals. It performs wordlist-based enumeration against one or more HTTP targets, with built-in support for authentication, rate limiting, wildcard detection, recursive scanning, session management, and structured reporting.

> Authorized security testing only. Ensure you have written permission before scanning any target.

---

## Features

- **Built-in wordlists** — `common`, `directories`, and `files` lists embedded in the binary; no external files required
- **JS endpoint extraction** — automatically crawls `<script src>` tags, fetches JS bundles, and extracts hardcoded URLs, API paths, env vars, WebSocket endpoints, and `fetch`/`axios` calls to a sidecar file
- **Auto-save results** — findings automatically saved to `webhoundresults/<host>-<timestamp>.txt` after every scan
- **Multi-target scanning** — scan a single URL or a file of targets in one run
- **Authentication support** — Bearer token, Basic auth, custom cookies, cookie files, and arbitrary HTTP headers
- **Smart wildcard detection** — probes random paths before scanning and suppresses false positives automatically
- **Recursive scanning** — automatic, interactive, or queue-based recursion into discovered directories
- **Rate limiting & delays** — configurable requests/second and per-worker delays to stay under radar
- **Response filtering** — filter by status code, content length, word count, line count
- **Proxy support** — HTTP/HTTPS and SOCKS5 proxies, custom CA certificates, TLS version control
- **Session & resume** — checkpoints every N requests; interrupted scans can be resumed
- **Multiple output formats** — plain text (default), JSON, CSV, HTML
- **Color terminal output** — status-coded, real-time progress with `--no-color` for CI/logging

---

## Installation

### From GitHub Releases (recommended)

Download the pre-built binary for your platform from the [Releases page](https://github.com/DigitalOhara/webhound/releases/latest):

| Platform | File |
|----------|------|
| Linux x64 | `webhound-linux-amd64` |
| Linux ARM64 | `webhound-linux-arm64` |
| macOS Intel | `webhound-darwin-amd64` |
| macOS Apple Silicon | `webhound-darwin-arm64` |
| Windows x64 | `webhound-windows-amd64.exe` |

```bash
# Linux / macOS
chmod +x webhound-linux-amd64
sudo mv webhound-linux-amd64 /usr/local/bin/webhound

# Verify
webhound --help
```

### Build from Source

Requires **Go 1.22+**.

```bash
git clone https://github.com/DigitalOhara/webhound.git
cd webhound
make deps
make build

# Optional: install to PATH
sudo cp webhound /usr/local/bin/webhound
```

### WSL2 / Kali Linux

```bash
git clone https://github.com/DigitalOhara/webhound.git ~/tools/webhound
cd ~/tools/webhound
make deps && make build
sudo cp webhound /usr/local/bin/webhound
```

---

## Quick Start

```bash
# Basic scan — results auto-saved to webhoundresults/
webhound scan --url https://example.com

# Use built-in wordlist with custom status codes
webhound scan --url https://example.com \
  --wordlist common \
  --status-codes 200,201,301,302,401,403,405,500

# Authenticated scan with Bearer token
webhound scan --url https://api.example.com \
  --bearer-token "eyJhbGci..." \
  --threads 20 --rate 30

# Multiple targets from file
webhound scan --file targets.txt --threads 20 --rate 50

# Save to specific format
webhound scan --url https://example.com -o report.html
webhound scan --url https://example.com -o report.json
webhound scan --url https://example.com -o report.csv
```

---

## Usage

```
webhound scan [flags]
```

### Target

| Flag | Description |
|------|-------------|
| `-u, --url` | Single target URL |
| `-f, --file` | File containing target URLs (one per line) |

### Authentication

| Flag | Description |
|------|-------------|
| `--bearer-token` | Bearer token added to `Authorization` header |
| `--basic-auth` | HTTP Basic credentials (`user:pass`) |
| `-H, --header` | Custom header, repeatable (`-H 'X-API-Key: abc'`) |
| `--cookie` | Cookie string, repeatable (`--cookie 'session=xyz'`) |
| `--cookie-file` | Load cookies from Netscape or `key=value` file |
| `--profile` | Named auth profile from config |

### Wordlists

| Flag | Description |
|------|-------------|
| `-w, --wordlist` | Wordlist file path or built-in name: `common`, `directories`, `files` |
| `-x, --extensions` | File extensions to append (e.g. `php,asp,aspx,js`) |
| `--no-extension` | Skip extension appending entirely |

### HTTP Tuning

| Flag | Default | Description |
|------|---------|-------------|
| `-t, --threads` | `10` | Concurrent workers |
| `-r, --rate` | `50` | Max requests per second (0 = unlimited) |
| `--delay` | `0` | Fixed delay between requests per worker (e.g. `200ms`) |
| `--timeout` | `10s` | HTTP request timeout |
| `--max-retries` | `3` | Retries per failed request |
| `--user-agent` | Chrome UA | Custom User-Agent string |
| `--follow-redirects` | `true` | Follow HTTP redirects |
| `--max-redirects` | `5` | Maximum redirect hops |
| `--method` | `GET` | HTTP method |

### Proxy / TLS

| Flag | Description |
|------|-------------|
| `-p, --proxy` | Proxy URL (`http://`, `https://`, `socks5://`) |
| `--insecure` | Skip TLS certificate verification |
| `--ca-cert` | Custom CA certificate file |
| `--tls-version` | Minimum TLS version (`1.0`, `1.1`, `1.2`) |

### Response Filtering

| Flag | Description |
|------|-------------|
| `--status-codes` | Status codes to include (default: `200,201,202,204,301,302,307,308,401,403,405`) |
| `--status-codes-blacklist` | Status codes to exclude |
| `--min-length` | Minimum content length |
| `--max-length` | Maximum content length (0 = no limit) |
| `--hide-length` | Suppress results with exact content lengths |
| `--hide-words` | Suppress results with exact word counts |
| `--hide-lines` | Suppress results with exact line counts |

### Recursion

| Flag | Default | Description |
|------|---------|-------------|
| `--recursive` | `false` | Automatically recurse into discovered directories |
| `--interactive-recursion` | `false` | Prompt before recursing into each directory |
| `--queue-recursion` | `false` | Collect directories and select at end of level |
| `--max-depth` | `3` | Maximum recursion depth |
| `--smart-recursion` | `true` | Prioritise high-value directories |
| `--exclude-recursion` | — | Path patterns to skip for recursion |

### Output

| Flag | Description |
|------|-------------|
| `-o, --output` | Output file path (format inferred from extension: `.txt`, `.json`, `.csv`, `.html`) |
| `--format` | Explicit output format: `txt`, `json`, `csv`, `html` |
| `-q, --quiet` | Suppress all output except findings |
| `-v, --verbose` | Enable debug logging |
| `--no-color` | Disable colour output |

### JS Extraction

| Flag | Description |
|------|-------------|
| `--no-js-extract` | Disable JS endpoint extraction (enabled by default) |

### Session

| Flag | Description |
|------|-------------|
| `--session` | Session name (default: auto-generated timestamp) |
| `--resume` | Resume a previously interrupted scan by session ID |
| `--checkpoint-interval` | Save checkpoint every N requests (default: 500) |

---

## JS Endpoint Extraction

WebHound automatically extracts hardcoded endpoints and URLs from JavaScript bundles found on the target. After each scan, it:

1. Fetches the target's root HTML and parses all `<script src>` tags
2. Fetches each JS bundle (handles relative, absolute, and protocol-relative URLs)
3. Passively analyses any `.js` files discovered during the wordlist scan
4. Extracts: absolute URLs, WebSocket URLs, relative API paths (`/api/*`, `/v1/*`, `/graphql`, `/admin/*`, etc.), environment variables with URL values (`REACT_APP_*`), `fetch()` / `axios.*()` calls, and `baseURL` assignments

Results are written to a sidecar file alongside the main scan output:

```
webhoundresults/
  example.com-2026-06-24-150405.txt
  example.com-2026-06-24-150405-js-endpoints.txt
```

**Example sidecar output:**
```
WebHound — JS Endpoint Extraction
Target  : https://example.com
JS Files: 2 analysed
Findings: 14
────────────────────────────────────────────────────────────────────────

[js] https://example.com/static/js/main.abc123.js (12 findings)
  [URL]   https://api.example.com/v1
  [ENV]   REACT_APP_API_URL → https://api.example.com
  [PATH]  /api/v1/users
  [PATH]  /admin/dashboard
  [WS]    wss://ws.example.com/live
  ...
```

Disable with `--no-js-extract`.

---

## Auto-Save Output

By default, WebHound saves all findings to a plain-text file without requiring any flags:

```
webhoundresults/
  joms.jazeeraairways.com-2026-06-20-150405.txt
  api.example.com-2026-06-20-162311.txt
```

**File format:**
```
WebHound v1.0.1 — Scan Report
Started : 2026-06-20 15:04:05 UTC
Finished: 2026-06-20 15:06:42 UTC
Target  : https://example.com
Results : 12
────────────────────────────────────────────────────────────────────────

[200] https://example.com/login                     [11.0K] [131ms] [DIR]
[200] https://example.com/dashboard                 [28.6K] [294ms] [DIR]
[302] https://example.com/admin/login.aspx          [145B]  [1.2s]  [DIR] -> https://example.com/admin/StaffAcceptLogin.aspx
[401] https://example.com/api/v1                    [320B]  [88ms]

────────────────────────────────────────────────────────────────────────
Duration: 2m37s
```

You can override the path and format with `--output`:
```bash
webhound scan --url https://example.com -o results.json
webhound scan --url https://example.com -o results.html
```

---

## Config File

Copy `configs/default.yaml` and pass it with `--config`:

```bash
webhound scan --url https://example.com --config my-config.yaml
```

Key config options:

```yaml
threads: 20
rate: 30
timeout: 15s
user_agent: "Mozilla/5.0 ..."
follow_redirects: true
status_codes: [200, 201, 301, 302, 401, 403, 405, 500]
extensions: [php, asp, aspx, js, json, txt]
recursive: false
max_depth: 3
```

---

## Examples

```bash
# Pentest-style scan — fast, no extensions, key status codes only
webhound scan --url https://target.com \
  --wordlist common \
  --threads 20 \
  --rate 30 \
  --timeout 15s \
  --status-codes 200,201,204,301,302,307,401,403,405,500 \
  --no-color

# API enumeration with JWT
webhound scan --url https://api.target.com \
  --wordlist common \
  --bearer-token "eyJhbGci..." \
  --threads 20 --rate 30 --timeout 15s \
  --status-codes 200,201,204,401,403,405,500

# Scan with Burp Suite proxy
webhound scan --url https://target.com \
  --proxy http://127.0.0.1:8080 \
  --insecure

# Recursive directory enumeration
webhound scan --url https://target.com \
  --wordlist directories \
  --recursive \
  --max-depth 3 \
  --no-extension

# Resume an interrupted scan
webhound scan --url https://target.com \
  --resume scan-2026-06-20-150405

# Quiet mode — findings only, no progress output
webhound scan --url https://target.com -q

# Multiple targets
webhound scan --file targets.txt \
  --threads 30 --rate 50 \
  --output campaign.json
```

---

## Built-in Wordlists

| Name | Entries | Best for |
|------|---------|----------|
| `common` | ~200 | General-purpose — files, directories, APIs, sensitive paths |
| `directories` | ~130 | Directory enumeration only |
| `files` | ~60 | Sensitive file discovery (`.env`, backups, configs) |

All three are embedded in the binary — no download required.

---

## Output Formats

| Format | Flag | Description |
|--------|------|-------------|
| TXT | `--format txt` | Plain text, human-readable (default) |
| JSON | `--format json` | Structured JSON for pipeline integration |
| CSV | `--format csv` | Spreadsheet-compatible |
| HTML | `--format html` | Self-contained HTML report |

---

## Building from Source

```bash
make build          # Build for current platform
make build-all      # Cross-compile Linux/macOS/Windows
make deps           # Download dependencies
make test           # Run unit tests
make test-all       # Run all tests including integration
make install        # Install to GOPATH/bin
make clean          # Remove build artefacts
```

---

## Legal

This tool is intended for authorized security testing only. Use against systems you own or have explicit written permission to test. The author assumes no liability for misuse.

---

## License

MIT — see [LICENSE](LICENSE).
