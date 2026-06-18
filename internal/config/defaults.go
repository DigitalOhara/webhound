package config

import "time"

const (
	DefaultUserAgent       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36"
	DefaultThreads         = 10
	DefaultRate            = 50.0
	DefaultTimeout         = 10 * time.Second
	DefaultMaxRetries      = 3
	DefaultMaxDepth        = 3
	DefaultMaxRedirects    = 5
	DefaultCheckpointEvery = 500
	MaxThreads             = 100
)

var DefaultExtensions = []string{
	"php", "asp", "aspx", "jsp", "js", "json", "txt", "xml",
	"bak", "old", "zip", "tar", "gz", "html", "htm", "cfg",
}

var DefaultStatusCodes = []int{
	200, 201, 202, 204,
	301, 302, 307, 308,
	401, 403, 405,
}

// DefaultConfig returns a ScanConfig pre-filled with safe defaults.
func DefaultConfig() *ScanConfig {
	return &ScanConfig{
		Threads:            DefaultThreads,
		Rate:               DefaultRate,
		Timeout:            DefaultTimeout,
		MaxRetries:         DefaultMaxRetries,
		MaxDepth:           DefaultMaxDepth,
		UserAgent:          DefaultUserAgent,
		FollowRedirects:    true,
		MaxRedirects:       DefaultMaxRedirects,
		Method:             "GET",
		Extensions:         DefaultExtensions,
		StatusCodes:        DefaultStatusCodes,
		CheckpointInterval: DefaultCheckpointEvery,
		SmartRecursion:     true,
	}
}
