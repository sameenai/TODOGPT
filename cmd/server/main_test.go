package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not get free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

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
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	err := run([]string{"--init"})
	if err != nil {
		t.Fatalf("run --init to default path failed: %v", err)
	}

	// Verify config was created in the temp home
	path := filepath.Join(tmpDir, ".daily-briefing", "config.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("config file was not created at default path")
	}
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

	// Get a free port to avoid conflicts
	port := freePort(t)
	go func() {
		_ = run([]string{"--config", path, "--port", fmt.Sprintf("%d", port)})
	}()

	// Poll until server is ready instead of fixed sleep
	addr := fmt.Sprintf("http://127.0.0.1:%d/api/weather", port)
	var resp *http.Response
	for i := 0; i < 20; i++ {
		resp, err = http.Get(addr)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("server not responding after retries: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
