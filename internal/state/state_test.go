package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeHash(t *testing.T) {
	// Create temp file with known content
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "test1.txt")
	file2 := filepath.Join(tmpDir, "test2.txt")
	file3 := filepath.Join(tmpDir, "test1_copy.txt")

	os.WriteFile(file1, []byte("Hello, World!"), 0644)
	os.WriteFile(file2, []byte("Different content"), 0644)
	os.WriteFile(file3, []byte("Hello, World!"), 0644) // Same as file1

	hash1, err := ComputeHash(file1)
	if err != nil {
		t.Fatalf("ComputeHash failed: %v", err)
	}

	hash2, err := ComputeHash(file2)
	if err != nil {
		t.Fatalf("ComputeHash failed: %v", err)
	}

	hash3, err := ComputeHash(file3)
	if err != nil {
		t.Fatalf("ComputeHash failed: %v", err)
	}

	// Same content = same hash
	if hash1 != hash3 {
		t.Errorf("Same content should produce same hash: %s != %s", hash1, hash3)
	}

	// Different content = different hash
	if hash1 == hash2 {
		t.Errorf("Different content should produce different hash")
	}

	// Hash should be 32 hex chars
	if len(hash1) != 32 {
		t.Errorf("Hash should be 32 chars, got %d", len(hash1))
	}
}

func TestComputeHashSmallFile(t *testing.T) {
	tmpDir := t.TempDir()
	smallFile := filepath.Join(tmpDir, "small.txt")
	os.WriteFile(smallFile, []byte("tiny"), 0644)

	hash, err := ComputeHash(smallFile)
	if err != nil {
		t.Fatalf("ComputeHash failed on small file: %v", err)
	}

	if len(hash) != 32 {
		t.Errorf("Hash should be 32 chars even for small files, got %d", len(hash))
	}
}

func TestStateStore(t *testing.T) {
	// Use temp directory for state
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	store, err := NewStateStore()
	if err != nil {
		t.Fatalf("NewStateStore failed: %v", err)
	}

	testHash := "abcdef1234567890abcdef1234567890"

	// GetPosition returns 0 for unknown hash
	pos := store.GetPosition(testHash)
	if pos != 0 {
		t.Errorf("Expected 0 for unknown hash, got %d", pos)
	}

	// SetPosition/GetPosition roundtrip
	err = store.SetPosition(testHash, 1234)
	if err != nil {
		t.Fatalf("SetPosition failed: %v", err)
	}

	pos = store.GetPosition(testHash)
	if pos != 1234 {
		t.Errorf("Expected 1234, got %d", pos)
	}

	// Clear removes entry
	err = store.Clear(testHash)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	pos = store.GetPosition(testHash)
	if pos != 0 {
		t.Errorf("Expected 0 after clear, got %d", pos)
	}
}

func TestStateStorePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	testHash := "abcdef1234567890abcdef1234567890"

	// Create store and set position
	store1, err := NewStateStore()
	if err != nil {
		t.Fatalf("NewStateStore failed: %v", err)
	}
	store1.SetPosition(testHash, 5678)

	// Create new store instance - should load persisted data
	store2, err := NewStateStore()
	if err != nil {
		t.Fatalf("NewStateStore failed: %v", err)
	}

	pos := store2.GetPosition(testHash)
	if pos != 5678 {
		t.Errorf("Expected 5678 from persisted state, got %d", pos)
	}
}
