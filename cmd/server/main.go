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
	if err := run(os.Args[1:]); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	configPath := fs.String("config", "", "Path to config file")
	port := fs.Int("port", 0, "Override server port")
	initConfig := fs.Bool("init", false, "Generate default config file")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *initConfig {
		cfg := config.DefaultConfig()
		if err := cfg.Save(*configPath); err != nil {
			return fmt.Errorf("error saving config: %w", err)
		}
		fmt.Println("Default config saved. Edit ~/.daily-briefing/config.json to add your API keys.")
		return nil
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("config error: %w", err)
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

	return server.Start()
}
