package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/todogpt/daily-briefing/internal/models"
	"github.com/todogpt/daily-briefing/internal/todo"
)

// openStore is a test helper that creates a Store backed by a temp file.
func openStore(t *testing.T) (*todo.Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "todos.json")
	store, err := todo.NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store, path
}

// TestTodoPersistenceRoundTrip verifies that items survive a service restart.
func TestTodoPersistenceRoundTrip(t *testing.T) {
	store, path := openStore(t)

	svc := NewTodoServiceWithStore(store)
	svc.Add(models.TodoItem{Title: "Buy milk", Source: "manual"})
	svc.Add(models.TodoItem{Title: "Fix bug #42", Source: "manual"})

	if got := len(svc.List()); got != 2 {
		t.Fatalf("expected 2 items, got %d", got)
	}

	// Simulate restart: open fresh store from same path, new service
	store2, err := todo.NewStore(path)
	if err != nil {
		t.Fatalf("NewStore (restart): %v", err)
	}
	svc2 := NewTodoServiceWithStore(store2)

	items := svc2.List()
	if len(items) != 2 {
		t.Fatalf("after restart: expected 2 items, got %d", len(items))
	}
	titles := map[string]bool{}
	for _, it := range items {
		titles[it.Title] = true
	}
	if !titles["Buy milk"] || !titles["Fix bug #42"] {
		t.Errorf("unexpected titles after restart: %v", titles)
	}
}

// TestTodoPersistenceDelete verifies deletions are persisted.
func TestTodoPersistenceDelete(t *testing.T) {
	store, path := openStore(t)

	svc := NewTodoServiceWithStore(store)
	svc.Add(models.TodoItem{ID: "a", Title: "Keep me", Source: "manual"})
	svc.Add(models.TodoItem{ID: "b", Title: "Delete me", Source: "manual"})
	svc.Delete("b")

	store2, _ := todo.NewStore(path)
	svc2 := NewTodoServiceWithStore(store2)
	items := svc2.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 item after delete+restart, got %d", len(items))
	}
	if items[0].Title != "Keep me" {
		t.Errorf("unexpected item: %q", items[0].Title)
	}
}

// TestTodoPersistenceComplete verifies completions are persisted.
func TestTodoPersistenceComplete(t *testing.T) {
	store, path := openStore(t)

	svc := NewTodoServiceWithStore(store)
	svc.Add(models.TodoItem{ID: "x", Title: "Task", Source: "manual"})
	svc.Complete("x")

	store2, _ := todo.NewStore(path)
	svc2 := NewTodoServiceWithStore(store2)
	items := svc2.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Status != models.TodoDone {
		t.Errorf("expected TodoDone, got %v", items[0].Status)
	}
	if items[0].CompletedAt == nil {
		t.Error("CompletedAt should be set after Complete()")
	}
}

// TestTodoPersistenceNoDuplicatesAfterRestart verifies that GenerateFromBriefing
// does not re-add source-generated todos that were already loaded from disk.
func TestTodoPersistenceNoDuplicatesAfterRestart(t *testing.T) {
	store, path := openStore(t)

	// First process: generate from briefing, persist
	svc := NewTodoServiceWithStore(store)
	briefing := &models.Briefing{
		GitHubNotifs: []models.GitHubNotification{
			{ID: "notif-1", Title: "PR review", Repo: "org/repo", Type: "PullRequest", Reason: "review_requested", Unread: true},
		},
	}
	svc.GenerateFromBriefing(briefing)
	if got := len(svc.List()); got != 1 {
		t.Fatalf("expected 1 item, got %d", got)
	}

	// Second process: load from disk, generate again — should still be 1
	store2, _ := todo.NewStore(path)
	svc2 := NewTodoServiceWithStore(store2)
	svc2.GenerateFromBriefing(briefing)
	if got := len(svc2.List()); got != 1 {
		t.Errorf("after restart + re-generate: expected 1 item (no duplicates), got %d", got)
	}
}

// TestTodoPersistenceFallbackOnStoreError verifies that a bad store path falls
// back to in-memory operation gracefully (NewHub path).
func TestTodoPersistenceFallbackOnStoreError(t *testing.T) {
	// Create a file where a directory would be needed, so NewStore fails
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	// Try to create a store inside a file (not a dir) — MkdirAll will fail
	_, err := todo.NewStore(filepath.Join(blocker, "sub", "todos.json"))
	if err == nil {
		t.Skip("OS allowed nested path inside file — skipping")
	}
	// Confirm NewTodoService() works fine (the fallback path in newTodoService)
	svc := NewTodoService()
	svc.Add(models.TodoItem{Title: "mem only"})
	if got := len(svc.List()); got != 1 {
		t.Errorf("in-memory fallback: expected 1, got %d", got)
	}
}
