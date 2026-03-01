package todo

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/todogpt/daily-briefing/internal/models"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test-todos.json")
	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	return s
}

func TestNewStore(t *testing.T) {
	s := newTestStore(t)
	items := s.All()
	if len(items) != 0 {
		t.Errorf("expected empty store, got %d items", len(items))
	}
}

func TestNewStoreDefaultPath(t *testing.T) {
	s, err := NewStore("")
	if err != nil {
		t.Fatalf("NewStore with empty path failed: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestAddAndGet(t *testing.T) {
	s := newTestStore(t)

	item := models.TodoItem{
		ID:       "test-1",
		Title:    "Test Task",
		Priority: models.PriorityHigh,
		Status:   models.TodoPending,
	}

	if err := s.Add(item); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	got := s.Get("test-1")
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Title != "Test Task" {
		t.Errorf("expected title 'Test Task', got %q", got.Title)
	}
	if got.Priority != models.PriorityHigh {
		t.Errorf("expected priority High, got %d", got.Priority)
	}
}

func TestGetNonExistent(t *testing.T) {
	s := newTestStore(t)
	got := s.Get("nonexistent")
	if got != nil {
		t.Error("expected nil for non-existent item")
	}
}

func TestAll(t *testing.T) {
	s := newTestStore(t)

	for i := 0; i < 5; i++ {
		s.Add(models.TodoItem{
			ID:    "item-" + string(rune('a'+i)),
			Title: "Task " + string(rune('A'+i)),
		})
	}

	items := s.All()
	if len(items) != 5 {
		t.Errorf("expected 5 items, got %d", len(items))
	}
}

func TestAllReturnsCopy(t *testing.T) {
	s := newTestStore(t)
	s.Add(models.TodoItem{ID: "orig", Title: "Original"})

	items := s.All()
	items[0].Title = "Modified"

	got := s.Get("orig")
	if got.Title != "Original" {
		t.Error("All() should return a copy, not a reference")
	}
}

func TestUpdate(t *testing.T) {
	s := newTestStore(t)
	s.Add(models.TodoItem{
		ID:     "upd-1",
		Title:  "Before",
		Status: models.TodoPending,
	})

	err := s.Update("upd-1", func(item *models.TodoItem) {
		item.Title = "After"
		item.Status = models.TodoDone
		now := time.Now()
		item.CompletedAt = &now
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got := s.Get("upd-1")
	if got.Title != "After" {
		t.Errorf("expected title 'After', got %q", got.Title)
	}
	if got.Status != models.TodoDone {
		t.Errorf("expected status Done, got %d", got.Status)
	}
	if got.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestUpdateNonExistent(t *testing.T) {
	s := newTestStore(t)
	err := s.Update("ghost", func(item *models.TodoItem) {
		item.Title = "Should not happen"
	})
	if err != nil {
		t.Fatalf("Update on non-existent should not error, got: %v", err)
	}
}

func TestDelete(t *testing.T) {
	s := newTestStore(t)
	s.Add(models.TodoItem{ID: "del-1", Title: "Delete Me"})
	s.Add(models.TodoItem{ID: "del-2", Title: "Keep Me"})

	if err := s.Delete("del-1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if s.Get("del-1") != nil {
		t.Error("deleted item still exists")
	}
	if s.Get("del-2") == nil {
		t.Error("non-deleted item was removed")
	}

	items := s.All()
	if len(items) != 1 {
		t.Errorf("expected 1 item after delete, got %d", len(items))
	}
}

func TestDeleteNonExistent(t *testing.T) {
	s := newTestStore(t)
	err := s.Delete("ghost")
	if err != nil {
		t.Fatalf("Delete non-existent should not error, got: %v", err)
	}
}

func TestSetAll(t *testing.T) {
	s := newTestStore(t)
	s.Add(models.TodoItem{ID: "old-1", Title: "Old"})

	newItems := []models.TodoItem{
		{ID: "new-1", Title: "New 1"},
		{ID: "new-2", Title: "New 2"},
		{ID: "new-3", Title: "New 3"},
	}

	if err := s.SetAll(newItems); err != nil {
		t.Fatalf("SetAll failed: %v", err)
	}

	items := s.All()
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
	if s.Get("old-1") != nil {
		t.Error("old item should be replaced")
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "persist-todos.json")

	// Create store and add items
	s1, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	s1.Add(models.TodoItem{ID: "persist-1", Title: "Persisted"})
	s1.Add(models.TodoItem{ID: "persist-2", Title: "Also Persisted"})

	// Create new store from same path
	s2, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore reload failed: %v", err)
	}

	items := s2.All()
	if len(items) != 2 {
		t.Errorf("expected 2 persisted items, got %d", len(items))
	}
	if s2.Get("persist-1") == nil {
		t.Error("persisted item not found")
	}
	if s2.Get("persist-1").Title != "Persisted" {
		t.Errorf("expected title 'Persisted', got %q", s2.Get("persist-1").Title)
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := newTestStore(t)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			item := models.TodoItem{
				ID:    "concurrent-" + string(rune('A'+i%26)) + string(rune('0'+i/26)),
				Title: "Concurrent Task",
			}
			s.Add(item)
		}(i)
	}

	// Also read concurrently
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.All()
		}()
	}

	wg.Wait()

	items := s.All()
	if len(items) != 50 {
		t.Errorf("expected 50 items from concurrent adds, got %d", len(items))
	}
}
