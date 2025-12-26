// Package scanner handles file system scanning for nuke
package scanner

import (
	"os"
	"path/filepath"

	"nuke/internal/filter"
)

// FileInfo represents information about a file to be deleted
type FileInfo struct {
	Path    string      // Absolute path to the file
	Size    int64       // Size in bytes
	Mode    os.FileMode // File mode
	ModTime int64       // Modification time (Unix timestamp)
	IsDir   bool        // Whether this is a directory
}

// Scan scans a path and returns all matching files
func Scan(path string, recursive bool, filterOpts *filter.Options) ([]FileInfo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Lstat(absPath)
	if err != nil {
		return nil, err
	}

	var files []FileInfo

	// If it's a file, check filter and return
	if !info.IsDir() {
		if filterOpts == nil || filterOpts.Match(absPath, info) {
			files = append(files, FileInfo{
				Path:    absPath,
				Size:    info.Size(),
				Mode:    info.Mode(),
				ModTime: info.ModTime().Unix(),
				IsDir:   false,
			})
		}
		return files, nil
	}

	// It's a directory
	if !recursive {
		// Non-recursive: just add the directory itself
		if filterOpts == nil || filterOpts.Match(absPath, info) {
			files = append(files, FileInfo{
				Path:    absPath,
				Size:    info.Size(),
				Mode:    info.Mode(),
				ModTime: info.ModTime().Unix(),
				IsDir:   true,
			})
		}
		return files, nil
	}

	// Recursive scan
	err = filepath.Walk(absPath, func(filePath string, fileInfo os.FileInfo, walkErr error) error {
		if walkErr != nil {
			// Skip files we can't access
			return nil
		}

		// Apply filters
		if filterOpts != nil && !filterOpts.Match(filePath, fileInfo) {
			return nil
		}

		files = append(files, FileInfo{
			Path:    filePath,
			Size:    fileInfo.Size(),
			Mode:    fileInfo.Mode(),
			ModTime: fileInfo.ModTime().Unix(),
			IsDir:   fileInfo.IsDir(),
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort files so that deeper paths come first (for deletion order)
	// This ensures we delete files before their parent directories
	sortByDepth(files)

	return files, nil
}

// sortByDepth sorts files by path depth (deepest first)
func sortByDepth(files []FileInfo) {
	// Simple bubble sort for stability - could use sort.Slice for larger sets
	n := len(files)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			depth1 := countSeparators(files[j].Path)
			depth2 := countSeparators(files[j+1].Path)

			// Sort by depth descending, then by path ascending for same depth
			if depth1 < depth2 || (depth1 == depth2 && files[j].Path > files[j+1].Path) {
				files[j], files[j+1] = files[j+1], files[j]
			}
		}
	}
}

// countSeparators counts the number of path separators in a path
func countSeparators(path string) int {
	count := 0
	for _, c := range path {
		if c == filepath.Separator {
			count++
		}
	}
	return count
}

// ScanWithCallback scans a path and calls the callback for each file
// This is useful for progress reporting during scanning
func ScanWithCallback(path string, recursive bool, filterOpts *filter.Options, callback func(FileInfo)) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	info, err := os.Lstat(absPath)
	if err != nil {
		return err
	}

	// If it's a file, check filter and call callback
	if !info.IsDir() {
		if filterOpts == nil || filterOpts.Match(absPath, info) {
			callback(FileInfo{
				Path:    absPath,
				Size:    info.Size(),
				Mode:    info.Mode(),
				ModTime: info.ModTime().Unix(),
				IsDir:   false,
			})
		}
		return nil
	}

	// Directory handling
	if !recursive {
		if filterOpts == nil || filterOpts.Match(absPath, info) {
			callback(FileInfo{
				Path:    absPath,
				Size:    info.Size(),
				Mode:    info.Mode(),
				ModTime: info.ModTime().Unix(),
				IsDir:   true,
			})
		}
		return nil
	}

	// Recursive scan
	return filepath.Walk(absPath, func(filePath string, fileInfo os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		if filterOpts != nil && !filterOpts.Match(filePath, fileInfo) {
			return nil
		}

		callback(FileInfo{
			Path:    filePath,
			Size:    fileInfo.Size(),
			Mode:    fileInfo.Mode(),
			ModTime: fileInfo.ModTime().Unix(),
			IsDir:   fileInfo.IsDir(),
		})

		return nil
	})
}
