package main

import (
	"database/sql"
	"io/ioutil"
	"log"

	"dorm-bot/internal/bot"
	"dorm-bot/internal/config"
	dbpkg "dorm-bot/internal/db"
	"dorm-bot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cfg := config.Load()

	db := dbpkg.Connect(cfg.DatabaseURL)
	defer db.Close()

	runMigrations(db)

	store := models.NewStore(db)

	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}
	api.Debug = false
	log.Printf("Authorized on account %s", api.Self.UserName)

	b := bot.NewBot(api, store, cfg.AdminChatID, cfg.GroupID)
	b.Run()
}

func runMigrations(db *sql.DB) {
	data, err := ioutil.ReadFile("internal/db/migrations.sql")
	if err != nil {
		log.Fatalf("failed to read migrations: %v", err)
	}
	if _, err := db.Exec(string(data)); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
}
