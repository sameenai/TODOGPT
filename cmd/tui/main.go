package main

import (
	"fmt"
	"os"

	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/services"
	"github.com/todogpt/daily-briefing/internal/tui"
)

func main() {
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	hub := services.NewHub(cfg)
	if err := tui.Run(hub, cfg.Pomodoro); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
