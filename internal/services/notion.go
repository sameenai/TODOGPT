package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

// notionAPIBaseURL is overridable in tests.
var notionAPIBaseURL = "https://api.notion.com"

// NotionService fetches pages from a configured Notion database.
type NotionService struct {
	cfg   config.NotionConfig
	cache []models.NotionPage
	mu    sync.RWMutex
}

func NewNotionService(cfg config.NotionConfig) *NotionService {
	return &NotionService{cfg: cfg}
}

// IsLive returns true when a Notion token and database ID are configured.
func (s *NotionService) IsLive() bool {
	return s.cfg.Enabled && s.cfg.Token != "" && s.cfg.DatabaseID != ""
}

func (s *NotionService) Fetch() ([]models.NotionPage, error) {
	if !s.cfg.Enabled || s.cfg.Token == "" || s.cfg.DatabaseID == "" {
		return s.mockPages(), nil
	}
	pages, err := s.fetchFromAPI()
	if err != nil {
		cached := s.GetCached()
		if len(cached) > 0 {
			return cached, nil
		}
		return s.mockPages(), nil
	}
	s.mu.Lock()
	s.cache = pages
	s.mu.Unlock()
	return pages, nil
}

func (s *NotionService) GetCached() []models.NotionPage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache != nil {
		return s.cache
	}
	return s.mockPages()
}

// notionQueryResponse is the shape of POST /v1/databases/{id}/query.
type notionQueryResponse struct {
	Results []struct {
		ID             string `json:"id"`
		URL            string `json:"url"`
		LastEditedTime string `json:"last_edited_time"`
		Properties     map[string]struct {
			Type  string `json:"type"`
			Title []struct {
				PlainText string `json:"plain_text"`
			} `json:"title"`
			Select *struct {
				Name string `json:"name"`
			} `json:"select"`
			Date *struct {
				Start string `json:"start"`
			} `json:"date"`
		} `json:"properties"`
	} `json:"results"`
}

func (s *NotionService) fetchFromAPI() ([]models.NotionPage, error) {
	url := notionAPIBaseURL + "/v1/databases/" + s.cfg.DatabaseID + "/query"

	// Filter to only incomplete pages
	body := []byte(`{"filter":{"property":"Status","select":{"does_not_equal":"Done"}},"page_size":20}`)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body)) // #nosec G107
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.Token)
	req.Header.Set("Notion-Version", "2022-06-28")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // #nosec G307

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("notion API returned %d", resp.StatusCode)
	}

	var raw notionQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	pages := make([]models.NotionPage, 0, len(raw.Results))
	for _, r := range raw.Results {
		page := models.NotionPage{
			ID:       r.ID,
			URL:      r.URL,
			Database: s.cfg.DatabaseID,
		}
		if t, err := time.Parse(time.RFC3339, r.LastEditedTime); err == nil {
			page.UpdatedAt = t
		}
		for propName, prop := range r.Properties {
			nameLower := strings.ToLower(propName)
			switch prop.Type {
			case "title":
				if len(prop.Title) > 0 {
					page.Title = prop.Title[0].PlainText
				}
			case "select":
				if prop.Select != nil {
					if strings.Contains(nameLower, "status") || nameLower == "state" {
						page.Status = prop.Select.Name
					} else if strings.Contains(nameLower, "priority") {
						page.Priority = prop.Select.Name
					} else if page.Status == "" {
						page.Status = prop.Select.Name
					}
				}
			case "date":
				if prop.Date != nil && prop.Date.Start != "" {
					if due, err := time.Parse("2006-01-02", prop.Date.Start); err == nil {
						page.DueDate = &due
					} else if due, err := time.Parse(time.RFC3339, prop.Date.Start); err == nil {
						page.DueDate = &due
					}
				}
			}
		}
		pages = append(pages, page)
	}
	return pages, nil
}

func (s *NotionService) mockPages() []models.NotionPage {
	now := time.Now()
	due := now.AddDate(0, 0, 2)
	return []models.NotionPage{
		{
			ID:        "notion-page-1",
			Title:     "Q2 Planning doc",
			Status:    "In Progress",
			Priority:  "High",
			DueDate:   &due,
			URL:       "https://notion.so/Q2-Planning",
			Database:  "tasks",
			UpdatedAt: now.Add(-2 * time.Hour),
		},
		{
			ID:        "notion-page-2",
			Title:     "Research competitor pricing",
			Status:    "Not Started",
			Priority:  "Medium",
			URL:       "https://notion.so/Competitor-Pricing",
			Database:  "tasks",
			UpdatedAt: now.Add(-24 * time.Hour),
		},
	}
}
