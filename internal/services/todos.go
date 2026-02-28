package services

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/todogpt/daily-briefing/internal/models"
)

// TodoService manages the smart interactive todo list.
type TodoService struct {
	items []models.TodoItem
	mu    sync.RWMutex
	seen  map[string]bool // track source IDs to avoid duplicates
}

func NewTodoService() *TodoService {
	return &TodoService{
		items: []models.TodoItem{},
		seen:  make(map[string]bool),
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
}

func (s *TodoService) Update(id string, fn func(*models.TodoItem)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			fn(&s.items[i])
			s.items[i].UpdatedAt = time.Now()
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
			ID:        fmt.Sprintf("todo-email-%s", email.ID),
			Title:     fmt.Sprintf("Reply: %s", email.Subject),
			Description: fmt.Sprintf("From: %s", email.From),
			Priority:  priority,
			Status:    models.TodoPending,
			Source:    "email",
			SourceID:  email.ID,
			Tags:      email.Labels,
			CreatedAt: email.Date,
			UpdatedAt: time.Now(),
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
			ID:        fmt.Sprintf("todo-slack-%d", i),
			Title:     title,
			Description: truncate(msg.Text, 200),
			Priority:  priority,
			Status:    models.TodoPending,
			Source:    "slack",
			SourceID:  sourceID,
			Tags:      []string{msg.Channel},
			CreatedAt: msg.Timestamp,
			UpdatedAt: time.Now(),
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
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
