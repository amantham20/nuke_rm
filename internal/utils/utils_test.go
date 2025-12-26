package utils

import (
	"testing"
	"time"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"100", 100, false},
		{"1K", 1024, false},
		{"1KB", 1024, false},
		{"1M", 1024 * 1024, false},
		{"1MB", 1024 * 1024, false},
		{"1G", 1024 * 1024 * 1024, false},
		{"1GB", 1024 * 1024 * 1024, false},
		{"1T", 1024 * 1024 * 1024 * 1024, false},
		{"1TB", 1024 * 1024 * 1024 * 1024, false},
		{"1.5K", 1536, false},
		{"invalid", 0, true},
		{"1X", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseSize(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseSize(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.expected {
			t.Errorf("ParseSize(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"1s", time.Second, false},
		{"1m", time.Minute, false},
		{"1h", time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"1w", 7 * 24 * time.Hour, false},
		{"1mo", 30 * 24 * time.Hour, false},
		{"1y", 365 * 24 * time.Hour, false},
		{"2.5h", 2*time.Hour + 30*time.Minute, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseDuration(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.expected {
			t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
