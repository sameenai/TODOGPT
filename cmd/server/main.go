package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/todogpt/daily-briefing/internal/api"
	"github.com/todogpt/daily-briefing/internal/config"
	"github.com/todogpt/daily-briefing/internal/services"
)

func main() {
	configPath := flag.String("config", "", "Path to config file")
	port := flag.Int("port", 0, "Override server port")
	initConfig := flag.Bool("init", false, "Generate default config file")
	flag.Parse()

	if *initConfig {
		cfg := config.DefaultConfig()
		if err := cfg.Save(*configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Default config saved. Edit ~/.daily-briefing/config.json to add your API keys.")
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	if *port > 0 {
		cfg.Server.Port = *port
	}

	hub := services.NewHub(cfg)

	// Start background polling for real-time updates
	go hub.StartPolling()

	server := api.NewServer(hub, cfg.Server.Host, cfg.Server.Port)

	log.Printf("Starting Daily Briefing Dashboard...")
	log.Printf("Open http://%s:%d in your browser", cfg.Server.Host, cfg.Server.Port)

	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
