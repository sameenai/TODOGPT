package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

type newsAPIResponse struct {
	Articles []struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		URL         string `json:"url"`
		Source      struct {
			Name string `json:"name"`
		} `json:"source"`
		PublishedAt string `json:"publishedAt"`
		URLToImage  string `json:"urlToImage"`
	} `json:"articles"`
}

func (s *NewsService) Fetch() ([]models.NewsItem, error) {
	if s.cfg.APIKey == "" {
		items := s.mockNews()
		s.mu.Lock()
		s.cache = items
		s.mu.Unlock()
		return items, nil
	}

	var url string
	if len(s.cfg.Sources) > 0 {
		url = fmt.Sprintf(
			"https://newsapi.org/v2/top-headlines?sources=%s&pageSize=%d&apiKey=%s",
			strings.Join(s.cfg.Sources, ","), s.cfg.MaxItems, s.cfg.APIKey,
		)
	} else if s.cfg.Country != "" {
		url = fmt.Sprintf(
			"https://newsapi.org/v2/top-headlines?country=%s&pageSize=%d&apiKey=%s",
			s.cfg.Country, s.cfg.MaxItems, s.cfg.APIKey,
		)
		if len(s.cfg.Categories) > 0 {
			url += "&category=" + s.cfg.Categories[0]
		}
	} else {
		url = fmt.Sprintf(
			"https://newsapi.org/v2/top-headlines?pageSize=%d&apiKey=%s",
			s.cfg.MaxItems, s.cfg.APIKey,
		)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("news API error: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var apiResp newsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("news decode error: %w", err)
	}

	var items []models.NewsItem
	for _, a := range apiResp.Articles {
		pubTime, err := time.Parse(time.RFC3339, a.PublishedAt)
		if err != nil {
			pubTime = time.Now()
		}
		items = append(items, models.NewsItem{
			Title:       a.Title,
			Description: a.Description,
			URL:         a.URL,
			Source:      a.Source.Name,
			PublishedAt: pubTime,
			ImageURL:    a.URLToImage,
		})
	}

	s.mu.Lock()
	s.cache = items
	s.mu.Unlock()

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
		{Title: "Go 1.23 Released with Major Performance Improvements", Description: "The latest Go release brings significant improvements to compilation speed and runtime performance.", URL: "#", Source: "Tech News", PublishedAt: now},
		{Title: "AI-Powered Developer Tools See Record Adoption", Description: "New survey shows 78% of developers now use AI-assisted coding tools daily.", URL: "#", Source: "Dev Weekly", PublishedAt: now.Add(-1 * time.Hour)},
		{Title: "Cloud Infrastructure Costs Drop 30% with New Optimization", Description: "Major cloud providers announce new cost-saving tiers for startup workloads.", URL: "#", Source: "Cloud Digest", PublishedAt: now.Add(-2 * time.Hour)},
		{Title: "Remote Work Trends Continue to Shape Tech Industry", Description: "Latest data shows distributed teams outperforming co-located counterparts.", URL: "#", Source: "Work Future", PublishedAt: now.Add(-3 * time.Hour)},
		{Title: "Open Source Security Initiative Launches", Description: "Major tech companies band together to fund critical open-source security audits.", URL: "#", Source: "Security Wire", PublishedAt: now.Add(-4 * time.Hour)},
	}
}
