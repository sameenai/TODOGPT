package services

import (
	"testing"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

// TestHubStartPollingTickerFires verifies that StartPolling broadcasts on
// the periodic ticker, not just on initial start.
func TestHubStartPollingTickerFires(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.DataDir = t.TempDir()
	cfg.Server.PollInterval = 1 // 1-second interval

	hub := NewHub(cfg)
	ch := hub.Subscribe()

	go hub.StartPolling()

	// Drain the initial broadcast
	select {
	case <-ch:
	case <-time.After(15 * time.Second):
		t.Fatal("timeout waiting for initial broadcast")
	}

	// Wait for the ticker to fire at least once more
	select {
	case update := <-ch:
		if update.Type != "full_refresh" {
			t.Errorf("expected full_refresh, got %q", update.Type)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("timeout waiting for ticker broadcast")
	}

	hub.Stop()
}

// TestHubStopBeforeTicker ensures Stop() before the ticker fires doesn't block.
func TestHubStopBeforeTicker(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.DataDir = t.TempDir()
	cfg.Server.PollInterval = 60 // long interval

	hub := NewHub(cfg)
	ch := hub.Subscribe()

	go hub.StartPolling()

	// Drain initial
	select {
	case <-ch:
	case <-time.After(15 * time.Second):
		t.Fatal("timeout on initial broadcast")
	}

	// Stop immediately — the ticker branch should never fire
	done := make(chan struct{})
	go func() {
		hub.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() blocked unexpectedly")
	}
}

// TestGenerateFromBriefingAssignReason covers the "assign" priority branch.
func TestGenerateFromBriefingAssignReason(t *testing.T) {
	svc := NewTodoService()
	now := time.Now()

	briefing := &models.Briefing{
		GitHubNotifs: []models.GitHubNotification{
			{
				ID:        "assign-gh",
				Title:     "Assigned issue",
				Repo:      "org/repo",
				Type:      "Issue",
				Reason:    "assign",
				Unread:    true,
				UpdatedAt: now,
			},
		},
	}

	svc.GenerateFromBriefing(briefing)
	items := svc.List()

	if len(items) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(items))
	}
	if items[0].Priority != models.PriorityHigh {
		t.Errorf("expected High priority for 'assign' reason, got %s", items[0].Priority.String())
	}
}

// TestGenerateFromBriefingImportantLabel covers the "important" label branch.
func TestGenerateFromBriefingImportantLabel(t *testing.T) {
	svc := NewTodoService()
	now := time.Now()

	briefing := &models.Briefing{
		UnreadEmails: []models.EmailMessage{
			{
				ID:       "imp-email",
				From:     "boss@co.com",
				Subject:  "Please read",
				IsUnread: true,
				Labels:   []string{"important"},
				Date:     now,
			},
		},
	}

	svc.GenerateFromBriefing(briefing)
	items := svc.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(items))
	}
	if items[0].Priority != models.PriorityHigh {
		t.Errorf("expected High priority for 'important' label, got %s", items[0].Priority.String())
	}
}

// TestGenerateFromBriefingSlackDMTitle covers the DM channel title branch.
func TestGenerateFromBriefingSlackDMTitle(t *testing.T) {
	svc := NewTodoService()
	now := time.Now()

	briefing := &models.Briefing{
		SlackMessages: []models.SlackMessage{
			{
				Channel:   "DM",
				User:      "charlie",
				Text:      "Can you help?",
				Timestamp: now,
				IsDM:      true,
			},
		},
	}

	svc.GenerateFromBriefing(briefing)
	items := svc.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(items))
	}
	if items[0].Source != "slack" {
		t.Errorf("expected source 'slack', got %q", items[0].Source)
	}
}
