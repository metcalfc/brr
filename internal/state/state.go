package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	stateFileName = "reading_positions.json"
	hashBytes     = 8192 // First 8KB for content hash
)

// ReadingState stores position for a single file
type ReadingState struct {
	WordIndex int `json:"word_index"`
}

// StateStore manages persistent reading state
type StateStore struct {
	path string
	data map[string]ReadingState
	mu   sync.RWMutex
}

// NewStateStore creates or loads state from XDG_STATE_HOME/brr/
func NewStateStore() (*StateStore, error) {
	dir := getStateDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	store := &StateStore{
		path: filepath.Join(dir, stateFileName),
		data: make(map[string]ReadingState),
	}
	if err := store.load(); err != nil {
		// Non-fatal - start with empty state
		store.data = make(map[string]ReadingState)
	}
	return store, nil
}

// getStateDir returns XDG_STATE_HOME/brr or ~/.local/state/brr
func getStateDir() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "brr")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "brr")
}

// ComputeHash generates content hash for file identity
func ComputeHash(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, hashBytes)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return "", err
	}

	hash := sha256.Sum256(buf[:n])
	return hex.EncodeToString(hash[:16]), nil // First 16 bytes = 32 hex chars
}

// GetPosition returns saved position for file, or 0 if not found
func (s *StateStore) GetPosition(hash string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if state, ok := s.data[hash]; ok {
		return state.WordIndex
	}
	return 0
}

// SetPosition saves position for file
func (s *StateStore) SetPosition(hash string, wordIndex int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[hash] = ReadingState{WordIndex: wordIndex}
	return s.save()
}

// Clear removes saved position for file
func (s *StateStore) Clear(hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, hash)
	return s.save()
}

func (s *StateStore) load() error {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.data)
}

func (s *StateStore) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}
