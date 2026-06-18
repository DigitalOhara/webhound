package recursion

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/digitalohara/webhound/internal/response"
)

// InteractiveMode tracks the user's recursion preference during a scan.
type InteractiveMode struct {
	autoAll   bool // user selected "All"
	disabled  bool // user selected "Quit"
	reader    *bufio.Reader
}

// NewInteractiveMode creates an InteractiveMode.
func NewInteractiveMode() *InteractiveMode {
	return &InteractiveMode{reader: bufio.NewReader(os.Stdin)}
}

// Prompt asks the user whether to recurse into a discovered directory.
// Returns true if recursion should proceed.
func (m *InteractiveMode) Prompt(r *response.Result, estimatedRequests int, estimatedDuration string) bool {
	if m.disabled {
		return false
	}
	if m.autoAll {
		return true
	}

	fmt.Printf("\n\033[1;33m[FOUND]\033[0m %s (%d)\n", r.URL, r.StatusCode)
	if estimatedRequests > 0 {
		fmt.Printf("  Estimated requests : %d\n", estimatedRequests)
		fmt.Printf("  Estimated duration : %s\n", estimatedDuration)
	}
	fmt.Printf("\n  Recurse into this directory?\n")
	fmt.Printf("    (Y) Yes\n    (N) No\n    (A) All future directories\n    (Q) Disable prompts\n")

	for {
		fmt.Printf("  Selection: ")
		input, err := m.reader.ReadString('\n')
		if err != nil {
			return false
		}
		switch strings.ToUpper(strings.TrimSpace(input)) {
		case "Y":
			return true
		case "N":
			return false
		case "A":
			m.autoAll = true
			return true
		case "Q":
			m.disabled = true
			return false
		default:
			fmt.Printf("  Invalid input. Please enter Y, N, A, or Q.\n")
		}
	}
}
