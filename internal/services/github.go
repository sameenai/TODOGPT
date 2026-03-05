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

// githubAPIBaseURL is package-level so tests can override it.
var githubAPIBaseURL = "https://api.github.com"

type GitHubService struct {
	cfg   config.GitHubConfig
	cache []models.GitHubNotification
	mu    sync.RWMutex
}

func NewGitHubService(cfg config.GitHubConfig) *GitHubService {
	return &GitHubService{cfg: cfg}
}

// IsLive returns true when a GitHub token is configured and the integration is enabled.
func (s *GitHubService) IsLive() bool { return s.cfg.Enabled && s.cfg.Token != "" }

// githubNotification is the raw API response shape from GET /notifications.
type githubNotification struct {
	ID   string `json:"id"`
	Repo struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Subject struct {
		Title string `json:"title"`
		URL   string `json:"url"`
		Type  string `json:"type"`
	} `json:"subject"`
	Reason    string `json:"reason"`
	Unread    bool   `json:"unread"`
	UpdatedAt string `json:"updated_at"`
}

func (s *GitHubService) Fetch() ([]models.GitHubNotification, error) {
	if !s.cfg.Enabled || s.cfg.Token == "" {
		notifs := s.mockNotifications()
		s.mu.Lock()
		s.cache = notifs
		s.mu.Unlock()
		return notifs, nil
	}

	notifs, err := s.fetchFromAPI()
	if err != nil {
		// Fall back to cache or mock on API error
		s.mu.RLock()
		cached := s.cache
		s.mu.RUnlock()
		if cached != nil {
			return cached, nil
		}
		return s.mockNotifications(), nil
	}

	s.mu.Lock()
	s.cache = notifs
	s.mu.Unlock()
	return notifs, nil
}

func (s *GitHubService) fetchFromAPI() ([]models.GitHubNotification, error) {
	url := githubAPIBaseURL + "/notifications?all=false&per_page=50"
	req, err := http.NewRequest(http.MethodGet, url, nil) // #nosec G107
	if err != nil {
		return nil, fmt.Errorf("github: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github: API returned status %d", resp.StatusCode)
	}

	var raw []githubNotification
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("github: decode error: %w", err)
	}

	notifs := make([]models.GitHubNotification, 0, len(raw))
	for _, n := range raw {
		updatedAt, _ := time.Parse(time.RFC3339, n.UpdatedAt)
		notif := models.GitHubNotification{
			ID:        n.ID,
			Title:     n.Subject.Title,
			Repo:      n.Repo.FullName,
			Type:      n.Subject.Type,
			URL:       n.Subject.URL,
			Reason:    n.Reason,
			Unread:    n.Unread,
			UpdatedAt: updatedAt,
		}
		// Filter to configured repos if specified
		if len(s.cfg.Repos) > 0 && !containsRepo(s.cfg.Repos, n.Repo.FullName) {
			continue
		}
		notifs = append(notifs, notif)
	}

	return notifs, nil
}

func containsRepo(repos []string, repo string) bool {
	for _, r := range repos {
		if r == repo {
			return true
		}
	}
	return false
}

func (s *GitHubService) GetCached() []models.GitHubNotification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache != nil {
		return s.cache
	}
	return s.mockNotifications()
}

func (s *GitHubService) mockNotifications() []models.GitHubNotification {
	now := time.Now()
	return []models.GitHubNotification{
		{
			ID:        "gh-1",
			Title:     "Fix memory leak in connection pool",
			Repo:      "myorg/backend",
			Type:      "PullRequest",
			Reason:    "review_requested",
			Unread:    true,
			UpdatedAt: now.Add(-30 * time.Minute),
		},
		{
			ID:        "gh-2",
			Title:     "Add rate limiting middleware",
			Repo:      "myorg/api-gateway",
			Type:      "PullRequest",
			Reason:    "author",
			Unread:    true,
			UpdatedAt: now.Add(-1 * time.Hour),
		},
		{
			ID:        "gh-3",
			Title:     "CI failing on main branch",
			Repo:      "myorg/frontend",
			Type:      "Issue",
			Reason:    "mention",
			Unread:    true,
			UpdatedAt: now.Add(-2 * time.Hour),
		},
		{
			ID:        "gh-4",
			Title:     "Upgrade dependencies to fix CVE-2024-1234",
			Repo:      "myorg/backend",
			Type:      "Issue",
			Reason:    "assign",
			Unread:    false,
			UpdatedAt: now.Add(-5 * time.Hour),
		},
	}
}
