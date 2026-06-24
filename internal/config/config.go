package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ScanConfig holds every tuneable parameter for a scan run.
type ScanConfig struct {
	// --- Target ---
	URL  string `yaml:"url"`
	File string `yaml:"file"`

	// --- Auth ---
	BearerToken string   `yaml:"bearer_token"`
	Headers     []string `yaml:"headers"`
	Cookies     []string `yaml:"cookies"`
	CookieFile  string   `yaml:"cookie_file"`
	BasicAuth   string   `yaml:"basic_auth"`
	Profile     string   `yaml:"profile"`

	// --- Wordlists ---
	Wordlists   []string `yaml:"wordlists"`
	Extensions  []string `yaml:"extensions"`
	NoExtension bool     `yaml:"no_extension"`

	// --- HTTP ---
	Threads         int           `yaml:"threads"`
	Rate            float64       `yaml:"rate"`
	Delay           time.Duration `yaml:"delay"`
	Timeout         time.Duration `yaml:"timeout"`
	MaxRetries      int           `yaml:"max_retries"`
	UserAgent       string        `yaml:"user_agent"`
	FollowRedirects bool          `yaml:"follow_redirects"`
	MaxRedirects    int           `yaml:"max_redirects"`
	Method          string        `yaml:"method"`

	// --- Proxy ---
	Proxy string `yaml:"proxy"`

	// --- TLS ---
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
	CACert             string `yaml:"ca_cert"`
	TLSMinVersion      string `yaml:"tls_min_version"`

	// --- Filtering ---
	StatusCodes          []int `yaml:"status_codes"`
	StatusCodesBlacklist []int `yaml:"status_codes_blacklist"`
	MinLength            int   `yaml:"min_length"`
	MaxLength            int   `yaml:"max_length"`
	HideLength           []int `yaml:"hide_length"`
	HideWords            []int `yaml:"hide_words"`
	HideLines            []int `yaml:"hide_lines"`

	// --- Recursion ---
	Recursive            bool     `yaml:"recursive"`
	InteractiveRecursion bool     `yaml:"interactive_recursion"`
	QueueRecursion       bool     `yaml:"queue_recursion"`
	MaxDepth             int      `yaml:"max_depth"`
	ExcludeRecursion     []string `yaml:"exclude_recursion"`
	SmartRecursion       bool     `yaml:"smart_recursion"`

	// --- Output ---
	Output  string `yaml:"output"`
	Format  string `yaml:"format"`
	Quiet   bool   `yaml:"quiet"`
	Verbose bool   `yaml:"verbose"`
	NoColor bool   `yaml:"no_color"`

	// --- Session ---
	Session            string `yaml:"session"`
	Resume             string `yaml:"resume"`
	CheckpointInterval int    `yaml:"checkpoint_interval"`

	// --- Safety ---
	NoConfirm   bool `yaml:"no_confirm"`
	MaxRequests int  `yaml:"max_requests"`

	// --- JS Extraction ---
	NoJSExtract bool `yaml:"no_js_extract"`
}

// LoadFile reads a YAML config file and returns a ScanConfig.
func LoadFile(path string) (*ScanConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	return cfg, nil
}

// Validate checks for required fields and logical consistency.
func (c *ScanConfig) Validate() error {
	if c.URL == "" && c.File == "" {
		return fmt.Errorf("either --url or --file is required")
	}
	if c.URL != "" && c.File != "" {
		return fmt.Errorf("--url and --file are mutually exclusive")
	}
	if c.Threads < 1 {
		return fmt.Errorf("threads must be >= 1")
	}
	if c.Threads > MaxThreads {
		return fmt.Errorf("threads cannot exceed %d", MaxThreads)
	}
	if c.Rate < 0 {
		return fmt.Errorf("rate cannot be negative")
	}
	if c.MaxDepth < 0 {
		return fmt.Errorf("max-depth cannot be negative")
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("max-retries cannot be negative")
	}
	recursionModes := 0
	if c.Recursive {
		recursionModes++
	}
	if c.InteractiveRecursion {
		recursionModes++
	}
	if c.QueueRecursion {
		recursionModes++
	}
	if recursionModes > 1 {
		return fmt.Errorf("only one recursion mode may be active at a time")
	}
	return nil
}
