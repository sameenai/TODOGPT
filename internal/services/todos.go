package services

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/todogpt/daily-briefing/internal/models"
)

// todoStorer is the persistence backend for TodoService.
// todo.Store satisfies this interface.
type todoStorer interface {
	All() []models.TodoItem
	SetAll([]models.TodoItem) error
}

// TodoService manages the smart interactive todo list.
type TodoService struct {
	items []models.TodoItem
	mu    sync.RWMutex
	seen  map[string]bool // track source IDs to avoid duplicates
	store todoStorer      // optional persistence backend; nil = in-memory only
}

func NewTodoService() *TodoService {
	return &TodoService{
		items: []models.TodoItem{},
		seen:  make(map[string]bool),
	}
}

// NewTodoServiceWithStore creates a TodoService backed by persistent storage.
// Existing items are loaded from store immediately, and the seen map is
// populated so GenerateFromBriefing won't re-add already-persisted todos.
func NewTodoServiceWithStore(store todoStorer) *TodoService {
	svc := &TodoService{
		items: []models.TodoItem{},
		seen:  make(map[string]bool),
		store: store,
	}
	if items := store.All(); len(items) > 0 {
		svc.items = items
		for _, item := range items {
			if item.Source != "" && item.SourceID != "" {
				// Add both forms to handle the different key formats used by
				// GenerateFromBriefing (email/github/calendar use "source:rawID";
				// slack stores the full key in SourceID directly).
				svc.seen[item.SourceID] = true
				svc.seen[item.Source+":"+item.SourceID] = true
			}
		}
	}
	return svc
}

// persist saves the current items to the backing store, if one is set.
// Must be called with s.mu held.
func (s *TodoService) persist() {
	if s.store == nil {
		return
	}
	if err := s.store.SetAll(s.items); err != nil {
		log.Printf("todo persist error: %v", err)
	}
}

func (s *TodoService) List() []models.TodoItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]models.TodoItem, len(s.items))
	copy(result, s.items)
	return result
}

func (s *TodoService) Add(item models.TodoItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item.ID == "" {
		item.ID = fmt.Sprintf("todo-%d", time.Now().UnixNano())
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	item.UpdatedAt = time.Now()
	s.items = append(s.items, item)
	s.persist()
}

func (s *TodoService) Update(id string, fn func(*models.TodoItem)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			fn(&s.items[i])
			s.items[i].UpdatedAt = time.Now()
			s.persist()
			return true
		}
	}
	return false
}

func (s *TodoService) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			s.items = append(s.items[:i], s.items[i+1:]...)
			s.persist()
			return true
		}
	}
	return false
}

func (s *TodoService) Complete(id string) bool {
	now := time.Now()
	return s.Update(id, func(t *models.TodoItem) {
		t.Status = models.TodoDone
		t.CompletedAt = &now
	})
}

func (s *TodoService) SetItems(items []models.TodoItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = items
	s.persist()
}

// GenerateFromBriefing extracts actionable todos from all signal sources.
func (s *TodoService) GenerateFromBriefing(b *models.Briefing) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Extract from emails
	for _, email := range b.UnreadEmails {
		sourceID := "email:" + email.ID
		if s.seen[sourceID] {
			continue
		}
		s.seen[sourceID] = true

		priority := models.PriorityMedium
		if email.IsStarred {
			priority = models.PriorityHigh
		}
		for _, label := range email.Labels {
			if label == "important" {
				priority = models.PriorityHigh
			}
		}
		if strings.Contains(strings.ToLower(email.Subject), "action required") ||
			strings.Contains(strings.ToLower(email.Subject), "urgent") {
			priority = models.PriorityUrgent
		}

		s.items = append(s.items, models.TodoItem{
			ID:          fmt.Sprintf("todo-email-%s", email.ID),
			Title:       fmt.Sprintf("Reply: %s", email.Subject),
			Description: fmt.Sprintf("From: %s", email.From),
			Priority:    priority,
			Status:      models.TodoPending,
			Source:      "email",
			SourceID:    email.ID,
			Tags:        email.Labels,
			CreatedAt:   email.Date,
			UpdatedAt:   time.Now(),
		})
	}

	// Extract from Slack DMs and urgent messages
	for i, msg := range b.SlackMessages {
		sourceID := fmt.Sprintf("slack:%s:%d", msg.Channel, i)
		if s.seen[sourceID] {
			continue
		}
		s.seen[sourceID] = true

		if !msg.IsDM && !msg.IsUrgent {
			continue
		}

		priority := models.PriorityMedium
		if msg.IsUrgent {
			priority = models.PriorityHigh
		}
		if msg.IsDM {
			priority = models.PriorityHigh
		}

		title := fmt.Sprintf("Respond to %s", msg.User)
		if msg.Channel != "DM" {
			title = fmt.Sprintf("Check %s: %s", msg.Channel, truncate(msg.Text, 50))
		}

		s.items = append(s.items, models.TodoItem{
			ID:          fmt.Sprintf("todo-slack-%d", i),
			Title:       title,
			Description: truncate(msg.Text, 200),
			Priority:    priority,
			Status:      models.TodoPending,
			Source:      "slack",
			SourceID:    sourceID,
			Tags:        []string{msg.Channel},
			CreatedAt:   msg.Timestamp,
			UpdatedAt:   time.Now(),
		})
	}

	// Extract from GitHub notifications
	for _, notif := range b.GitHubNotifs {
		sourceID := "github:" + notif.ID
		if s.seen[sourceID] {
			continue
		}
		s.seen[sourceID] = true

		if !notif.Unread {
			continue
		}

		priority := models.PriorityMedium
		if notif.Reason == "review_requested" || notif.Reason == "assign" {
			priority = models.PriorityHigh
		}

		action := "Review"
		if notif.Type == "Issue" {
			action = "Check"
		}

		s.items = append(s.items, models.TodoItem{
			ID:          fmt.Sprintf("todo-gh-%s", notif.ID),
			Title:       fmt.Sprintf("%s: %s", action, notif.Title),
			Description: fmt.Sprintf("[%s] %s — %s", notif.Repo, notif.Type, notif.Reason),
			Priority:    priority,
			Status:      models.TodoPending,
			Source:      "github",
			SourceID:    notif.ID,
			SourceURL:   notif.URL,
			Tags:        []string{notif.Repo, notif.Type},
			CreatedAt:   notif.UpdatedAt,
			UpdatedAt:   time.Now(),
		})
	}

	// Extract from Jira tickets assigned to user
	for _, ticket := range b.JiraTickets {
		sourceID := "jira:" + ticket.Key
		if s.seen[sourceID] {
			continue
		}
		s.seen[sourceID] = true

		priority := models.PriorityMedium
		switch ticket.Priority {
		case "Critical", "Blocker":
			priority = models.PriorityUrgent
		case "High", "Major":
			priority = models.PriorityHigh
		case "Low", "Minor", "Trivial":
			priority = models.PriorityLow
		}

		var dueDate *time.Time
		if !ticket.DueDate.IsZero() {
			d := ticket.DueDate
			dueDate = &d
		}

		s.items = append(s.items, models.TodoItem{
			ID:          fmt.Sprintf("todo-jira-%s", ticket.Key),
			Title:       fmt.Sprintf("[%s] %s", ticket.Key, ticket.Summary),
			Description: fmt.Sprintf("Status: %s · Type: %s", ticket.Status, ticket.Type),
			Priority:    priority,
			Status:      models.TodoPending,
			Source:      "jira",
			SourceID:    ticket.Key,
			SourceURL:   ticket.URL,
			DueDate:     dueDate,
			Tags:        []string{ticket.Type, ticket.Status},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	// Extract from Notion pages
	for _, page := range b.NotionPages {
		sourceID := "notion:" + page.ID
		if s.seen[sourceID] {
			continue
		}
		s.seen[sourceID] = true

		priority := models.PriorityMedium
		switch page.Priority {
		case "Critical", "Urgent":
			priority = models.PriorityUrgent
		case "High":
			priority = models.PriorityHigh
		case "Low":
			priority = models.PriorityLow
		}

		s.items = append(s.items, models.TodoItem{
			ID:          fmt.Sprintf("todo-notion-%s", page.ID),
			Title:       page.Title,
			Description: fmt.Sprintf("Status: %s", page.Status),
			Priority:    priority,
			Status:      models.TodoPending,
			Source:      "notion",
			SourceID:    page.ID,
			SourceURL:   page.URL,
			DueDate:     page.DueDate,
			Tags:        []string{"notion"},
			CreatedAt:   page.UpdatedAt,
			UpdatedAt:   time.Now(),
		})
	}

	// Extract from calendar (upcoming meetings needing prep)
	for _, evt := range b.Events {
		sourceID := "calendar:" + evt.ID
		if s.seen[sourceID] {
			continue
		}
		s.seen[sourceID] = true

		if evt.Description != "" {
			s.items = append(s.items, models.TodoItem{
				ID:          fmt.Sprintf("todo-cal-%s", evt.ID),
				Title:       fmt.Sprintf("Prepare for: %s", evt.Title),
				Description: evt.Description,
				Priority:    models.PriorityMedium,
				Status:      models.TodoPending,
				Source:      "calendar",
				SourceID:    evt.ID,
				DueDate:     &evt.StartTime,
				Tags:        []string{"meeting"},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			})
		}
	}

	s.persist()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
