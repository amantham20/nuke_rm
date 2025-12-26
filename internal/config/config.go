// Package config handles configuration management for nuke
package config

import (
	"os"
	"path/filepath"
	"strings"
)

// Config holds the application configuration
type Config struct {
	// ProtectedPaths lists paths that should never be deleted
	ProtectedPaths []string
	// TrashRetentionDays is how long to keep files in trash before auto-delete (default: 30)
	TrashRetentionDays int
	// TrashMaxSizeMB is the maximum size of trash directory in MB (default: 5000)
	TrashMaxSizeMB int
	// AutoCleanupEnabled enables automatic trash cleanup (default: true)
	AutoCleanupEnabled bool
}

// DefaultProtectedPaths returns the default list of protected paths
func DefaultProtectedPaths() []string {
	return []string{
		// Root and system directories
		"/",
		"/bin",
		"/sbin",
		"/usr",
		"/usr/bin",
		"/usr/sbin",
		"/usr/lib",
		"/usr/local",
		"/etc",
		"/var",
		"/lib",
		"/lib64",
		"/boot",
		"/sys",
		"/proc",
		"/dev",
		"/run",
		"/tmp", // Protect system tmp

		// macOS specific
		"/System",
		"/Library",
		"/Applications",
		"/private",
		"/cores",

		// User sensitive directories (will be expanded with home dir)
		"~/.ssh",
		"~/.gnupg",
		"~/.config",
		"~/.local/share",
		"~/Library", // macOS

		// Version control
		".git",

		// Common important directories
		"node_modules", // Optional but often important
	}
}

// LoadConfig loads the configuration from file or returns defaults
func LoadConfig() *Config {
	cfg := &Config{
		ProtectedPaths:     DefaultProtectedPaths(),
		TrashRetentionDays: 30,
		TrashMaxSizeMB:     5000,
		AutoCleanupEnabled: true,
	}

	// Expand home directory in paths
	homeDir, err := os.UserHomeDir()
	if err == nil {
		expandedPaths := make([]string, 0, len(cfg.ProtectedPaths))
		for _, p := range cfg.ProtectedPaths {
			if strings.HasPrefix(p, "~/") {
				p = filepath.Join(homeDir, p[2:])
			}
			expandedPaths = append(expandedPaths, p)
		}
		cfg.ProtectedPaths = expandedPaths
	}

	// Try to load user config file
	configPath := filepath.Join(homeDir, ".config", "nuke", "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		// Config file exists - could parse YAML here
		// For now, we just use defaults
		cfg.loadUserConfig(configPath)
	}

	return cfg
}

// loadUserConfig loads additional configuration from user config file
func (c *Config) loadUserConfig(path string) {
	// Read config file
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	// Simple parsing - look for protected_paths section
	lines := strings.Split(string(data), "\n")
	inProtectedPaths := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "protected_paths:" {
			inProtectedPaths = true
			continue
		}

		if inProtectedPaths {
			if strings.HasPrefix(line, "- ") {
				path := strings.TrimPrefix(line, "- ")
				path = strings.Trim(path, "\"'")
				if path != "" {
					// Expand home directory
					if strings.HasPrefix(path, "~/") {
						if homeDir, err := os.UserHomeDir(); err == nil {
							path = filepath.Join(homeDir, path[2:])
						}
					}
					c.ProtectedPaths = append(c.ProtectedPaths, path)
				}
			} else if !strings.HasPrefix(line, "#") && line != "" {
				// End of protected_paths section
				inProtectedPaths = false
			}
		}
	}
}

// IsProtected checks if a path is protected
func (c *Config) IsProtected(path string) bool {
	// Normalize the path
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	absPath = filepath.Clean(absPath)

	for _, protected := range c.ProtectedPaths {
		protected = filepath.Clean(protected)

		// Exact match
		if absPath == protected {
			return true
		}

		// Check if path is a child of protected path (for directories like .git)
		if !strings.HasPrefix(protected, "/") {
			// Relative protected path (like .git)
			if strings.Contains(absPath, "/"+protected+"/") || strings.HasSuffix(absPath, "/"+protected) {
				return true
			}
			// Check base name
			if filepath.Base(absPath) == protected {
				return true
			}
		}

		// Check if protected path is a parent of the target
		if strings.HasPrefix(absPath, protected+"/") {
			// Allow deletion within protected directories only if
			// the target is not a critical subdirectory
			criticalSubdirs := []string{"/bin", "/sbin", "/lib", "/etc"}
			for _, critical := range criticalSubdirs {
				if strings.HasPrefix(absPath, protected+critical) {
					return true
				}
			}
		}

		// Exact match for root-level protected paths
		if strings.HasPrefix(protected, "/") && absPath == protected {
			return true
		}
	}

	return false
}

// AddProtectedPath adds a new protected path
func (c *Config) AddProtectedPath(path string) {
	// Expand home directory
	if strings.HasPrefix(path, "~/") {
		if homeDir, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(homeDir, path[2:])
		}
	}
	c.ProtectedPaths = append(c.ProtectedPaths, path)
}
