package reporting

import (
	"fmt"
	"html/template"
	"os"
	"time"

	"github.com/digitalohara/webhound/internal/config"
	"github.com/digitalohara/webhound/internal/response"
)

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>WebHound Scan Report</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: 'Segoe UI', system-ui, sans-serif; background: #0d1117; color: #c9d1d9; }
  header { background: #161b22; border-bottom: 1px solid #30363d; padding: 20px 40px; }
  header h1 { color: #58a6ff; font-size: 1.8rem; }
  header p { color: #8b949e; margin-top: 4px; font-size: 0.9rem; }
  .container { max-width: 1400px; margin: 0 auto; padding: 24px 40px; }
  .stats { display: flex; gap: 16px; flex-wrap: wrap; margin-bottom: 24px; }
  .stat-card { background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 16px 24px; flex: 1; min-width: 160px; }
  .stat-card .value { font-size: 2rem; font-weight: bold; color: #58a6ff; }
  .stat-card .label { color: #8b949e; font-size: 0.85rem; margin-top: 4px; }
  .config-section { background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 16px; margin-bottom: 24px; }
  .config-section h2 { color: #f0f6fc; font-size: 1rem; margin-bottom: 12px; }
  .config-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 8px; }
  .config-item { display: flex; gap: 8px; font-size: 0.85rem; }
  .config-key { color: #8b949e; min-width: 100px; }
  .config-val { color: #c9d1d9; }
  table { width: 100%; border-collapse: collapse; background: #161b22; border: 1px solid #30363d; border-radius: 8px; overflow: hidden; }
  th { background: #21262d; color: #8b949e; font-size: 0.8rem; text-transform: uppercase; letter-spacing: 0.05em; padding: 10px 16px; text-align: left; position: sticky; top: 0; }
  td { padding: 8px 16px; border-top: 1px solid #21262d; font-size: 0.85rem; font-family: 'SF Mono', 'Cascadia Code', monospace; white-space: nowrap; }
  tr:hover td { background: #21262d; }
  .s2xx { color: #3fb950; font-weight: bold; }
  .s3xx { color: #79c0ff; }
  .s4xx { color: #d29922; }
  .s5xx { color: #f85149; }
  .dir-badge { background: #3d2b8e; color: #a5a0ff; border-radius: 4px; padding: 1px 6px; font-size: 0.75rem; margin-left: 6px; }
  .url-cell { max-width: 600px; overflow: hidden; text-overflow: ellipsis; }
  .footer { color: #8b949e; font-size: 0.8rem; text-align: center; padding: 24px; }
</style>
</head>
<body>
<header>
  <h1>🔍 WebHound Scan Report</h1>
  <p>Generated {{ .FinishedAt.Format "2006-01-02 15:04:05 UTC" }} · Duration {{ .Duration }}</p>
</header>
<div class="container">
  <div class="stats">
    <div class="stat-card"><div class="value">{{ .Stats.Found }}</div><div class="label">Total Findings</div></div>
    <div class="stat-card"><div class="value">{{ .Stats.Directories }}</div><div class="label">Directories</div></div>
    <div class="stat-card"><div class="value">{{ .Stats.Files }}</div><div class="label">Files</div></div>
    <div class="stat-card"><div class="value">{{ .Stats.TotalRequests }}</div><div class="label">Requests</div></div>
    <div class="stat-card"><div class="value">{{ len .Targets }}</div><div class="label">Targets</div></div>
  </div>

  <div class="config-section">
    <h2>Scan Configuration</h2>
    <div class="config-grid">
      <div class="config-item"><span class="config-key">Threads</span><span class="config-val">{{ .Cfg.Threads }}</span></div>
      <div class="config-item"><span class="config-key">Rate</span><span class="config-val">{{ .Cfg.Rate }} req/s</span></div>
      <div class="config-item"><span class="config-key">Max Depth</span><span class="config-val">{{ .Cfg.MaxDepth }}</span></div>
      <div class="config-item"><span class="config-key">Auth Mode</span><span class="config-val">{{ .AuthMode }}</span></div>
      <div class="config-item"><span class="config-key">Started</span><span class="config-val">{{ .StartedAt.Format "15:04:05" }}</span></div>
    </div>
  </div>

  <table>
    <thead>
      <tr>
        <th>Status</th>
        <th>URL</th>
        <th>Length</th>
        <th>Words</th>
        <th>Lines</th>
        <th>Type</th>
        <th>Time</th>
        <th>Depth</th>
      </tr>
    </thead>
    <tbody>
    {{ range .Results }}
      <tr>
        <td class="{{ statusClass .StatusCode }}">{{ .StatusCode }}</td>
        <td class="url-cell">
          <a href="{{ .URL }}" style="color:inherit;text-decoration:none;" target="_blank">{{ .URL }}</a>
          {{ if .IsDirectory }}<span class="dir-badge">DIR</span>{{ end }}
        </td>
        <td>{{ formatSize .ContentLength }}</td>
        <td>{{ .Words }}</td>
        <td>{{ .Lines }}</td>
        <td>{{ .ContentType }}</td>
        <td>{{ formatDuration .ResponseTime }}</td>
        <td>{{ .Depth }}</td>
      </tr>
    {{ end }}
    </tbody>
  </table>
</div>
<div class="footer">WebHound v1.0.0 · Authorized Security Assessment Only</div>
</body>
</html>`

type htmlData struct {
	FinishedAt time.Time
	StartedAt  time.Time
	Duration   string
	Targets    []string
	Results    []*response.Result
	Stats      reportStats
	Cfg        *config.ScanConfig
	AuthMode   string
}

// WriteHTML writes a self-contained HTML report to path.
func WriteHTML(path string, cfg *config.ScanConfig, results []*response.Result, targets []string, startedAt time.Time) error {
	funcMap := template.FuncMap{
		"statusClass": func(code int) string {
			switch {
			case code >= 500:
				return "s5xx"
			case code >= 400:
				return "s4xx"
			case code >= 300:
				return "s3xx"
			default:
				return "s2xx"
			}
		},
		"formatSize": func(n int64) string { return formatSize(n) },
		"formatDuration": func(d time.Duration) string {
			return fmt.Sprintf("%dms", d.Milliseconds())
		},
	}

	tmpl, err := template.New("report").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("parsing HTML template: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating HTML file: %w", err)
	}
	defer f.Close()

	data := htmlData{
		FinishedAt: time.Now(),
		StartedAt:  startedAt,
		Duration:   time.Since(startedAt).Round(time.Millisecond).String(),
		Targets:    targets,
		Results:    results,
		Stats:      computeStats(results),
		Cfg:        cfg,
		AuthMode:   detectAuthMode(cfg),
	}

	return tmpl.Execute(f, data)
}
