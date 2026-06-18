package recursion

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/digitalohara/webhound/internal/response"
)

// QueueManager collects discovered directories and presents them for review.
type QueueManager struct {
	items []*response.Result
}

// NewQueueManager creates a QueueManager.
func NewQueueManager() *QueueManager {
	return &QueueManager{}
}

// Enqueue adds a discovered directory to the queue.
func (q *QueueManager) Enqueue(r *response.Result) {
	q.items = append(q.items, r)
}

// HasItems returns true if there are directories pending review.
func (q *QueueManager) HasItems() bool {
	return len(q.items) > 0
}

// ReviewAndSelect presents the collected directories to the user and
// returns the subset selected for recursion.
func (q *QueueManager) ReviewAndSelect() ([]*response.Result, error) {
	if len(q.items) == 0 {
		return nil, nil
	}

	fmt.Printf("\n\033[1;34m[*]\033[0m Discovered %d recurseable director(ies):\n\n", len(q.items))
	for i, r := range q.items {
		fmt.Printf("  %2d. %s (%d)\n", i+1, r.URL, r.StatusCode)
	}

	fmt.Printf("\nSelect directories to recurse (e.g. 1,3 / all / none): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	input = strings.TrimSpace(input)

	return q.parseSelection(input)
}

func (q *QueueManager) parseSelection(input string) ([]*response.Result, error) {
	lower := strings.ToLower(input)
	switch lower {
	case "all", "a":
		return q.items, nil
	case "none", "n", "":
		return nil, nil
	}

	var selected []*response.Result
	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var idx int
		if _, err := fmt.Sscanf(part, "%d", &idx); err != nil {
			return nil, fmt.Errorf("invalid selection %q", part)
		}
		if idx < 1 || idx > len(q.items) {
			return nil, fmt.Errorf("selection %d out of range (1-%d)", idx, len(q.items))
		}
		selected = append(selected, q.items[idx-1])
	}
	return selected, nil
}
