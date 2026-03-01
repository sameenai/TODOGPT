package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

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
	Pomodoro PomodoroConfig `json:"pomodoro"`
}

type ServerConfig struct {
	Port         int    `json:"port"`
	Host         string `json:"host"`
	PollInterval int    `json:"poll_interval_seconds"`
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
	CredentialsFile string `json:"credentials_file"`
	TokenFile       string `json:"token_file"`
	CalendarEnabled bool   `json:"calendar_enabled"`
	GmailEnabled    bool   `json:"gmail_enabled"`
}

type SlackConfig struct {
	BotToken string   `json:"bot_token"`
	AppToken string   `json:"app_token"`
	Channels []string `json:"channels"`
	Enabled  bool     `json:"enabled"`
}

type EmailConfig struct {
	IMAPServer string `json:"imap_server"`
	IMAPPort   int    `json:"imap_port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Enabled    bool   `json:"enabled"`
}

type GitHubConfig struct {
	Token   string   `json:"token"`
	Repos   []string `json:"repos"`
	Enabled bool     `json:"enabled"`
}

type JiraConfig struct {
	BaseURL string `json:"base_url"`
	Email   string `json:"email"`
	Token   string `json:"api_token"`
	Project string `json:"project_key"`
	Enabled bool   `json:"enabled"`
}

type NotionConfig struct {
	Token      string `json:"token"`
	DatabaseID string `json:"database_id"`
	Enabled    bool   `json:"enabled"`
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
		Google: GoogleConfig{
			CredentialsFile: "credentials.json",
			TokenFile:       "token.json",
			CalendarEnabled: true,
			GmailEnabled:    true,
		},
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

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Save(path string) error {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dir := filepath.Join(home, ".daily-briefing")
		if err := os.MkdirAll(dir, 0755); err != nil {
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
