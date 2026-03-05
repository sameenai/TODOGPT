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

func restoreJiraPath(orig string) func() {
	return func() { jiraAPIPath = orig }
}

func sampleJiraResponse() map[string]interface{} {
	return map[string]interface{}{
		"issues": []map[string]interface{}{
			{
				"key": "PROJ-42",
				"fields": map[string]interface{}{
					"summary":     "Fix the login bug",
					"status":      map[string]interface{}{"name": "In Progress"},
					"priority":    map[string]interface{}{"name": "High"},
					"assignee":    map[string]interface{}{"displayName": "Alice"},
					"duedate":     "2026-04-01",
					"issuetype":   map[string]interface{}{"name": "Bug"},
				},
			},
			{
				"key": "PROJ-43",
				"fields": map[string]interface{}{
					"summary":   "Add dark mode",
					"status":    map[string]interface{}{"name": "To Do"},
					"priority":  map[string]interface{}{"name": "Medium"},
					"assignee":  nil,
					"duedate":   "",
					"issuetype": map[string]interface{}{"name": "Story"},
				},
			},
		},
	}
}

func TestJiraFetchMockWhenDisabled(t *testing.T) {
	svc := NewJiraService(config.JiraConfig{Enabled: false})
	tickets, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tickets) == 0 {
		t.Error("expected mock tickets when disabled")
	}
}

func TestJiraFetchMockWhenNoToken(t *testing.T) {
	svc := NewJiraService(config.JiraConfig{Enabled: true, BaseURL: "http://jira.example.com", Project: "PROJ"})
	tickets, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tickets) == 0 {
		t.Error("expected mock tickets when no token configured")
	}
}

func TestJiraFetchRealAPI(t *testing.T) {
	defer restoreJiraPath(jiraAPIPath)()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Error("expected Accept: application/json")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sampleJiraResponse())
	}))
	defer srv.Close()

	jiraAPIPath = "/rest/api/3/search"
	svc := NewJiraService(config.JiraConfig{
		Enabled: true, BaseURL: srv.URL, Email: "user@example.com",
		Token: "test-token", Project: "PROJ",
	})

	tickets, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tickets) != 2 {
		t.Fatalf("expected 2 tickets, got %d", len(tickets))
	}
	if tickets[0].Key != "PROJ-42" {
		t.Errorf("expected PROJ-42, got %s", tickets[0].Key)
	}
	if tickets[0].Summary != "Fix the login bug" {
		t.Errorf("unexpected summary: %s", tickets[0].Summary)
	}
	if tickets[0].Status != "In Progress" {
		t.Errorf("unexpected status: %s", tickets[0].Status)
	}
	if tickets[0].Priority != "High" {
		t.Errorf("unexpected priority: %s", tickets[0].Priority)
	}
	if tickets[0].Assignee != "Alice" {
		t.Errorf("unexpected assignee: %s", tickets[0].Assignee)
	}
	if tickets[0].Type != "Bug" {
		t.Errorf("unexpected type: %s", tickets[0].Type)
	}
	expected, _ := time.Parse("2006-01-02", "2026-04-01")
	if !tickets[0].DueDate.Equal(expected) {
		t.Errorf("unexpected due date: %v", tickets[0].DueDate)
	}
	if tickets[0].URL != srv.URL+"/browse/PROJ-42" {
		t.Errorf("unexpected URL: %s", tickets[0].URL)
	}
	// Second ticket: no assignee, no due date
	if tickets[1].Assignee != "" {
		t.Errorf("expected empty assignee, got %s", tickets[1].Assignee)
	}
	if !tickets[1].DueDate.IsZero() {
		t.Errorf("expected zero due date, got %v", tickets[1].DueDate)
	}
}

func TestJiraFetchNon200FallsBackToCache(t *testing.T) {
	defer restoreJiraPath(jiraAPIPath)()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	jiraAPIPath = "/rest/api/3/search"
	svc := NewJiraService(config.JiraConfig{
		Enabled: true, BaseURL: srv.URL, Email: "u", Token: "t", Project: "P",
	})
	// Pre-populate cache
	svc.cache = []models.JiraTicket{{Key: "CACHED-1", Summary: "cached ticket"}}

	tickets, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tickets) != 1 || tickets[0].Key != "CACHED-1" {
		t.Errorf("expected cached ticket, got %+v", tickets)
	}
}

func TestJiraFetchNon200NoCacheFallsBackToMock(t *testing.T) {
	defer restoreJiraPath(jiraAPIPath)()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	jiraAPIPath = "/rest/api/3/search"
	svc := NewJiraService(config.JiraConfig{
		Enabled: true, BaseURL: srv.URL, Email: "u", Token: "t", Project: "P",
	})

	tickets, _ := svc.Fetch()
	if len(tickets) == 0 {
		t.Error("expected mock fallback tickets")
	}
}

func TestJiraFetchBadJSONFallback(t *testing.T) {
	defer restoreJiraPath(jiraAPIPath)()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	jiraAPIPath = "/rest/api/3/search"
	svc := NewJiraService(config.JiraConfig{
		Enabled: true, BaseURL: srv.URL, Email: "u", Token: "t", Project: "P",
	})

	tickets, _ := svc.Fetch()
	if len(tickets) == 0 {
		t.Error("expected mock fallback on JSON decode failure")
	}
}

func TestJiraGetCachedReturnsMockWhenEmpty(t *testing.T) {
	svc := NewJiraService(config.JiraConfig{Enabled: true})
	cached := svc.GetCached()
	if len(cached) == 0 {
		t.Error("expected mock tickets from GetCached when cache empty")
	}
}

func TestJiraGetCachedReturnsCache(t *testing.T) {
	svc := NewJiraService(config.JiraConfig{})
	svc.cache = []models.JiraTicket{{Key: "X-1", Summary: "cached"}}
	cached := svc.GetCached()
	if len(cached) != 1 || cached[0].Key != "X-1" {
		t.Errorf("expected cached ticket, got %+v", cached)
	}
}

func TestJiraMockTicketsHaveRequiredFields(t *testing.T) {
	svc := NewJiraService(config.JiraConfig{})
	mocks := svc.mockTickets()
	for _, m := range mocks {
		if m.Key == "" {
			t.Error("mock ticket missing Key")
		}
		if m.Summary == "" {
			t.Error("mock ticket missing Summary")
		}
		if m.Status == "" {
			t.Error("mock ticket missing Status")
		}
		if m.Priority == "" {
			t.Error("mock ticket missing Priority")
		}
	}
}
