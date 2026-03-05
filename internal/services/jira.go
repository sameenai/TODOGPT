package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

// jiraAPIPath is the search endpoint appended to cfg.BaseURL; overridable in tests.
var jiraAPIPath = "/rest/api/3/search"

// JiraService fetches assigned Jira tickets from the REST API.
type JiraService struct {
	cfg   config.JiraConfig
	cache []models.JiraTicket
	mu    sync.RWMutex
}

func NewJiraService(cfg config.JiraConfig) *JiraService {
	return &JiraService{cfg: cfg}
}

// IsLive returns true when Jira credentials are fully configured.
func (s *JiraService) IsLive() bool {
	return s.cfg.Enabled && s.cfg.BaseURL != "" && s.cfg.Token != ""
}

func (s *JiraService) Fetch() ([]models.JiraTicket, error) {
	if !s.cfg.Enabled || s.cfg.BaseURL == "" || s.cfg.Token == "" {
		return s.mockTickets(), nil
	}
	tickets, err := s.fetchFromAPI()
	if err != nil {
		cached := s.GetCached()
		if len(cached) > 0 {
			return cached, nil
		}
		return s.mockTickets(), nil
	}
	s.mu.Lock()
	s.cache = tickets
	s.mu.Unlock()
	return tickets, nil
}

func (s *JiraService) GetCached() []models.JiraTicket {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache != nil {
		return s.cache
	}
	return s.mockTickets()
}

// jiraSearchResponse is the top-level shape returned by /rest/api/3/search.
type jiraSearchResponse struct {
	Issues []struct {
		Key    string `json:"key"`
		Fields struct {
			Summary string `json:"summary"`
			Status  struct {
				Name string `json:"name"`
			} `json:"status"`
			Priority struct {
				Name string `json:"name"`
			} `json:"priority"`
			Assignee *struct {
				DisplayName string `json:"displayName"`
			} `json:"assignee"`
			DueDate   string `json:"duedate"`
			IssueType struct {
				Name string `json:"name"`
			} `json:"issuetype"`
		} `json:"fields"`
	} `json:"issues"`
}

func (s *JiraService) fetchFromAPI() ([]models.JiraTicket, error) {
	jql := fmt.Sprintf("project=%s AND assignee=currentUser() AND statusCategory != Done ORDER BY priority ASC", s.cfg.Project)
	params := url.Values{}
	params.Set("jql", jql)
	params.Set("fields", "summary,status,priority,assignee,duedate,issuetype")
	params.Set("maxResults", "20")
	apiURL := s.cfg.BaseURL + jiraAPIPath + "?" + params.Encode()

	req, err := http.NewRequest(http.MethodGet, apiURL, nil) // #nosec G107
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.cfg.Email, s.cfg.Token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // #nosec G307

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jira API returned %d", resp.StatusCode)
	}

	var raw jiraSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	tickets := make([]models.JiraTicket, 0, len(raw.Issues))
	for _, issue := range raw.Issues {
		t := models.JiraTicket{
			Key:      issue.Key,
			Summary:  issue.Fields.Summary,
			Status:   issue.Fields.Status.Name,
			Priority: issue.Fields.Priority.Name,
			Type:     issue.Fields.IssueType.Name,
			URL:      s.cfg.BaseURL + "/browse/" + issue.Key,
		}
		if issue.Fields.Assignee != nil {
			t.Assignee = issue.Fields.Assignee.DisplayName
		}
		if issue.Fields.DueDate != "" {
			if due, err := time.Parse("2006-01-02", issue.Fields.DueDate); err == nil {
				t.DueDate = due
			}
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

func (s *JiraService) mockTickets() []models.JiraTicket {
	now := time.Now()
	due := now.AddDate(0, 0, 3)
	return []models.JiraTicket{
		{
			Key:      "PROJ-101",
			Summary:  "Implement user authentication flow",
			Status:   "In Progress",
			Priority: "High",
			Assignee: "You",
			DueDate:  due,
			URL:      "https://jira.example.com/browse/PROJ-101",
			Type:     "Story",
		},
		{
			Key:      "PROJ-98",
			Summary:  "Fix null pointer in payment service",
			Status:   "To Do",
			Priority: "Critical",
			Assignee: "You",
			DueDate:  now.AddDate(0, 0, 1),
			URL:      "https://jira.example.com/browse/PROJ-98",
			Type:     "Bug",
		},
		{
			Key:      "PROJ-87",
			Summary:  "Write unit tests for order processor",
			Status:   "To Do",
			Priority: "Medium",
			Assignee: "You",
			URL:      "https://jira.example.com/browse/PROJ-87",
			Type:     "Task",
		},
	}
}
