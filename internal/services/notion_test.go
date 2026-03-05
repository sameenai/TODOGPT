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

func restoreNotionURL(orig string) func() {
	return func() { notionAPIBaseURL = orig }
}

func sampleNotionResponse() map[string]interface{} {
	due := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	return map[string]interface{}{
		"results": []map[string]interface{}{
			{
				"id":               "page-abc-123",
				"url":              "https://notion.so/page-abc-123",
				"last_edited_time": "2026-03-04T10:00:00Z",
				"properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"type":  "title",
						"title": []map[string]interface{}{{"plain_text": "Write quarterly review"}},
					},
					"Status": map[string]interface{}{
						"type":   "select",
						"select": map[string]interface{}{"name": "In Progress"},
					},
					"Priority": map[string]interface{}{
						"type":   "select",
						"select": map[string]interface{}{"name": "High"},
					},
					"Due": map[string]interface{}{
						"type": "date",
						"date": map[string]interface{}{"start": due},
					},
				},
			},
			{
				"id":               "page-def-456",
				"url":              "https://notion.so/page-def-456",
				"last_edited_time": "2026-03-03T08:00:00Z",
				"properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"type":  "title",
						"title": []map[string]interface{}{{"plain_text": "Update onboarding docs"}},
					},
					"Status": map[string]interface{}{
						"type":   "select",
						"select": map[string]interface{}{"name": "Not Started"},
					},
				},
			},
		},
	}
}

func TestNotionFetchMockWhenDisabled(t *testing.T) {
	svc := NewNotionService(config.NotionConfig{Enabled: false})
	pages, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) == 0 {
		t.Error("expected mock pages when disabled")
	}
}

func TestNotionFetchMockWhenNoToken(t *testing.T) {
	svc := NewNotionService(config.NotionConfig{Enabled: true, DatabaseID: "some-db"})
	pages, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) == 0 {
		t.Error("expected mock pages when no token")
	}
}

func TestNotionFetchMockWhenNoDatabaseID(t *testing.T) {
	svc := NewNotionService(config.NotionConfig{Enabled: true, Token: "tok"})
	pages, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) == 0 {
		t.Error("expected mock pages when no database ID")
	}
}

func TestNotionFetchRealAPI(t *testing.T) {
	defer restoreNotionURL(notionAPIBaseURL)()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Notion-Version") != "2022-06-28" {
			t.Errorf("expected Notion-Version header, got %s", r.Header.Get("Notion-Version"))
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected auth: %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sampleNotionResponse())
	}))
	defer srv.Close()

	notionAPIBaseURL = srv.URL
	svc := NewNotionService(config.NotionConfig{
		Enabled: true, Token: "test-token", DatabaseID: "my-db-id",
	})

	pages, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}
	if pages[0].Title != "Write quarterly review" {
		t.Errorf("unexpected title: %s", pages[0].Title)
	}
	if pages[0].Status != "In Progress" {
		t.Errorf("unexpected status: %s", pages[0].Status)
	}
	if pages[0].Priority != "High" {
		t.Errorf("unexpected priority: %s", pages[0].Priority)
	}
	if pages[0].DueDate == nil {
		t.Error("expected non-nil due date")
	}
	if pages[0].URL != "https://notion.so/page-abc-123" {
		t.Errorf("unexpected URL: %s", pages[0].URL)
	}
	if pages[0].Database != "my-db-id" {
		t.Errorf("unexpected database: %s", pages[0].Database)
	}
	if pages[0].UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
	// Second page: no priority property
	if pages[1].Title != "Update onboarding docs" {
		t.Errorf("unexpected second page title: %s", pages[1].Title)
	}
}

func TestNotionFetchNon200FallsBackToCache(t *testing.T) {
	defer restoreNotionURL(notionAPIBaseURL)()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	notionAPIBaseURL = srv.URL
	svc := NewNotionService(config.NotionConfig{
		Enabled: true, Token: "t", DatabaseID: "db",
	})
	svc.cache = []models.NotionPage{{ID: "cached-1", Title: "cached page"}}

	pages, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) != 1 || pages[0].ID != "cached-1" {
		t.Errorf("expected cached page, got %+v", pages)
	}
}

func TestNotionFetchNon200NoCacheFallsBackToMock(t *testing.T) {
	defer restoreNotionURL(notionAPIBaseURL)()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	notionAPIBaseURL = srv.URL
	svc := NewNotionService(config.NotionConfig{
		Enabled: true, Token: "t", DatabaseID: "db",
	})

	pages, _ := svc.Fetch()
	if len(pages) == 0 {
		t.Error("expected mock fallback pages")
	}
}

func TestNotionFetchBadJSONFallback(t *testing.T) {
	defer restoreNotionURL(notionAPIBaseURL)()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("garbage"))
	}))
	defer srv.Close()

	notionAPIBaseURL = srv.URL
	svc := NewNotionService(config.NotionConfig{
		Enabled: true, Token: "t", DatabaseID: "db",
	})

	pages, _ := svc.Fetch()
	if len(pages) == 0 {
		t.Error("expected mock fallback on JSON decode failure")
	}
}

func TestNotionGetCachedReturnsMockWhenEmpty(t *testing.T) {
	svc := NewNotionService(config.NotionConfig{})
	cached := svc.GetCached()
	if len(cached) == 0 {
		t.Error("expected mock pages from GetCached when cache empty")
	}
}

func TestNotionGetCachedReturnsCache(t *testing.T) {
	svc := NewNotionService(config.NotionConfig{})
	svc.cache = []models.NotionPage{{ID: "p1", Title: "cached"}}
	cached := svc.GetCached()
	if len(cached) != 1 || cached[0].ID != "p1" {
		t.Errorf("expected cached page, got %+v", cached)
	}
}

func TestNotionMockPagesHaveRequiredFields(t *testing.T) {
	svc := NewNotionService(config.NotionConfig{})
	mocks := svc.mockPages()
	for _, p := range mocks {
		if p.ID == "" {
			t.Error("mock page missing ID")
		}
		if p.Title == "" {
			t.Error("mock page missing Title")
		}
		if p.Status == "" {
			t.Error("mock page missing Status")
		}
	}
}

func TestNotionFetchAPIDateRFC3339(t *testing.T) {
	defer restoreNotionURL(notionAPIBaseURL)()

	// Test RFC3339 date format in addition to YYYY-MM-DD
	rfc3339Date := time.Now().AddDate(0, 0, 3).Format(time.RFC3339)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"id":               "page-rfc",
					"url":              "https://notion.so/page-rfc",
					"last_edited_time": "2026-03-04T10:00:00Z",
					"properties": map[string]interface{}{
						"Name": map[string]interface{}{
							"type":  "title",
							"title": []map[string]interface{}{{"plain_text": "RFC date test"}},
						},
						"Due": map[string]interface{}{
							"type": "date",
							"date": map[string]interface{}{"start": rfc3339Date},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	notionAPIBaseURL = srv.URL
	svc := NewNotionService(config.NotionConfig{
		Enabled: true, Token: "tok", DatabaseID: "db",
	})

	pages, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) == 0 {
		t.Fatal("expected pages")
	}
	if pages[0].DueDate == nil {
		t.Error("expected DueDate parsed from RFC3339 format")
	}
}
