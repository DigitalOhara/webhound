package response

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	probeCount       = 3
	lengthTolerancePct = 15
)

// Baseline holds the wildcard fingerprint for a single target host.
type Baseline struct {
	IsWildcard    bool
	StatusCode    int
	LengthMin     int64
	LengthMax     int64
	BodyHashes    [probeCount]string
	StructHashes  [probeCount]string
}

// WildcardDetector performs baseline probing and false-positive suppression.
type WildcardDetector struct {
	mu        sync.RWMutex
	baselines map[string]*Baseline // key: base URL
	client    *http.Client
	userAgent string
}

// NewWildcardDetector creates a WildcardDetector.
func NewWildcardDetector(client *http.Client, userAgent string) *WildcardDetector {
	return &WildcardDetector{
		baselines: make(map[string]*Baseline),
		client:    client,
		userAgent: userAgent,
	}
}

// Probe sends random-path requests to baseURL and builds a Baseline.
// Safe to call multiple times; subsequent calls for the same URL are no-ops.
func (d *WildcardDetector) Probe(ctx context.Context, baseURL string) (*Baseline, error) {
	d.mu.RLock()
	if b, ok := d.baselines[baseURL]; ok {
		d.mu.RUnlock()
		return b, nil
	}
	d.mu.RUnlock()

	d.mu.Lock()
	defer d.mu.Unlock()

	// Double-check after acquiring write lock.
	if b, ok := d.baselines[baseURL]; ok {
		return b, nil
	}

	bl := &Baseline{}
	statusCodes := make([]int, probeCount)
	lengths := make([]int64, probeCount)

	for i := 0; i < probeCount; i++ {
		randomPath := uuid.New().String()
		probeURL := strings.TrimRight(baseURL, "/") + "/" + randomPath

		reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, probeURL, nil)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("creating probe request: %w", err)
		}
		req.Header.Set("User-Agent", d.userAgent)

		resp, err := d.client.Do(req)
		cancel()
		if err != nil {
			// If we can't probe, assume no wildcard.
			d.baselines[baseURL] = bl
			return bl, nil
		}

		body := readBodyLimited(resp)
		statusCodes[i] = resp.StatusCode
		lengths[i] = resp.ContentLength
		if lengths[i] <= 0 {
			lengths[i] = int64(len(body))
		}
		bl.BodyHashes[i] = hashBody(body)
		bl.StructHashes[i] = hashStructure(body)
		resp.Body.Close()
	}

	// Wildcard if all probes return the same status code.
	if statusCodes[0] == statusCodes[1] && statusCodes[1] == statusCodes[2] {
		bl.IsWildcard = true
		bl.StatusCode = statusCodes[0]
		bl.LengthMin = minInt64(lengths)
		bl.LengthMax = maxInt64(lengths)
	}

	d.baselines[baseURL] = bl
	return bl, nil
}

// IsFalsePositive returns true if the result looks like a wildcard response.
func (d *WildcardDetector) IsFalsePositive(r *Result, body []byte) bool {
	d.mu.RLock()
	bl, ok := d.baselines[r.Target]
	d.mu.RUnlock()
	if !ok || !bl.IsWildcard {
		return false
	}
	if r.StatusCode != bl.StatusCode {
		return false
	}

	// Check content length within tolerance.
	if bl.LengthMax > 0 {
		tol := bl.LengthMax * lengthTolerancePct / 100
		if r.ContentLength >= bl.LengthMin-tol && r.ContentLength <= bl.LengthMax+tol {
			bh := hashBody(body)
			sh := hashStructure(body)
			for i := 0; i < probeCount; i++ {
				if bh == bl.BodyHashes[i] || sh == bl.StructHashes[i] {
					return true
				}
			}
			// Length in range but hash differs — likely a real page.
			// Still suppress if lengths are very tight (0 variance).
			if bl.LengthMin == bl.LengthMax && r.ContentLength == bl.LengthMin {
				return true
			}
		}
	}

	return false
}

// hashBody returns SHA-256 hex of the full body.
func hashBody(body []byte) string {
	h := sha256.Sum256(body)
	return fmt.Sprintf("%x", h)
}

// hashStructure returns a structural hash of the HTML (tag names only).
func hashStructure(body []byte) string {
	s := string(body)
	var tags strings.Builder
	inTag := false
	for _, ch := range s {
		switch {
		case ch == '<':
			inTag = true
			tags.WriteRune('<')
		case ch == '>':
			inTag = false
			tags.WriteRune('>')
		case inTag && (ch == ' ' || ch == '\n' || ch == '\t'):
			// Stop at first whitespace inside tag to get just the tag name.
			if tags.Len() > 0 {
				tags.WriteRune('>')
				inTag = false
			}
		case inTag:
			tags.WriteRune(ch)
		}
	}
	h := sha256.Sum256([]byte(tags.String()))
	return fmt.Sprintf("%x", h)
}

func readBodyLimited(resp *http.Response) []byte {
	buf := make([]byte, 1<<20) // 1 MB cap
	n, _ := resp.Body.Read(buf)
	return buf[:n]
}

func minInt64(s []int64) int64 {
	m := s[0]
	for _, v := range s[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func maxInt64(s []int64) int64 {
	m := s[0]
	for _, v := range s[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
