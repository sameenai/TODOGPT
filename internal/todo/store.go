package todo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/todogpt/daily-briefing/internal/models"
)

// Store persists todos to disk.
type Store struct {
	path  string
	items []models.TodoItem
	mu    sync.RWMutex
}

func NewStore(path string) (*Store, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir := filepath.Join(home, ".daily-briefing")
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, err
		}
		path = filepath.Join(dir, "todos.json")
	}

	s := &Store{path: path}
	s.load()
	return s, nil
}

func (s *Store) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		s.items = []models.TodoItem{}
		return
	}
	if err := json.Unmarshal(data, &s.items); err != nil {
		s.items = []models.TodoItem{}
	}
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func (s *Store) All() []models.TodoItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]models.TodoItem, len(s.items))
	copy(result, s.items)
	return result
}

func (s *Store) Add(item models.TodoItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, item)
	return s.save()
}

func (s *Store) Update(id string, fn func(*models.TodoItem)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			fn(&s.items[i])
			return s.save()
		}
	}
	return nil
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			s.items = append(s.items[:i], s.items[i+1:]...)
			return s.save()
		}
	}
	return nil
}

func (s *Store) Get(id string) *models.TodoItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.items {
		if item.ID == id {
			return &item
		}
	}
	return nil
}

func (s *Store) SetAll(items []models.TodoItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = items
	return s.save()
}
