package trash

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrashOperations(t *testing.T) {
	// Create a temporary directory for the trash
	tmpDir, err := os.MkdirTemp("", "nuke-trash-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	mgr, err := NewManagerAt(tmpDir)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test MoveToTrash
	if err := mgr.MoveToTrash(testFile); err != nil {
		t.Fatalf("failed to move file to trash: %v", err)
	}

	// Verify file is gone from original location
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Errorf("expected file to be gone from original location")
	}

	// Test List
	entries, totalSize, err := mgr.List()
	if err != nil {
		t.Fatalf("failed to list trash: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry in trash, got %d", len(entries))
	}
	if totalSize != 11 {
		t.Errorf("expected total size 11, got %d", totalSize)
	}

	// Test Restore
	if err := mgr.Restore("test.txt"); err != nil {
		t.Fatalf("failed to restore file: %v", err)
	}

	// Verify file is back
	if _, err := os.Stat(testFile); err != nil {
		t.Errorf("expected file to be restored to original location")
	}

	// Test Empty
	if err := mgr.MoveToTrash(testFile); err != nil {
		t.Fatalf("failed to move file to trash again: %v", err)
	}
	if err := mgr.Empty(); err != nil {
		t.Fatalf("failed to empty trash: %v", err)
	}
	entries, _, _ = mgr.List()
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after empty, got %d", len(entries))
	}
}
