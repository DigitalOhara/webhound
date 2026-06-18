package reporting

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/digitalohara/webhound/internal/response"
)

// WriteCSV writes results to a CSV file.
func WriteCSV(path string, results []*response.Result) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating CSV file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"url", "path", "status_code", "content_length",
		"content_type", "words", "lines", "response_time_ms",
		"redirect_url", "is_directory", "depth", "found_at",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, r := range results {
		record := []string{
			r.URL,
			r.Path,
			strconv.Itoa(r.StatusCode),
			strconv.FormatInt(r.ContentLength, 10),
			r.ContentType,
			strconv.Itoa(r.Words),
			strconv.Itoa(r.Lines),
			strconv.FormatInt(r.ResponseTime.Milliseconds(), 10),
			r.RedirectURL,
			strconv.FormatBool(r.IsDirectory),
			strconv.Itoa(r.Depth),
			r.FoundAt.Format(time.RFC3339),
		}
		if err := w.Write(record); err != nil {
			return err
		}
	}
	return nil
}
