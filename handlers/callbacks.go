package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"dorm-bot/config"
	"dorm-bot/database"
	"dorm-bot/services"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CallbackHandler struct {
	bot          *tgbotapi.BotAPI
	db           *database.DB
	config       *config.Config
	groupManager *services.GroupManager
}

func NewCallbackHandler(bot *tgbotapi.BotAPI, db *database.DB, cfg *config.Config, gm *services.GroupManager) *CallbackHandler {
	return &CallbackHandler{
		bot:          bot,
		db:           db,
		config:       cfg,
		groupManager: gm,
	}
}

func (h *CallbackHandler) HandleCallback(update tgbotapi.Update) {
	query := update.CallbackQuery

	if !h.config.IsAdmin(query.From.ID) {
		h.bot.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, "❌ У вас нет доступа"))
		return
	}

	parts := strings.Split(query.Data, "_")
	if len(parts) < 2 {
		return
	}

	action := parts[0]
	tgIDStr := parts[1]
	tgID, err := strconv.ParseInt(tgIDStr, 10, 64)
	if err != nil {
		log.Printf("Error parsing telegram ID: %v", err)
		return
	}

	switch action {
	case "approve":
		h.handleApprove(query, tgID)
	case "reject":
		h.handleReject(query, tgID)
	case "rework":
		h.handleRework(query, tgID)
	case "block":
		h.handleBlock(query, tgID)
	}
}

func (h *CallbackHandler) handleApprove(query *tgbotapi.CallbackQuery, tgID int64) {
	// 1. Добавляем в whitelist
	err := h.db.AddToWhitelist(tgID, "")
	if err != nil {
		log.Printf("Error adding to whitelist: %v", err)
		h.bot.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, "❌ Ошибка"))
		return
	}

	// 2. Добавляем в группу
	h.groupManager.AddUserToGroup(tgID, "")

	// 3. Удаляем документы
	h.db.DeleteApplicationDocuments(tgID)

	// 4. Логируем
	adminID := query.From.ID
	h.db.AddVerificationLog(tgID, "approved", &adminID, "")

	// 5. Обновляем статус application
	h.db.UpdateApplicationStatus(tgID, "approved")

	// 6. Уведомляем юзера
	h.bot.Send(tgbotapi.NewMessage(tgID,
		"✅ Одобрено! Добро пожаловать в группу!"))

	// 7. Редактируем сообщение админу
	h.bot.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, "✅ Одобрено"))
	h.bot.EditMessageText(tgbotapi.EditMessageTextConfig{
		BaseEdit: tgbotapi.BaseEdit{ChatID: query.Message.Chat.ID, MessageID: query.Message.MessageID},
		Text:     "✅ Одобрено администратором: " + query.From.FirstName,
	})
}

func (h *CallbackHandler) handleReject(query *tgbotapi.CallbackQuery, tgID int64) {
	// 1. Обновляем статус
	h.db.UpdateApplicationStatus(tgID, "rejected")

	// 2. Удаляем документы
	h.db.DeleteApplicationDocuments(tgID)

	// 3. Логируем
	adminID := query.From.ID
	h.db.AddVerificationLog(tgID, "rejected", &adminID, "")

	// 4. Уведомляем юзера
	h.bot.Send(tgbotapi.NewMessage(tgID,
		"❌ Отклонено. Ты можешь попробовать подать заявку через неделю."))

	// 5. Редактируем сообщение админу
	h.bot.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, "❌ Отклонено"))
	h.bot.EditMessageText(tgbotapi.EditMessageTextConfig{
		BaseEdit: tgbotapi.BaseEdit{ChatID: query.Message.Chat.ID, MessageID: query.Message.MessageID},
		Text:     "❌ Отклонено администратором: " + query.From.FirstName,
	})
}

func (h *CallbackHandler) handleRework(query *tgbotapi.CallbackQuery, tgID int64) {
	h.bot.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, "Введи комментарий"))

	// Здесь нужна state machine для админов
	// Упрощенная версия: админ отправляет команду с комментарием
	// Пример: /rework_comment 123456789 Паспорт нечитаемый

	h.bot.Send(tgbotapi.NewMessage(query.From.ID,
		"Отправь: /rework_comment <telegram_id> <комментарий>"))
}

func (h *CallbackHandler) handleBlock(query *tgbotapi.CallbackQuery, tgID int64) {
	// 1. Добавляем в blacklist
	reason := "Решение администратора"
	adminID := query.From.ID
	err := h.db.AddToBlacklist(tgID, "", reason, adminID)
	if err != nil {
		log.Printf("Error adding to blacklist: %v", err)
		h.bot.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, "❌ Ошибка"))
		return
	}

	// 2. Удаляем документы
	h.db.DeleteApplicationDocuments(tgID)

	// 3. Логируем
	h.db.AddVerificationLog(tgID, "auto_blocked", &adminID, "admin_decision")

	// 4. Уведомляем юзера
	h.bot.Send(tgbotapi.NewMessage(tgID,
		"❌ Ты заблокирован. Свяжись с администрацией."))

	// 5. Редактируем сообщение админу
	h.bot.AnswerCallbackQuery(tgbotapi.NewCallback(query.ID, "⛔ Заблокирован"))
	h.bot.EditMessageText(tgbotapi.EditMessageTextConfig{
		BaseEdit: tgbotapi.BaseEdit{ChatID: query.Message.Chat.ID, MessageID: query.Message.MessageID},
		Text:     "⛔ Пользователь заблокирован администратором: " + query.From.FirstName,
	})
}
