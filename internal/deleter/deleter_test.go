package deleter

import (
	"os"
	"path/filepath"
	"testing"

	"nuke/internal/scanner"
)

func TestDeleter(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "nuke-deleter-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create some test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file1, []byte("test1"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("test2"), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}

	files := []scanner.FileInfo{
		{Path: file1, IsDir: false, Size: 5},
		{Path: file2, IsDir: false, Size: 5},
	}

	// Test regular deletion
	d := New(2, false, nil)
	d.Delete(files, func(path string, err error) {
		if err != nil {
			t.Errorf("failed to delete %s: %v", path, err)
		}
	})

	// Verify files are gone
	if _, err := os.Stat(file1); !os.IsNotExist(err) {
		t.Errorf("expected file1 to be gone")
	}
	if _, err := os.Stat(file2); !os.IsNotExist(err) {
		t.Errorf("expected file2 to be gone")
	}

	// Test shredding
	file3 := filepath.Join(tmpDir, "file3.txt")
	if err := os.WriteFile(file3, []byte("shred me"), 0644); err != nil {
		t.Fatalf("failed to write file3: %v", err)
	}
	files = []scanner.FileInfo{
		{Path: file3, IsDir: false, Size: 8},
	}

	dShred := New(1, true, nil)
	dShred.Delete(files, func(path string, err error) {
		if err != nil {
			t.Errorf("failed to shred %s: %v", path, err)
		}
	})

	// Verify file is gone
	if _, err := os.Stat(file3); !os.IsNotExist(err) {
		t.Errorf("expected file3 to be gone after shredding")
	}
}
