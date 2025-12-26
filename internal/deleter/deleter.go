// Package deleter handles concurrent file deletion for nuke
package deleter

import (
	"crypto/rand"
	"os"
	"sort"
	"sync"

	"nuke/internal/scanner"
	"nuke/internal/trash"
)

// Deleter handles file deletion operations
type Deleter struct {
	workers  int            // Number of concurrent workers
	shred    bool           // Whether to securely shred files
	trashMgr *trash.Manager // Trash manager for soft delete
}

// New creates a new Deleter
func New(workers int, shred bool, trashMgr *trash.Manager) *Deleter {
	if workers <= 0 {
		workers = 8
	}
	return &Deleter{
		workers:  workers,
		shred:    shred,
		trashMgr: trashMgr,
	}
}

// ProgressCallback is called for each file processed
type ProgressCallback func(path string, err error)

// Delete deletes the given files concurrently
func (d *Deleter) Delete(files []scanner.FileInfo, onProgress ProgressCallback) {
	// Separate files and directories
	var regularFiles []scanner.FileInfo
	var directories []scanner.FileInfo

	for _, f := range files {
		if f.IsDir {
			directories = append(directories, f)
		} else {
			regularFiles = append(regularFiles, f)
		}
	}

	// Delete regular files concurrently
	d.deleteFilesConcurrently(regularFiles, onProgress)

	// Delete directories in order (deepest first)
	// Sort directories by depth (deepest first)
	sort.Slice(directories, func(i, j int) bool {
		return len(directories[i].Path) > len(directories[j].Path)
	})

	for _, dir := range directories {
		err := d.deleteDirectory(dir)
		if onProgress != nil {
			onProgress(dir.Path, err)
		}
	}
}

// deleteFilesConcurrently deletes files using multiple workers
func (d *Deleter) deleteFilesConcurrently(files []scanner.FileInfo, onProgress ProgressCallback) {
	if len(files) == 0 {
		return
	}

	// Create work channel
	workChan := make(chan scanner.FileInfo, len(files))

	// Create wait group
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < d.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range workChan {
				var err error
				if d.shred {
					err = d.shredFile(file)
				} else {
					err = d.softDelete(file)
				}
				if onProgress != nil {
					onProgress(file.Path, err)
				}
			}
		}()
	}

	// Send work to workers
	for _, file := range files {
		workChan <- file
	}
	close(workChan)

	// Wait for all workers to complete
	wg.Wait()
}

// softDelete moves a file to trash
func (d *Deleter) softDelete(file scanner.FileInfo) error {
	if d.trashMgr == nil {
		// Fall back to hard delete if no trash manager
		return os.Remove(file.Path)
	}
	return d.trashMgr.MoveToTrash(file.Path)
}

// shredFile securely overwrites and deletes a file
func (d *Deleter) shredFile(file scanner.FileInfo) error {
	// Open file for writing
	f, err := os.OpenFile(file.Path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}

	// Get file size
	size := file.Size

	// Perform multiple overwrite passes
	passes := 3                  // DoD standard is 3 passes
	buf := make([]byte, 64*1024) // 64KB buffer

	for pass := 0; pass < passes; pass++ {
		// Seek to beginning
		if _, err := f.Seek(0, 0); err != nil {
			f.Close()
			return err
		}

		remaining := size
		for remaining > 0 {
			toWrite := int64(len(buf))
			if toWrite > remaining {
				toWrite = remaining
			}

			// Fill buffer with random data (or zeros for alternating passes)
			if pass%2 == 0 {
				rand.Read(buf[:toWrite])
			} else {
				for i := range buf[:toWrite] {
					buf[i] = 0
				}
			}

			written, err := f.Write(buf[:toWrite])
			if err != nil {
				f.Close()
				return err
			}
			remaining -= int64(written)
		}

		// Sync to ensure data is written to disk
		if err := f.Sync(); err != nil {
			f.Close()
			return err
		}
	}

	f.Close()

	// Remove the file
	return os.Remove(file.Path)
}

// deleteDirectory removes a directory
func (d *Deleter) deleteDirectory(dir scanner.FileInfo) error {
	if d.trashMgr == nil || d.shred {
		// Hard delete for shred mode or if no trash manager
		return os.Remove(dir.Path)
	}
	return d.trashMgr.MoveToTrash(dir.Path)
}

// DeleteSingle deletes a single file
func (d *Deleter) DeleteSingle(file scanner.FileInfo) error {
	if file.IsDir {
		return d.deleteDirectory(file)
	}

	if d.shred {
		return d.shredFile(file)
	}
	return d.softDelete(file)
}
