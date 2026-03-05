package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

func restoreGitHubURL() func() {
	orig := githubAPIBaseURL
	return func() { githubAPIBaseURL = orig }
}

// sampleAPINotifications returns a minimal valid API payload.
func sampleAPINotifications() []githubNotification {
	return []githubNotification{
		{
			ID: "n1",
			Repo: struct {
				FullName string `json:"full_name"`
			}{FullName: "org/repo"},
			Subject: struct {
				Title string `json:"title"`
				URL   string `json:"url"`
				Type  string `json:"type"`
			}{Title: "Fix bug", URL: "https://api.github.com/repos/org/repo/pulls/1", Type: "PullRequest"},
			Reason:    "review_requested",
			Unread:    true,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		},
		{
			ID: "n2",
			Repo: struct {
				FullName string `json:"full_name"`
			}{FullName: "org/other"},
			Subject: struct {
				Title string `json:"title"`
				URL   string `json:"url"`
				Type  string `json:"type"`
			}{Title: "Issue report", URL: "", Type: "Issue"},
			Reason:    "mention",
			Unread:    false,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}
}

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

func TestGitHubFetchDisabledReturnsMock(t *testing.T) {
	svc := NewGitHubService(config.GitHubConfig{Enabled: false})
	notifs, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifs) == 0 {
		t.Error("expected mock notifications when disabled")
	}
}

func TestGitHubFetchNoTokenReturnsMock(t *testing.T) {
	svc := NewGitHubService(config.GitHubConfig{Enabled: true, Token: ""})
	notifs, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifs) == 0 {
		t.Error("expected mock notifications when no token")
	}
}

func TestGitHubFetchReal(t *testing.T) {
	defer restoreGitHubURL()()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sampleAPINotifications())
	}))
	defer ts.Close()
	githubAPIBaseURL = ts.URL

	svc := NewGitHubService(config.GitHubConfig{Enabled: true, Token: "test-token"})
	notifs, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifs) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(notifs))
	}
	if notifs[0].ID != "n1" || notifs[0].Title != "Fix bug" {
		t.Errorf("unexpected first notification: %+v", notifs[0])
	}
	if notifs[0].Type != "PullRequest" || !notifs[0].Unread {
		t.Errorf("unexpected fields: type=%s unread=%v", notifs[0].Type, notifs[0].Unread)
	}
}

func TestGitHubFetchRepoFilter(t *testing.T) {
	defer restoreGitHubURL()()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(sampleAPINotifications())
	}))
	defer ts.Close()
	githubAPIBaseURL = ts.URL

	svc := NewGitHubService(config.GitHubConfig{
		Enabled: true,
		Token:   "tok",
		Repos:   []string{"org/repo"},
	})
	notifs, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifs) != 1 || notifs[0].Repo != "org/repo" {
		t.Errorf("expected 1 notification from org/repo, got %v", notifs)
	}
}

func TestGitHubFetchNon200FallsBackToMock(t *testing.T) {
	defer restoreGitHubURL()()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer ts.Close()
	githubAPIBaseURL = ts.URL

	svc := NewGitHubService(config.GitHubConfig{Enabled: true, Token: "tok"})
	notifs, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifs) == 0 {
		t.Error("expected fallback mock on API error")
	}
}

func TestGitHubFetchInvalidJSONFallsBackToMock(t *testing.T) {
	defer restoreGitHubURL()()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer ts.Close()
	githubAPIBaseURL = ts.URL

	svc := NewGitHubService(config.GitHubConfig{Enabled: true, Token: "tok"})
	notifs, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifs) == 0 {
		t.Error("expected fallback mock on JSON decode error")
	}
}

func TestGitHubFetchFallsBackToCacheOnError(t *testing.T) {
	defer restoreGitHubURL()()

	svc := NewGitHubService(config.GitHubConfig{Enabled: true, Token: "tok"})
	svc.mu.Lock()
	svc.cache = []models.GitHubNotification{{ID: "cached-1", Title: "Cached"}}
	svc.mu.Unlock()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer ts.Close()
	githubAPIBaseURL = ts.URL

	notifs, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifs) != 1 || notifs[0].ID != "cached-1" {
		t.Errorf("expected cached notification, got %v", notifs)
	}
}

func TestGitHubFetchUpdatedAtParseError(t *testing.T) {
	defer restoreGitHubURL()()

	bad := []githubNotification{{
		ID: "x",
		Repo: struct {
			FullName string `json:"full_name"`
		}{FullName: "org/r"},
		Subject: struct {
			Title string `json:"title"`
			URL   string `json:"url"`
			Type  string `json:"type"`
		}{Title: "T", Type: "Issue"},
		Reason:    "mention",
		Unread:    true,
		UpdatedAt: "not-a-date",
	}}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(bad)
	}))
	defer ts.Close()
	githubAPIBaseURL = ts.URL

	svc := NewGitHubService(config.GitHubConfig{Enabled: true, Token: "tok"})
	notifs, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifs) != 1 || !notifs[0].UpdatedAt.IsZero() {
		t.Errorf("expected zero UpdatedAt for unparseable date, got %v", notifs)
	}
}

func TestContainsRepo(t *testing.T) {
	if !containsRepo([]string{"org/a", "org/b"}, "org/a") {
		t.Error("expected to find org/a")
	}
	if containsRepo([]string{"org/a"}, "org/c") {
		t.Error("expected not to find org/c")
	}
	if containsRepo(nil, "org/a") {
		t.Error("expected false for nil repos")
	}
}
