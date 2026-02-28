package services

import (
	"testing"

	"github.com/todogpt/daily-briefing/internal/config"
)

func TestNewNewsService(t *testing.T) {
	svc := NewNewsService(config.NewsConfig{
		Sources:  []string{"techcrunch"},
		MaxItems: 5,
	})
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewsFetchNoAPIKey(t *testing.T) {
	svc := NewNewsService(config.NewsConfig{
		Sources:  []string{"techcrunch"},
		MaxItems: 10,
	})

	items, err := svc.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) == 0 {
		t.Error("expected mock news items")
	}
	for _, item := range items {
		if item.Title == "" {
			t.Error("news item should have a title")
		}
		if item.Source == "" {
			t.Error("news item should have a source")
		}
		if item.PublishedAt.IsZero() {
			t.Error("news item should have a publish time")
		}
	}
}

func TestNewsGetCachedEmpty(t *testing.T) {
	svc := NewNewsService(config.NewsConfig{MaxItems: 5})
	items := svc.GetCached()
	if len(items) == 0 {
		t.Error("expected mock news when cache empty")
	}
}

func TestNewsGetCachedAfterFetch(t *testing.T) {
	svc := NewNewsService(config.NewsConfig{MaxItems: 5})
	svc.Fetch()
	cached := svc.GetCached()
	if len(cached) == 0 {
		t.Error("expected cached news items")
	}
}

func TestNewsMockDataContent(t *testing.T) {
	svc := NewNewsService(config.NewsConfig{MaxItems: 5})
	items := svc.mockNews()

	if len(items) != 5 {
		t.Errorf("expected 5 mock news items, got %d", len(items))
	}

	for i, item := range items {
		if item.Title == "" {
			t.Errorf("item %d: expected non-empty title", i)
		}
		if item.Description == "" {
			t.Errorf("item %d: expected non-empty description", i)
		}
	}
}
