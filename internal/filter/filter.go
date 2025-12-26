// Package filter provides file filtering capabilities for nuke
package filter

import (
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// Options represents filtering options for file selection
type Options struct {
	// Time-based filters
	OlderThan *time.Time // Only files older than this time
	NewerThan *time.Time // Only files newer than this time

	// Size-based filters
	SizeFilter int64  // Size threshold in bytes
	SizeOp     string // Operator: "+" for greater than, "-" for less than

	// Pattern-based filters
	Exclude []string       // Glob patterns to exclude
	Include []string       // Glob patterns to include (if set, only these match)
	Regex   *regexp.Regexp // Regex pattern to match

	// Skip hidden files
	SkipHidden bool
}

// Match checks if a file matches the filter criteria
func (o *Options) Match(path string, info os.FileInfo) bool {
	if o == nil {
		return true
	}

	// Check hidden files
	if o.SkipHidden && isHidden(path) {
		return false
	}

	// Check time-based filters
	if o.OlderThan != nil {
		if info.ModTime().After(*o.OlderThan) {
			return false
		}
	}

	if o.NewerThan != nil {
		if info.ModTime().Before(*o.NewerThan) {
			return false
		}
	}

	// Check size-based filters (only for regular files)
	if o.SizeFilter > 0 && !info.IsDir() {
		switch o.SizeOp {
		case "+":
			if info.Size() <= o.SizeFilter {
				return false
			}
		case "-":
			if info.Size() >= o.SizeFilter {
				return false
			}
		}
	}

	// Check include patterns (if set, file must match at least one)
	if len(o.Include) > 0 {
		matched := false
		baseName := filepath.Base(path)
		for _, pattern := range o.Include {
			if match, _ := filepath.Match(pattern, baseName); match {
				matched = true
				break
			}
			// Also try matching against full path
			if match, _ := filepath.Match(pattern, path); match {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check exclude patterns
	if len(o.Exclude) > 0 {
		baseName := filepath.Base(path)
		for _, pattern := range o.Exclude {
			if match, _ := filepath.Match(pattern, baseName); match {
				return false
			}
			// Also try matching against full path
			if match, _ := filepath.Match(pattern, path); match {
				return false
			}
		}
	}

	// Check regex pattern
	if o.Regex != nil {
		if !o.Regex.MatchString(path) && !o.Regex.MatchString(filepath.Base(path)) {
			return false
		}
	}

	return true
}

// isHidden checks if a file is hidden (starts with .)
func isHidden(path string) bool {
	baseName := filepath.Base(path)
	return len(baseName) > 0 && baseName[0] == '.'
}

// MatchesGlob checks if a path matches any of the given glob patterns
func MatchesGlob(path string, patterns []string) bool {
	baseName := filepath.Base(path)
	for _, pattern := range patterns {
		if match, _ := filepath.Match(pattern, baseName); match {
			return true
		}
		if match, _ := filepath.Match(pattern, path); match {
			return true
		}
	}
	return false
}
