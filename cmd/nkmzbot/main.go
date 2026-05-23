package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/susu3304/nkmzbot/internal/bot"
	"github.com/susu3304/nkmzbot/internal/config"
	"github.com/susu3304/nkmzbot/internal/db"
	"github.com/susu3304/nkmzbot/internal/imm"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	database, err := db.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := database.RunMigrations(context.Background()); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	immRunner := imm.NewRunner(cfg.IMMBinaryPath)
	immRunner.Timeout = cfg.IMMTimeout
	immRunner.MaxSourceBytes = cfg.IMMMaxSourceBytes
	immRunner.MaxOutputBytes = cfg.IMMMaxOutputBytes

	// Initialize Discord bot
	discordBot, err := bot.New(cfg.DiscordToken, cfg.APIURL, cfg.APIToken, database, immRunner)
	if err != nil {
		log.Fatalf("Failed to create discord bot: %v", err)
	}

	// Start Discord bot
	if err := discordBot.Start(); err != nil {
		log.Fatalf("Failed to start discord bot: %v", err)
	}
	defer discordBot.Stop()

	// Wait for signal to stop
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down...")
}
