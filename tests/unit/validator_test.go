package unit_test

import (
	"testing"

	"github.com/digitalohara/webhound/internal/input"
)

func TestValidateAndNormalize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "full https URL",
			input: "https://example.com",
			want:  "https://example.com",
		},
		{
			name:  "adds https scheme",
			input: "example.com",
			want:  "https://example.com",
		},
		{
			name:  "strips trailing slash",
			input: "https://example.com/",
			want:  "https://example.com",
		},
		{
			name:  "strips default http port",
			input: "http://example.com:80",
			want:  "http://example.com",
		},
		{
			name:  "strips default https port",
			input: "https://example.com:443",
			want:  "https://example.com",
		},
		{
			name:  "preserves non-default port",
			input: "https://example.com:8443",
			want:  "https://example.com:8443",
		},
		{
			name:  "preserves path",
			input: "https://example.com/api/v1",
			want:  "https://example.com/api/v1",
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid scheme",
			input:   "ftp://example.com",
			wantErr: true,
		},
		{
			name:    "no host",
			input:   "https://",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := input.ValidateAndNormalize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndNormalize(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ValidateAndNormalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateURLList_Deduplication(t *testing.T) {
	raw := []string{
		"https://example.com",
		"https://example.com/",
		"https://example.com:443",
		"https://other.com",
	}
	valid, errs := input.ValidateURLList(raw)
	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(valid) != 2 {
		t.Errorf("expected 2 unique URLs, got %d: %v", len(valid), valid)
	}
}

func TestValidateURLList_ErrorsDoNotAbort(t *testing.T) {
	raw := []string{
		"https://good.com",
		"ftp://bad.com",
		"https://also-good.com",
	}
	valid, errs := input.ValidateURLList(raw)
	if len(valid) != 2 {
		t.Errorf("expected 2 valid URLs, got %d", len(valid))
	}
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d: %v", len(errs), errs)
	}
}
