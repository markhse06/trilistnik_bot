package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken    string
	AdminChatID int64
	GroupID     int64
	DatabaseURL string
}

func Load() *Config {
	_ = godotenv.Load()

	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("BOT_TOKEN is required")
	}

	adminChatIDStr := os.Getenv("ADMIN_CHAT_ID")
	groupIDStr := os.Getenv("GROUP_ID")
	dbURL := os.Getenv("DATABASE_URL")

	if adminChatIDStr == "" || groupIDStr == "" || dbURL == "" {
		log.Fatal("ADMIN_CHAT_ID, GROUP_ID, DATABASE_URL are required")
	}

	adminChatID, err := strconv.ParseInt(adminChatIDStr, 10, 64)
	if err != nil {
		log.Fatalf("invalid ADMIN_CHAT_ID: %v", err)
	}

	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		log.Fatalf("invalid GROUP_ID: %v", err)
	}

	return &Config{
		BotToken:    botToken,
		AdminChatID: adminChatID,
		GroupID:     groupID,
		DatabaseURL: dbURL,
	}
}
