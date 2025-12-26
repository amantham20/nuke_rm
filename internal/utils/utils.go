// Package utils provides utility functions for nuke
package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// FormatSize formats a size in bytes to a human-readable string
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// ParseSize parses a size string (e.g., "100M", "1G") to bytes
func ParseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// Extract number and unit
	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*([KMGT]?B?)?$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid size format: %s", s)
	}

	numStr := matches[1]
	unit := matches[2]

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", numStr)
	}

	multiplier := int64(1) // Default for bytes
	switch unit {
	case "K", "KB":
		multiplier = 1024
	case "M", "MB":
		multiplier = 1024 * 1024
	case "G", "GB":
		multiplier = 1024 * 1024 * 1024
	case "T", "TB":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "B", "":
		// Use default multiplier of 1
	default:
		return 0, fmt.Errorf("unknown unit: %s", unit)
	}

	return int64(num * float64(multiplier)), nil
}

// ParseSizeFilter parses a size filter string (e.g., "+100M", "-1G")
// Returns the size in bytes and the operator ("+" or "-")
func ParseSizeFilter(s string) (int64, string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, "", fmt.Errorf("empty size filter")
	}

	var op string
	switch s[0] {
	case '+':
		op = "+"
		s = s[1:]
	case '-':
		op = "-"
		s = s[1:]
	default:
		op = "+" // Default to greater than
	}

	size, err := ParseSize(s)
	if err != nil {
		return 0, "", err
	}

	return size, op, nil
}

// ParseDuration parses a duration string (e.g., "30d", "24h", "1w")
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	// Check for custom units (days, weeks, months, years)
	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*([a-z]+)$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		// Try standard Go duration parsing
		return time.ParseDuration(s)
	}

	numStr := matches[1]
	unit := matches[2]

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", numStr)
	}

	var multiplier time.Duration
	switch unit {
	case "s", "sec", "second", "seconds":
		multiplier = time.Second
	case "m", "min", "minute", "minutes":
		multiplier = time.Minute
	case "h", "hr", "hour", "hours":
		multiplier = time.Hour
	case "d", "day", "days":
		multiplier = 24 * time.Hour
	case "w", "week", "weeks":
		multiplier = 7 * 24 * time.Hour
	case "mo", "month", "months":
		multiplier = 30 * 24 * time.Hour // Approximate
	case "y", "year", "years":
		multiplier = 365 * 24 * time.Hour // Approximate
	default:
		return 0, fmt.Errorf("unknown time unit: %s", unit)
	}

	return time.Duration(num * float64(multiplier)), nil //nolint:gosec // Duration calculation is safe
}

// TruncatePath truncates a path for display
func TruncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}

	// Try to keep the filename visible
	parts := strings.Split(path, "/")
	if len(parts) <= 2 {
		return path[:maxLen-3] + "..."
	}

	filename := parts[len(parts)-1]
	if len(filename) > maxLen-6 {
		return "..." + path[len(path)-maxLen+3:]
	}

	prefix := parts[0]
	if parts[0] == "" && len(parts) > 1 {
		prefix = "/" + parts[1]
	}

	remaining := maxLen - len(prefix) - len(filename) - 4 // 4 for "/..."
	if remaining < 0 {
		return "..." + path[len(path)-maxLen+3:]
	}

	return prefix + "/..." + "/" + filename
}

// ConfirmationMessage generates a confirmation message based on file count
func ConfirmationMessage(fileCount int, totalSize int64) string {
	sizeStr := FormatSize(totalSize)

	if fileCount == 1 {
		return fmt.Sprintf("You are about to delete 1 file (%s).", sizeStr)
	}
	return fmt.Sprintf("You are about to delete %d files totaling %s.", fileCount, sizeStr)
}

// IsGlobPattern checks if a string contains glob metacharacters
func IsGlobPattern(s string) bool {
	for _, c := range s {
		switch c {
		case '*', '?', '[', ']':
			return true
		}
	}
	return false
}

// SanitizePath removes dangerous characters from a path
func SanitizePath(path string) string {
	// Remove null bytes and other control characters
	var result strings.Builder
	for _, r := range path {
		if r >= 32 && r != 127 {
			result.WriteRune(r)
		}
	}
	return result.String()
}
