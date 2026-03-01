package services

import (
	"testing"

	"github.com/todogpt/daily-briefing/internal/config"
)

func TestNewGitHubService(t *testing.T) {
	svc := NewGitHubService(config.GitHubConfig{})
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestGitHubFetch(t *testing.T) {
	svc := NewGitHubService(config.GitHubConfig{Enabled: true})

	notifs, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifs) == 0 {
		t.Error("expected mock notifications")
	}

	for _, n := range notifs {
		if n.ID == "" {
			t.Error("notification should have an ID")
		}
		if n.Title == "" {
			t.Error("notification should have a title")
		}
		if n.Repo == "" {
			t.Error("notification should have a repo")
		}
		if n.Type == "" {
			t.Error("notification should have a type")
		}
		if n.UpdatedAt.IsZero() {
			t.Error("notification should have an update time")
		}
	}
}

func TestGitHubGetCachedEmpty(t *testing.T) {
	svc := NewGitHubService(config.GitHubConfig{})
	notifs := svc.GetCached()
	if len(notifs) == 0 {
		t.Error("expected mock notifications when cache empty")
	}
}

func TestGitHubGetCachedAfterFetch(t *testing.T) {
	svc := NewGitHubService(config.GitHubConfig{})
	svc.Fetch()
	cached := svc.GetCached()
	if len(cached) == 0 {
		t.Error("expected cached notifications")
	}
}

func TestGitHubMockHasTypes(t *testing.T) {
	svc := NewGitHubService(config.GitHubConfig{})
	notifs := svc.mockNotifications()

	hasPR := false
	hasIssue := false
	for _, n := range notifs {
		if n.Type == "PullRequest" {
			hasPR = true
		}
		if n.Type == "Issue" {
			hasIssue = true
		}
	}

	if !hasPR {
		t.Error("expected at least one PullRequest notification")
	}
	if !hasIssue {
		t.Error("expected at least one Issue notification")
	}
}

func TestGitHubMockHasUnread(t *testing.T) {
	svc := NewGitHubService(config.GitHubConfig{})
	notifs := svc.mockNotifications()

	unread := 0
	for _, n := range notifs {
		if n.Unread {
			unread++
		}
	}
	if unread == 0 {
		t.Error("expected at least one unread notification")
	}
}

func TestGitHubMockReasons(t *testing.T) {
	svc := NewGitHubService(config.GitHubConfig{})
	notifs := svc.mockNotifications()

	reasons := make(map[string]bool)
	for _, n := range notifs {
		reasons[n.Reason] = true
	}

	if !reasons["review_requested"] {
		t.Error("expected review_requested reason in mock data")
	}
	if !reasons["author"] {
		t.Error("expected author reason in mock data")
	}
}
