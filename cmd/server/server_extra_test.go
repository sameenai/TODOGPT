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

// TestRunStartsServerDefaultPort starts the server without --port and uses
// the port written into the config file, covering the *port == 0 branch.
func TestRunStartsServerDefaultPort(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	// Pick a free port and bake it into the config
	port := freePort(t)

	cfg := map[string]interface{}{
		"server": map[string]interface{}{
			"port":                  port,
			"host":                  "127.0.0.1",
			"poll_interval_seconds": 300,
		},
	}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	go func() {
		_ = run([]string{"--config", path})
	}()

	addr := fmt.Sprintf("http://127.0.0.1:%d/api/weather", port)
	var err error
	for i := 0; i < 20; i++ {
		var resp *http.Response
		resp, err = http.Get(addr)
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("server not ready: %v", err)
	}
}
