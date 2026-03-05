package todo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/todogpt/daily-briefing/internal/models"
)

// TestSaveToReadOnlyFile triggers the os.WriteFile error path in save().
func TestSaveToReadOnlyFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses file permission checks")
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "todos.json")

	// Create the file and make it read-only
	if err := os.WriteFile(path, []byte("[]"), 0400); err != nil {
		t.Fatalf("setup: %v", err)
	}

	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	// Add should fail because save() cannot write to a read-only file
	err = s.Add(models.TodoItem{ID: "x", Title: "fails"})
	if err == nil {
		t.Error("expected error when saving to read-only file")
	}
}

// TestNewStoreLoadsMissingFileGracefully exercises the path where the
// file does not exist yet (load falls through silently).
func TestNewStoreLoadsMissingFileGracefully(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	if len(s.All()) != 0 {
		t.Error("expected empty store when file is missing")
	}
}

// TestUpdateTriggersReSave verifies that Update calls save on a match.
func TestUpdateTriggersReSave(t *testing.T) {
	s := newTestStore(t)
	s.Add(models.TodoItem{ID: "save-1", Title: "Before"})

	err := s.Update("save-1", func(item *models.TodoItem) {
		item.Title = "After"
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Reload to verify persistence
	s2, _ := NewStore(s.path)
	got := s2.Get("save-1")
	if got == nil || got.Title != "After" {
		t.Errorf("expected persisted title 'After', got %v", got)
	}
}

// TestDeleteTriggersReSave verifies that Delete calls save after removal.
func TestDeleteTriggersReSave(t *testing.T) {
	s := newTestStore(t)
	s.Add(models.TodoItem{ID: "del-save", Title: "Gone"})

	if err := s.Delete("del-save"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	s2, _ := NewStore(s.path)
	if s2.Get("del-save") != nil {
		t.Error("deleted item should not be present after reload")
	}
}
