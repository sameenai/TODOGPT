package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/todogpt/daily-briefing/internal/config"
)

func restoreNewsURL() func() {
	orig := hackerNewsBaseURL
	return func() { hackerNewsBaseURL = orig }
}

func TestFetchHackerNewsNonOKStatus(t *testing.T) {
	defer restoreNewsURL()()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()
	hackerNewsBaseURL = ts.URL

	svc := NewNewsService(config.NewsConfig{MaxItems: 5})
	items, err := svc.Fetch()
	if err != nil {
		t.Fatalf("Fetch should not return error (falls back to mock): %v", err)
	}
	if len(items) == 0 {
		t.Error("expected mock items when API returns non-200")
	}
}

func TestFetchHackerNewsInvalidJSONTopStories(t *testing.T) {
	defer restoreNewsURL()()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not a json array{{"))
	}))
	defer ts.Close()
	hackerNewsBaseURL = ts.URL

	svc := NewNewsService(config.NewsConfig{MaxItems: 5})
	items, err := svc.Fetch()
	if err != nil {
		t.Fatalf("Fetch should not return error: %v", err)
	}
	if len(items) == 0 {
		t.Error("expected mock items when JSON decode fails")
	}
}

func TestFetchHackerNewsEmptyStories(t *testing.T) {
	defer restoreNewsURL()()

	// Returns empty story ID list → fetchHackerNews returns "no stories" error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer ts.Close()
	hackerNewsBaseURL = ts.URL

	svc := NewNewsService(config.NewsConfig{MaxItems: 5})
	items, err := svc.Fetch()
	if err != nil {
		t.Fatalf("Fetch should not return error: %v", err)
	}
	if len(items) == 0 {
		t.Error("expected mock items on empty story list")
	}
}

func TestFetchHackerNewsStoryFetchFails(t *testing.T) {
	defer restoreNewsURL()()

	// Top stories returns IDs, but individual item fetches fail
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v0/topstories.json" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[1,2,3]`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()
	hackerNewsBaseURL = ts.URL

	svc := NewNewsService(config.NewsConfig{MaxItems: 3})
	items, err := svc.Fetch()
	if err != nil {
		t.Fatalf("Fetch should not return error: %v", err)
	}
	// All individual fetches failed → no stories → falls back to mock
	if len(items) == 0 {
		t.Error("expected mock items when all story fetches fail")
	}
}

func TestFetchHackerNewsLongDescription(t *testing.T) {
	defer restoreNewsURL()()

	longText := ""
	for i := 0; i < 250; i++ {
		longText += "x"
	}

	story := hnItem{
		ID:    42,
		Title: "Long Story",
		Text:  longText,
		By:    "user",
		Score: 100,
		Time:  1000000,
		Type:  "story",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v0/topstories.json" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[42]`))
		} else {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(story)
		}
	}))
	defer ts.Close()
	hackerNewsBaseURL = ts.URL

	svc := NewNewsService(config.NewsConfig{MaxItems: 1})
	items, err := svc.Fetch()
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one item")
	}
	if len(items[0].Description) > 200 {
		t.Errorf("description should be truncated to 200 chars, got %d", len(items[0].Description))
	}
}

func TestFetchHackerNewsNoURLUsesHNLink(t *testing.T) {
	defer restoreNewsURL()()

	story := hnItem{
		ID:    99,
		Title: "Ask HN: something",
		URL:   "", // no URL → should use HN item link
		By:    "user",
		Score: 50,
		Time:  1000000,
		Type:  "story",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v0/topstories.json" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[99]`))
		} else {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(story)
		}
	}))
	defer ts.Close()
	hackerNewsBaseURL = ts.URL

	svc := NewNewsService(config.NewsConfig{MaxItems: 1})
	items, err := svc.Fetch()
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one item")
	}
	if items[0].URL == "" {
		t.Error("expected HN item URL as fallback")
	}
	if items[0].Description == "" {
		t.Error("expected score/by description fallback")
	}
}

func TestFetchHackerNewsStoryItemInvalidJSON(t *testing.T) {
	defer restoreNewsURL()()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v0/topstories.json" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[1]`))
		} else {
			// Return invalid JSON for the item
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"id":%d,"title":`, 1)))
		}
	}))
	defer ts.Close()
	hackerNewsBaseURL = ts.URL

	svc := NewNewsService(config.NewsConfig{MaxItems: 1})
	items, err := svc.Fetch()
	if err != nil {
		t.Fatalf("Fetch should not error: %v", err)
	}
	// item decode failed → no items → mock fallback
	if len(items) == 0 {
		t.Error("expected mock fallback when item JSON decode fails")
	}
}
