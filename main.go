package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"dorm-bot/config"
	"dorm-bot/database"
	"dorm-bot/handlers"
	"dorm-bot/services"
)

func main() {
	// Загрузка .env
	godotenv.Load()

	// Загрузка конфига
	cfg, err := config.Load("config/config.yml")
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	log.Printf("Bot token: %s...", cfg.Telegram.Token[:20])
	log.Printf("Admin IDs: %v", cfg.Telegram.AdminIDs)
	log.Printf("Group IDs: %v", cfg.Telegram.GroupIDs)

	// Подключение к БД
	db, err := database.Connect(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("Database error: %v", err)
	}
	defer db.Close()

	// Миграции
	if err := db.RunMigrations(); err != nil {
		log.Fatalf("Migration error: %v", err)
	}

	// Инициализация бота
	bot, err := tgbotapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		log.Fatalf("Bot error: %v", err)
	}

	log.Printf("✅ Authorized as @%s", bot.Self.UserName)

	// Инициализация сервисов
	groupManager := services.NewGroupManager(bot, cfg.Telegram.GroupIDs)

	// Инициализация обработчиков
	userHandler := handlers.NewUserHandler(bot, db, groupManager)
	adminHandler := handlers.NewAdminHandler(bot, db, cfg)
	callbackHandler := handlers.NewCallbackHandler(bot, db, cfg, groupManager)

	// Запуск слушателя обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	log.Println("🤖 Bot started and waiting for updates...")

	for update := range updates {
		// Обработка callback кнопок
		if update.CallbackQuery != nil {
			go callbackHandler.HandleCallback(update)
			continue
		}

		// Обработка сообщений
		if update.Message != nil {
			if update.Message.IsCommand() {
				go adminHandler.HandleCommand(update)
				go adminHandler.HandleTextCommand(update)
			} else {
				go userHandler.HandleMessage(update)
			}
		}
	}
}
