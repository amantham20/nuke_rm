// Package trash provides soft delete (trash) functionality for nuke
package trash

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Manager handles trash operations
type Manager struct {
	trashDir string // Path to trash directory
	metaDir  string // Path to metadata directory
}

// TrashEntry represents metadata for a trashed file
type TrashEntry struct {
	OriginalPath string    `json:"original_path"`
	TrashPath    string    `json:"trash_path"`
	DeletedAt    time.Time `json:"deleted_at"`
	Size         int64     `json:"size"`
	IsDir        bool      `json:"is_dir"`
}

// NewManager creates a new trash manager
func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create trash directories
	trashDir := filepath.Join(homeDir, ".nuke-trash", "files")
	metaDir := filepath.Join(homeDir, ".nuke-trash", "meta")

	if err := os.MkdirAll(trashDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create trash directory: %w", err)
	}

	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create metadata directory: %w", err)
	}

	return &Manager{
		trashDir: trashDir,
		metaDir:  metaDir,
	}, nil
}

// MoveToTrash moves a file to the trash directory
func (m *Manager) MoveToTrash(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Get file info
	info, err := os.Lstat(absPath)
	if err != nil {
		return err
	}

	// Generate unique trash name
	timestamp := time.Now().UnixNano()
	baseName := filepath.Base(absPath)
	trashName := fmt.Sprintf("%d_%s", timestamp, baseName)
	trashPath := filepath.Join(m.trashDir, trashName)

	// Move file to trash
	if err := os.Rename(absPath, trashPath); err != nil {
		// If rename fails (e.g., cross-device), try copy and delete
		if err := copyPath(absPath, trashPath); err != nil {
			return fmt.Errorf("failed to move to trash: %w", err)
		}
		if err := os.RemoveAll(absPath); err != nil {
			// Try to clean up the copy
			os.RemoveAll(trashPath)
			return fmt.Errorf("failed to remove original: %w", err)
		}
	}

	// Save metadata
	entry := TrashEntry{
		OriginalPath: absPath,
		TrashPath:    trashPath,
		DeletedAt:    time.Now(),
		Size:         info.Size(),
		IsDir:        info.IsDir(),
	}

	metaPath := filepath.Join(m.metaDir, trashName+".json")
	metaData, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to create metadata: %w", err)
	}

	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	return nil
}

// Restore restores a file from trash
func (m *Manager) Restore(filename string) error {
	// Find the file in metadata
	entries, err := os.ReadDir(m.metaDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		metaPath := filepath.Join(m.metaDir, entry.Name())
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}

		var trashEntry TrashEntry
		if err := json.Unmarshal(data, &trashEntry); err != nil {
			continue
		}

		// Check if this is the file we're looking for
		if filepath.Base(trashEntry.OriginalPath) == filename ||
			strings.Contains(trashEntry.OriginalPath, filename) {

			// Check if trash file still exists
			if _, err := os.Stat(trashEntry.TrashPath); os.IsNotExist(err) {
				return fmt.Errorf("trash file no longer exists: %s", trashEntry.TrashPath)
			}

			// Check if original location is available
			if _, err := os.Stat(trashEntry.OriginalPath); err == nil {
				return fmt.Errorf("original location already exists: %s", trashEntry.OriginalPath)
			}

			// Create parent directory if needed
			parentDir := filepath.Dir(trashEntry.OriginalPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Restore the file
			if err := os.Rename(trashEntry.TrashPath, trashEntry.OriginalPath); err != nil {
				if err := copyPath(trashEntry.TrashPath, trashEntry.OriginalPath); err != nil {
					return fmt.Errorf("failed to restore file: %w", err)
				}
				os.RemoveAll(trashEntry.TrashPath)
			}

			// Remove metadata
			os.Remove(metaPath)

			return nil
		}
	}

	return fmt.Errorf("file not found in trash: %s", filename)
}

// List returns all files in trash
func (m *Manager) List() ([]TrashEntry, int64, error) {
	entries, err := os.ReadDir(m.metaDir)
	if err != nil {
		return nil, 0, err
	}

	var trashEntries []TrashEntry
	var totalSize int64

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		metaPath := filepath.Join(m.metaDir, entry.Name())
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}

		var trashEntry TrashEntry
		if err := json.Unmarshal(data, &trashEntry); err != nil {
			continue
		}

		// Verify the trash file still exists
		if info, err := os.Stat(trashEntry.TrashPath); err == nil {
			if info.IsDir() {
				// Calculate directory size
				dirSize := int64(0)
				filepath.Walk(trashEntry.TrashPath, func(_ string, info os.FileInfo, _ error) error {
					if info != nil && !info.IsDir() {
						dirSize += info.Size()
					}
					return nil
				})
				trashEntry.Size = dirSize
			}
			trashEntries = append(trashEntries, trashEntry)
			totalSize += trashEntry.Size
		}
	}

	return trashEntries, totalSize, nil
}

// Empty permanently deletes all files in trash
func (m *Manager) Empty() error {
	// Remove all files in trash directory
	if err := os.RemoveAll(m.trashDir); err != nil {
		return fmt.Errorf("failed to empty trash: %w", err)
	}

	// Remove all metadata
	if err := os.RemoveAll(m.metaDir); err != nil {
		return fmt.Errorf("failed to remove metadata: %w", err)
	}

	// Recreate directories
	if err := os.MkdirAll(m.trashDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(m.metaDir, 0755); err != nil {
		return err
	}

	return nil
}

// GetTrashDir returns the trash directory path
func (m *Manager) GetTrashDir() string {
	return m.trashDir
}

// AutoCleanup removes old files and enforces size limits
// Returns number of files cleaned and total size freed
func (m *Manager) AutoCleanup(retentionDays int, maxSizeMB int) (int, int64, error) {
	entries, totalSize, err := m.List()
	if err != nil {
		return 0, 0, err
	}

	if len(entries) == 0 {
		return 0, 0, nil
	}

	maxSizeBytes := int64(maxSizeMB) * 1024 * 1024
	now := time.Now()
	cutoffTime := now.AddDate(0, 0, -retentionDays)

	var itemsRemoved int
	var bytesFreed int64

	// First pass: remove files older than retention period
	for _, entry := range entries {
		if entry.DeletedAt.Before(cutoffTime) {
			if err := os.RemoveAll(entry.TrashPath); err == nil {
				bytesFreed += entry.Size
				itemsRemoved++
			}
			// Remove metadata
			metaPath := filepath.Join(m.metaDir, filepath.Base(entry.TrashPath)+".json")
			os.Remove(metaPath)
		}
	}

	// Check if we need to do size-based cleanup
	newTotalSize := totalSize - bytesFreed
	if newTotalSize > maxSizeBytes {
		// Need to remove more files - remove oldest files first
		remaining, _, _ := m.List()

		// Sort by deletion time (oldest first)
		for i := 0; i < len(remaining)-1; i++ {
			for j := 0; j < len(remaining)-i-1; j++ {
				if remaining[j].DeletedAt.After(remaining[j+1].DeletedAt) {
					remaining[j], remaining[j+1] = remaining[j+1], remaining[j]
				}
			}
		}

		// Remove oldest files until we're under the size limit
		for _, entry := range remaining {
			if newTotalSize <= maxSizeBytes {
				break
			}

			if err := os.RemoveAll(entry.TrashPath); err == nil {
				bytesFreed += entry.Size
				newTotalSize -= entry.Size
				itemsRemoved++
			}

			// Remove metadata
			metaPath := filepath.Join(m.metaDir, filepath.Base(entry.TrashPath)+".json")
			os.Remove(metaPath)
		}
	}

	return itemsRemoved, bytesFreed, nil
}

// copyPath copies a file or directory
func copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, srcInfo.Mode())
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
