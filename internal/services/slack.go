package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/models"
)

// slackAPIBase is package-level so tests can override it.
var slackAPIBase = "https://slack.com/api"

// urgentKeywords triggers IsUrgent on a message when any keyword is found.
var urgentKeywords = []string{
	"urgent", "asap", "help", "error", "down", "incident",
	"outage", "critical", "broken", "emergency", "@channel", "@here",
}

type SlackService struct {
	cfg       config.SlackConfig
	cache     []models.SlackMessage
	mu        sync.RWMutex
	userCache map[string]string // userID → display name
}

func NewSlackService(cfg config.SlackConfig) *SlackService {
	return &SlackService{cfg: cfg, userCache: make(map[string]string)}
}

// IsLive returns true when a Slack bot token is configured and the integration is enabled.
func (s *SlackService) IsLive() bool { return s.cfg.Enabled && s.cfg.BotToken != "" }

func (s *SlackService) Fetch() ([]models.SlackMessage, error) {
	if !s.IsLive() {
		msgs := s.mockMessages()
		s.mu.Lock()
		s.cache = msgs
		s.mu.Unlock()
		return msgs, nil
	}

	msgs, err := s.fetchFromAPI()
	if err != nil {
		s.mu.RLock()
		cached := s.cache
		s.mu.RUnlock()
		if cached != nil {
			return cached, nil
		}
		return s.mockMessages(), nil
	}

	s.mu.Lock()
	s.cache = msgs
	s.mu.Unlock()
	return msgs, nil
}

func (s *SlackService) GetCached() []models.SlackMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cache != nil {
		return s.cache
	}
	return s.mockMessages()
}

// ── Slack Web API ─────────────────────────────────────────────────────────────

type slackHistoryResp struct {
	OK       bool          `json:"ok"`
	Messages []slackRawMsg `json:"messages"`
	Error    string        `json:"error"`
}

type slackRawMsg struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	Text    string `json:"text"`
	User    string `json:"user"`
	BotID   string `json:"bot_id"`
	Ts      string `json:"ts"`
}

type slackUserResp struct {
	OK   bool `json:"ok"`
	User struct {
		Name    string `json:"name"`
		Profile struct {
			DisplayName string `json:"display_name"`
			RealName    string `json:"real_name"`
		} `json:"profile"`
	} `json:"user"`
	Error string `json:"error"`
}

type slackChannelResp struct {
	OK      bool `json:"ok"`
	Channel struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		IsIM bool   `json:"is_im"`
	} `json:"channel"`
	Error string `json:"error"`
}

func (s *SlackService) fetchFromAPI() ([]models.SlackMessage, error) {
	if len(s.cfg.Channels) == 0 {
		return []models.SlackMessage{}, nil
	}

	var all []models.SlackMessage
	for _, channelID := range s.cfg.Channels {
		msgs, err := s.fetchChannel(channelID)
		if err != nil {
			// Skip failed channels; don't abort everything
			continue
		}
		all = append(all, msgs...)
	}
	return all, nil
}

func (s *SlackService) fetchChannel(channelID string) ([]models.SlackMessage, error) {
	url := fmt.Sprintf("%s/conversations.history?channel=%s&limit=20", slackAPIBase, channelID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.BotToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("slack API returned %d", resp.StatusCode)
	}

	var parsed slackHistoryResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if !parsed.OK {
		return nil, fmt.Errorf("slack error: %s", parsed.Error)
	}

	// Determine channel name and DM status
	channelName, isDM := s.resolveChannel(channelID)

	var result []models.SlackMessage
	for _, m := range parsed.Messages {
		if m.Type != "message" || m.Subtype == "bot_message" && m.BotID != "" {
			continue
		}
		if m.Text == "" {
			continue
		}

		userName := s.resolveUser(m.User)
		ts := parseSlackTS(m.Ts)

		result = append(result, models.SlackMessage{
			Channel:   "#" + channelName,
			User:      userName,
			Text:      m.Text,
			Timestamp: ts,
			IsUrgent:  isUrgentText(m.Text),
			IsDM:      isDM,
		})
	}
	return result, nil
}

func (s *SlackService) resolveUser(userID string) string {
	if userID == "" {
		return "unknown"
	}
	if name, ok := s.userCache[userID]; ok {
		return name
	}

	url := fmt.Sprintf("%s/users.info?user=%s", slackAPIBase, userID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return userID
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.BotToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return userID
	}
	defer resp.Body.Close()

	var parsed slackUserResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil || !parsed.OK {
		return userID
	}

	name := parsed.User.Profile.DisplayName
	if name == "" {
		name = parsed.User.Profile.RealName
	}
	if name == "" {
		name = parsed.User.Name
	}

	s.userCache[userID] = name
	return name
}

func (s *SlackService) resolveChannel(channelID string) (name string, isDM bool) {
	// DM channel IDs start with "D"
	if strings.HasPrefix(channelID, "D") {
		return channelID, true
	}

	url := fmt.Sprintf("%s/conversations.info?channel=%s", slackAPIBase, channelID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return channelID, false
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.BotToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return channelID, false
	}
	defer resp.Body.Close()

	var parsed slackChannelResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil || !parsed.OK {
		return channelID, false
	}
	return parsed.Channel.Name, parsed.Channel.IsIM
}

// parseSlackTS converts a Slack timestamp string ("1234567890.123456") to time.Time.
func parseSlackTS(ts string) time.Time {
	if ts == "" {
		return time.Now()
	}
	f, err := strconv.ParseFloat(ts, 64)
	if err != nil {
		return time.Now()
	}
	return time.Unix(int64(f), 0)
}

// isUrgentText returns true if the message text contains any urgency signal.
func isUrgentText(text string) bool {
	lower := strings.ToLower(text)
	for _, kw := range urgentKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func (s *SlackService) mockMessages() []models.SlackMessage {
	now := time.Now()
	return []models.SlackMessage{
		{
			Channel:   "#engineering",
			User:      "sarah",
			Text:      "Deployed v2.3.1 to staging. Can someone verify the auth flow?",
			Timestamp: now.Add(-30 * time.Minute),
			IsUrgent:  true,
		},
		{
			Channel:   "#general",
			User:      "mike",
			Text:      "Team lunch moved to 12:30 today",
			Timestamp: now.Add(-1 * time.Hour),
		},
		{
			Channel:   "DM",
			User:      "alex",
			Text:      "Hey, can you review my PR when you get a chance?",
			Timestamp: now.Add(-2 * time.Hour),
			IsDM:      true,
		},
		{
			Channel:   "#incidents",
			User:      "pagerduty-bot",
			Text:      "RESOLVED: API latency spike on us-east-1 has been mitigated",
			Timestamp: now.Add(-3 * time.Hour),
			IsUrgent:  true,
		},
		{
			Channel:   "#random",
			User:      "jordan",
			Text:      "Anyone up for a coffee run at 3pm?",
			Timestamp: now.Add(-4 * time.Hour),
		},
	}
}
