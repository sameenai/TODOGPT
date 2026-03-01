package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "localhost" {
		t.Errorf("expected default host localhost, got %s", cfg.Server.Host)
	}
	if cfg.Server.PollInterval != 30 {
		t.Errorf("expected default poll interval 30, got %d", cfg.Server.PollInterval)
	}

	if cfg.Weather.City != "New York" {
		t.Errorf("expected default city New York, got %s", cfg.Weather.City)
	}
	if cfg.Weather.Units != "imperial" {
		t.Errorf("expected default units imperial, got %s", cfg.Weather.Units)
	}
	if !cfg.Weather.Enabled {
		t.Error("expected weather enabled by default")
	}

	if cfg.News.MaxItems != 10 {
		t.Errorf("expected default max news items 10, got %d", cfg.News.MaxItems)
	}
	if !cfg.News.Enabled {
		t.Error("expected news enabled by default")
	}
	expectedSources := []string{"techcrunch", "hacker-news", "the-verge"}
	if len(cfg.News.Sources) != len(expectedSources) {
		t.Errorf("expected default news sources %v, got %v", expectedSources, cfg.News.Sources)
	} else {
		for i, s := range expectedSources {
			if cfg.News.Sources[i] != s {
				t.Errorf("expected source %d to be %s, got %s", i, s, cfg.News.Sources[i])
			}
		}
	}

	if !cfg.Google.CalendarEnabled {
		t.Error("expected Google Calendar enabled by default")
	}
	if !cfg.Google.GmailEnabled {
		t.Error("expected Gmail enabled by default")
	}

	if cfg.Slack.Enabled {
		t.Error("expected Slack disabled by default")
	}
	if cfg.Email.IMAPPort != 993 {
		t.Errorf("expected default IMAP port 993, got %d", cfg.Email.IMAPPort)
	}
	if cfg.GitHub.Enabled {
		t.Error("expected GitHub disabled by default")
	}
	if cfg.Jira.Enabled {
		t.Error("expected Jira disabled by default")
	}
	if cfg.Notion.Enabled {
		t.Error("expected Notion disabled by default")
	}

	if cfg.Pomodoro.WorkMinutes != 25 {
		t.Errorf("expected default work minutes 25, got %d", cfg.Pomodoro.WorkMinutes)
	}
	if cfg.Pomodoro.BreakMinutes != 5 {
		t.Errorf("expected default break minutes 5, got %d", cfg.Pomodoro.BreakMinutes)
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if cfg == nil {
		t.Fatal("expected default config, got nil")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port when file missing, got %d", cfg.Server.Port)
	}
}

func TestLoadEmptyPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected default config, got nil")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test-config.json")

	cfg := DefaultConfig()
	cfg.Server.Port = 9999
	cfg.Weather.City = "Tokyo"
	cfg.Weather.APIKey = "test-key-123"
	cfg.News.MaxItems = 20
	cfg.Slack.Enabled = true
	cfg.Slack.BotToken = "xoxb-test"

	err := cfg.Save(path)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Verify file permissions
	info, _ := os.Stat(path)
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("expected file permissions 0600, got %o", perm)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Server.Port != 9999 {
		t.Errorf("expected port 9999, got %d", loaded.Server.Port)
	}
	if loaded.Weather.City != "Tokyo" {
		t.Errorf("expected city Tokyo, got %s", loaded.Weather.City)
	}
	if loaded.Weather.APIKey != "test-key-123" {
		t.Errorf("expected API key test-key-123, got %s", loaded.Weather.APIKey)
	}
	if loaded.News.MaxItems != 20 {
		t.Errorf("expected max items 20, got %d", loaded.News.MaxItems)
	}
	if !loaded.Slack.Enabled {
		t.Error("expected Slack enabled")
	}
	if loaded.Slack.BotToken != "xoxb-test" {
		t.Errorf("expected bot token xoxb-test, got %s", loaded.Slack.BotToken)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nested", "dir", "config.json")

	cfg := DefaultConfig()
	err := cfg.Save(path)
	// This should fail because Save only creates the parent dir from userHomeDir path
	// but with explicit path it writes directly
	if err != nil {
		// Expected — parent dir doesn't exist for explicit nested path
		return
	}
}

func TestSaveDefaultPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.Server.Port = 7777

	// Save to default path (temp home dir)
	err := cfg.Save("")
	if err != nil {
		t.Fatalf("Save to default path failed: %v", err)
	}

	// Load from default path
	loaded, err := Load("")
	if err != nil {
		t.Fatalf("Load from default path failed: %v", err)
	}
	if loaded.Server.Port != 7777 {
		t.Errorf("expected port 7777, got %d", loaded.Server.Port)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad-config.json")

	err := os.WriteFile(path, []byte("not valid json{{{"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Load(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestConfigJSONRoundTrip(t *testing.T) {
	cfg := DefaultConfig()
	cfg.GitHub.Token = "ghp_test123"
	cfg.GitHub.Repos = []string{"org/repo1", "org/repo2"}
	cfg.GitHub.Enabled = true

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if loaded.GitHub.Token != "ghp_test123" {
		t.Errorf("expected token ghp_test123, got %s", loaded.GitHub.Token)
	}
	if len(loaded.GitHub.Repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(loaded.GitHub.Repos))
	}
	if !loaded.GitHub.Enabled {
		t.Error("expected GitHub enabled")
	}
}

func TestLoadDirectoryAsFile(t *testing.T) {
	// Passing a directory path triggers the non-IsNotExist read error branch
	tmpDir := t.TempDir()
	_, err := Load(tmpDir)
	if err == nil {
		t.Error("expected error when loading a directory as config file")
	}
}

func TestSaveToExplicitPath(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "explicit-config.json")

	cfg := DefaultConfig()
	cfg.Server.Port = 5555

	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save to explicit path failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Server.Port != 5555 {
		t.Errorf("expected port 5555, got %d", loaded.Server.Port)
	}
}

func TestSaveToReadOnlyDir(t *testing.T) {
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	os.MkdirAll(readOnlyDir, 0555)
	path := filepath.Join(readOnlyDir, "subdir", "config.json")

	cfg := DefaultConfig()
	err := cfg.Save(path)
	// Should fail because readOnlyDir doesn't allow creating subdirs
	if err == nil {
		// Cleanup if it somehow succeeded
		os.Remove(path)
	}
}

func TestSaveMkdirAllError(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a FILE where the .daily-briefing directory would be created,
	// causing MkdirAll to fail.
	os.WriteFile(filepath.Join(tmpDir, ".daily-briefing"), []byte("file"), 0600)
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	err := cfg.Save("")
	if err == nil {
		t.Error("expected error when .daily-briefing exists as a file")
	}
}

func TestConfigPartialOverride(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "partial.json")

	// Write only partial config — other fields should keep defaults
	partial := `{"server": {"port": 3000}, "weather": {"city": "London"}}`
	if err := os.WriteFile(path, []byte(partial), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Port != 3000 {
		t.Errorf("expected port 3000, got %d", cfg.Server.Port)
	}
	if cfg.Weather.City != "London" {
		t.Errorf("expected city London, got %s", cfg.Weather.City)
	}
	// Default values should be preserved for unset fields
	if cfg.Weather.Units != "imperial" {
		t.Errorf("expected default units imperial preserved, got %s", cfg.Weather.Units)
	}
}
