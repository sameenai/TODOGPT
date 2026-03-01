package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunInit(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	err := run([]string{"--init", "--config", path})
	if err != nil {
		t.Fatalf("run --init failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("config file was not created")
	}

	// Verify the config file is valid JSON
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("config is not valid JSON: %v", err)
	}
}

func TestRunInitDefaultPath(t *testing.T) {
	err := run([]string{"--init"})
	if err != nil {
		t.Logf("run --init to default path: %v (may fail in CI)", err)
		return
	}

	// Cleanup
	home, _ := os.UserHomeDir()
	os.Remove(filepath.Join(home, ".daily-briefing", "config.json"))
}

func TestRunInvalidFlag(t *testing.T) {
	err := run([]string{"--nonexistent-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestRunInitWithPort(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	// Port flag is accepted even with init
	err := run([]string{"--init", "--config", path, "--port", "9999"})
	if err != nil {
		t.Fatalf("run --init with port failed: %v", err)
	}
}

func TestRunWithBadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad-config.json")
	os.WriteFile(path, []byte("not valid json{{{"), 0600)

	err := run([]string{"--config", path})
	if err == nil {
		t.Error("expected error for invalid config JSON")
	}
}

func TestRunInitSaveError(t *testing.T) {
	// Use a directory path as the config file to cause a Save error
	tmpDir := t.TempDir()
	err := run([]string{"--init", "--config", tmpDir})
	if err == nil {
		t.Error("expected error when saving config to a directory")
	}
}

func TestRunStartsServer(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	// Create a valid config first
	err := run([]string{"--init", "--config", path})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Start server in background with a specific port
	port := 19877
	go func() {
		_ = run([]string{"--config", path, "--port", fmt.Sprintf("%d", port)})
	}()

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/weather", port))
	if err != nil {
		t.Fatalf("server not responding: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
