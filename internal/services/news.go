package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

type NewsService struct {
	cfg   config.NewsConfig
	cache []models.NewsItem
	mu    sync.RWMutex
}

func NewNewsService(cfg config.NewsConfig) *NewsService {
	return &NewsService{cfg: cfg}
}

// hnItem is a Hacker News story item.
type hnItem struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
	By    string `json:"by"`
	Score int    `json:"score"`
	Time  int64  `json:"time"`
	Type  string `json:"type"`
	Text  string `json:"text"`
}

func (s *NewsService) Fetch() ([]models.NewsItem, error) {
	maxItems := s.cfg.MaxItems
	if maxItems <= 0 {
		maxItems = 10
	}

	items, err := s.fetchHackerNews(maxItems)
	if err != nil {
		// Fall back to mock data on error
		items = s.mockNews()
	}

	s.mu.Lock()
	s.cache = items
	s.mu.Unlock()

	return items, nil
}

func (s *NewsService) fetchHackerNews(maxItems int) ([]models.NewsItem, error) {
	// Fetch top story IDs
	resp, err := http.Get("https://hacker-news.firebaseio.com/v0/topstories.json")
	if err != nil {
		return nil, fmt.Errorf("HN top stories error: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HN API returned status %d", resp.StatusCode)
	}

	var storyIDs []int
	if err := json.NewDecoder(resp.Body).Decode(&storyIDs); err != nil {
		return nil, fmt.Errorf("HN decode error: %w", err)
	}

	// Limit to maxItems
	if len(storyIDs) > maxItems {
		storyIDs = storyIDs[:maxItems]
	}

	// Fetch each story concurrently
	type result struct {
		index int
		item  *hnItem
	}

	results := make(chan result, len(storyIDs))
	var wg sync.WaitGroup

	for i, id := range storyIDs {
		wg.Add(1)
		go func(idx, storyID int) {
			defer wg.Done()
			url := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", storyID)
			r, err := http.Get(url)
			if err != nil {
				return
			}
			defer func() { _ = r.Body.Close() }()

			var item hnItem
			if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
				return
			}
			results <- result{index: idx, item: &item}
		}(i, id)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	itemsByIndex := make(map[int]*hnItem)
	for r := range results {
		itemsByIndex[r.index] = r.item
	}

	// Build ordered news items
	var items []models.NewsItem
	for i := 0; i < len(storyIDs); i++ {
		hn, ok := itemsByIndex[i]
		if !ok || hn == nil {
			continue
		}

		articleURL := hn.URL
		if articleURL == "" {
			articleURL = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", hn.ID)
		}

		desc := hn.Text
		if desc == "" {
			desc = fmt.Sprintf("%d points by %s", hn.Score, hn.By)
		}
		// Truncate HTML text from HN
		if len(desc) > 200 {
			desc = desc[:197] + "..."
		}

		items = append(items, models.NewsItem{
			Title:       hn.Title,
			Description: desc,
			URL:         articleURL,
			Source:      "Hacker News",
			PublishedAt: time.Unix(hn.Time, 0),
		})
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no stories fetched from HN")
	}

	return items, nil
}

func (s *NewsService) GetCached() []models.NewsItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache != nil {
		return s.cache
	}
	return s.mockNews()
}

func (s *NewsService) mockNews() []models.NewsItem {
	now := time.Now()
	return []models.NewsItem{
		{Title: "Go 1.23 Released with Major Performance Improvements", Description: "The latest Go release brings significant improvements to compilation speed and runtime performance.", URL: "#", Source: "Hacker News", PublishedAt: now},
		{Title: "AI-Powered Developer Tools See Record Adoption", Description: "New survey shows 78% of developers now use AI-assisted coding tools daily.", URL: "#", Source: "Hacker News", PublishedAt: now.Add(-1 * time.Hour)},
		{Title: "Cloud Infrastructure Costs Drop 30% with New Optimization", Description: "Major cloud providers announce new cost-saving tiers for startup workloads.", URL: "#", Source: "Hacker News", PublishedAt: now.Add(-2 * time.Hour)},
		{Title: "Remote Work Trends Continue to Shape Tech Industry", Description: "Latest data shows distributed teams outperforming co-located counterparts.", URL: "#", Source: "Hacker News", PublishedAt: now.Add(-3 * time.Hour)},
		{Title: "Open Source Security Initiative Launches", Description: "Major tech companies band together to fund critical open-source security audits.", URL: "#", Source: "Hacker News", PublishedAt: now.Add(-4 * time.Hour)},
	}
}
