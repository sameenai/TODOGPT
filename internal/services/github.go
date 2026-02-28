package services

import (
	"sync"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

type GitHubService struct {
	cfg   config.GitHubConfig
	cache []models.GitHubNotification
	mu    sync.RWMutex
}

func NewGitHubService(cfg config.GitHubConfig) *GitHubService {
	return &GitHubService{cfg: cfg}
}

func (s *GitHubService) Fetch() ([]models.GitHubNotification, error) {
	// When a GitHub token is configured, this would use the GitHub API.
	notifs := s.mockNotifications()
	s.mu.Lock()
	s.cache = notifs
	s.mu.Unlock()
	return notifs, nil
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
