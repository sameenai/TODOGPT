package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// applyEnvOverrides merges standard environment variables into cfg.
// Env vars take precedence over the config file and auto-enable the
// corresponding integration when a credential is supplied.
//
// Supported variables:
//
//	GITHUB_TOKEN          → github.token  (enables GitHub)
//	ANTHROPIC_API_KEY     → ai.api_key    (enables AI summary)
//	SLACK_BOT_TOKEN       → slack.bot_token
//	NOTION_TOKEN          → notion.token
//	JIRA_API_TOKEN        → jira.api_token
//	GOOGLE_CLIENT_ID      → google.client_id  (enables Google OAuth)
//	GOOGLE_CLIENT_SECRET  → google.client_secret
//	ICAL_URL              → google.ical_url
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("GITHUB_TOKEN"); v != "" {
		cfg.GitHub.Token = v
		cfg.GitHub.Enabled = true
	}
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		cfg.AI.APIKey = v
		cfg.AI.Enabled = true
	}
	if v := os.Getenv("SLACK_BOT_TOKEN"); v != "" {
		cfg.Slack.BotToken = v
	}
	if v := os.Getenv("NOTION_TOKEN"); v != "" {
		cfg.Notion.Token = v
	}
	if v := os.Getenv("JIRA_API_TOKEN"); v != "" {
		cfg.Jira.Token = v
	}
	if v := os.Getenv("GOOGLE_CLIENT_ID"); v != "" {
		cfg.Google.ClientID = v
	}
	if v := os.Getenv("GOOGLE_CLIENT_SECRET"); v != "" {
		cfg.Google.ClientSecret = v
	}
	if v := os.Getenv("ICAL_URL"); v != "" {
		cfg.Google.ICalURL = v
	}
}

type Config struct {
	Server   ServerConfig   `json:"server"`
	Weather  WeatherConfig  `json:"weather"`
	News     NewsConfig     `json:"news"`
	Google   GoogleConfig   `json:"google"`
	Slack    SlackConfig    `json:"slack"`
	Email    EmailConfig    `json:"email"`
	GitHub   GitHubConfig   `json:"github"`
	Jira     JiraConfig     `json:"jira"`
	Notion   NotionConfig   `json:"notion"`
	AI       AIConfig       `json:"ai"`
	Pomodoro PomodoroConfig `json:"pomodoro"`
}

type ServerConfig struct {
	Port         int    `json:"port"`
	Host         string `json:"host"`
	PollInterval int    `json:"poll_interval_seconds"`
	DataDir      string `json:"data_dir,omitempty"` // overrides ~/.daily-briefing for all persistent data
}

type WeatherConfig struct {
	APIKey  string  `json:"api_key,omitempty"`
	City    string  `json:"city"`
	Country string  `json:"country"`
	Units   string  `json:"units"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Enabled bool    `json:"enabled"`
}

type NewsConfig struct {
	APIKey     string   `json:"api_key,omitempty"`
	Sources    []string `json:"sources"`
	Country    string   `json:"country"`
	Categories []string `json:"categories"`
	MaxItems   int      `json:"max_items"`
	Enabled    bool     `json:"enabled"`
}

type GoogleConfig struct {
	// OAuth2 credentials — set these to enable browser SSO.
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`

	// ICalURL is a private iCalendar subscription URL (works with Google Calendar,
	// iCloud, Outlook — no OAuth required). When set, real calendar data is fetched.
	ICalURL string `json:"ical_url,omitempty"`
}

type SlackConfig struct {
	BotToken string   `json:"bot_token,omitempty"`
	Channels []string `json:"channels"`
	Enabled  bool     `json:"enabled"`
}

type EmailConfig struct {
	IMAPServer string `json:"imap_server"`
	IMAPPort   int    `json:"imap_port"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	Enabled    bool   `json:"enabled"`
}

type GitHubConfig struct {
	Token   string   `json:"token,omitempty"`
	Repos   []string `json:"repos"`
	Enabled bool     `json:"enabled"`
}

type JiraConfig struct {
	BaseURL string `json:"base_url"`
	Email   string `json:"email"`
	Token   string `json:"api_token,omitempty"`
	Project string `json:"project_key"`
	Enabled bool   `json:"enabled"`
}

type NotionConfig struct {
	Token      string `json:"token,omitempty"`
	DatabaseID string `json:"database_id"`
	Enabled    bool   `json:"enabled"`
}

type AIConfig struct {
	APIKey  string `json:"api_key,omitempty"`
	Model   string `json:"model"`
	Enabled bool   `json:"enabled"`
}

type PomodoroConfig struct {
	WorkMinutes  int  `json:"work_minutes"`
	BreakMinutes int  `json:"break_minutes"`
	Enabled      bool `json:"enabled"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         8080,
			Host:         "localhost",
			PollInterval: 30,
		},
		Weather: WeatherConfig{
			City:    "New York",
			Units:   "imperial",
			Enabled: true,
		},
		News: NewsConfig{
			Sources:  []string{"techcrunch", "hacker-news", "the-verge"},
			MaxItems: 10,
			Enabled:  true,
		},
		Google: GoogleConfig{},
		Slack: SlackConfig{
			Enabled: false,
		},
		Email: EmailConfig{
			IMAPPort: 993,
			Enabled:  false,
		},
		GitHub: GitHubConfig{
			Enabled: false,
		},
		Jira: JiraConfig{
			Enabled: false,
		},
		Notion: NotionConfig{
			Enabled: false,
		},
		AI: AIConfig{
			Model:   "claude-sonnet-4-6",
			Enabled: false,
		},
		Pomodoro: PomodoroConfig{
			WorkMinutes:  25,
			BreakMinutes: 5,
			Enabled:      true,
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return cfg, nil
		}
		path = filepath.Join(home, ".daily-briefing", "config.json")
	}

	data, err := os.ReadFile(path) // #nosec G304 -- path is user-supplied config file location
	if err != nil {
		if os.IsNotExist(err) {
			applyEnvOverrides(cfg)
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	applyEnvOverrides(cfg)
	return cfg, nil
}

func (c *Config) Save(path string) error {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dir := filepath.Join(home, ".daily-briefing")
		if err := os.MkdirAll(dir, 0750); err != nil {
			return err
		}
		path = filepath.Join(dir, "config.json")
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
