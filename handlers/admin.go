package handlers

import (
	"fmt"
	"log"
	"strconv"

	"dorm-bot/config"
	"dorm-bot/database"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type AdminHandler struct {
	bot    *tgbotapi.BotAPI
	db     *database.DB
	config *config.Config
}

func NewAdminHandler(bot *tgbotapi.BotAPI, db *database.DB, cfg *config.Config) *AdminHandler {
	return &AdminHandler{
		bot:    bot,
		db:     db,
		config: cfg,
	}
}

func (h *AdminHandler) HandleCommand(update tgbotapi.Update) {
	if update.Message == nil || !update.Message.IsCommand() {
		return
	}

	msg := update.Message

	if !h.config.IsAdmin(msg.From.ID) {
		return
	}

	switch msg.Command() {
	case "applications":
		h.handleApplicationsCommand(msg)
	case "whitelist":
		h.handleWhitelistCommand(msg)
	case "blacklist":
		h.handleBlacklistCommand(msg)
	case "stats":
		h.handleStatsCommand(msg)
	case "export_whitelist":
		h.handleExportWhitelistCommand(msg)
	case "export_blacklist":
		h.handleExportBlacklistCommand(msg)
	}
}

func (h *AdminHandler) handleApplicationsCommand(msg *tgbotapi.Message) {
	apps, err := h.db.GetPendingApplications()
	if err != nil {
		log.Printf("Error getting applications: %v", err)
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Ошибка"))
		return
	}

	if len(apps) == 0 {
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Нет ожидающих заявок!"))
		return
	}

	for _, app := range apps {
		text := fmt.Sprintf(`
👤 Заявка пользователя

Telegram ID: %d
Попытка: %d
Отправлена: %s
Status: %s
		`, app.TelegramID, app.ReworkAttempts+1, app.SubmittedAt.Format("02.01.2006 15:04"), app.Status)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Одобрить", fmt.Sprintf("approve_%d", app.TelegramID)),
				tgbotapi.NewInlineKeyboardButtonData("⚠️ На доработку", fmt.Sprintf("rework_%d", app.TelegramID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("reject_%d", app.TelegramID)),
				tgbotapi.NewInlineKeyboardButtonData("⛔ В ЧС", fmt.Sprintf("block_%d", app.TelegramID)),
			),
		)

		newMsg := tgbotapi.NewMessage(msg.Chat.ID, text)
		newMsg.ReplyMarkup = keyboard
		h.bot.Send(newMsg)
	}
}

func (h *AdminHandler) handleWhitelistCommand(msg *tgbotapi.Message) {
	entries, err := h.db.GetWhitelistAll()
	if err != nil {
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Ошибка"))
		return
	}

	if len(entries) == 0 {
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Белый список пуст"))
		return
	}

	text := fmt.Sprintf("📋 Белый список (%d):\n\n", len(entries))
	for i, entry := range entries {
		if i >= 20 {
			text += fmt.Sprintf("... и ещё %d\n", len(entries)-20)
			break
		}
		text += fmt.Sprintf("%d. %s (%d)\n", i+1, entry.FirstName, entry.TelegramID)
	}

	h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
}

func (h *AdminHandler) handleBlacklistCommand(msg *tgbotapi.Message) {
	entries, err := h.db.GetBlacklistAll()
	if err != nil {
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Ошибка"))
		return
	}

	if len(entries) == 0 {
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Чёрный список пуст"))
		return
	}

	text := fmt.Sprintf("⛔ Чёрный список (%d):\n\n", len(entries))
	for i, entry := range entries {
		if i >= 20 {
			text += fmt.Sprintf("... и ещё %d\n", len(entries)-20)
			break
		}
		text += fmt.Sprintf("%d. %s (%d) - %s\n", i+1, entry.FirstName, entry.TelegramID, entry.Reason)
	}

	h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
}

func (h *AdminHandler) handleStatsCommand(msg *tgbotapi.Message) {
	stats, err := h.db.GetStats()
	if err != nil {
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Ошибка"))
		return
	}

	text := fmt.Sprintf(`
📊 Статистика:

⏳ В процессе: %d
🔄 На доработку: %d
✅ Одобрено: %d
❌ Отклонено: %d
⛔ Заблокировано: %d
	`, stats["pending"], stats["rework"], stats["approved"], stats["rejected"], stats["blocked"])

	h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
}

func (h *AdminHandler) handleExportWhitelistCommand(msg *tgbotapi.Message) {
	entries, err := h.db.GetWhitelistAll()
	if err != nil {
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Ошибка"))
		return
	}

	text := "telegram_id,first_name,added_at\n"
	for _, entry := range entries {
		text += fmt.Sprintf("%d,%s,%s\n", entry.TelegramID, entry.FirstName, entry.AddedAt.Format("2006-01-02 15:04:05"))
	}

	h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
}

func (h *AdminHandler) handleExportBlacklistCommand(msg *tgbotapi.Message) {
	entries, err := h.db.GetBlacklistAll()
	if err != nil {
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Ошибка"))
		return
	}

	text := "telegram_id,first_name,reason,added_at\n"
	for _, entry := range entries {
		text += fmt.Sprintf("%d,%s,%s,%s\n", entry.TelegramID, entry.FirstName, entry.Reason, entry.AddedAt.Format("2006-01-02 15:04:05"))
	}

	h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
}

func (h *AdminHandler) HandleTextCommand(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	msg := update.Message

	if !h.config.IsAdmin(msg.From.ID) {
		return
	}

	// /whitelist_remove 123456
	if len(msg.Text) > 18 && msg.Text[:18] == "/whitelist_remove " {
		idStr := msg.Text[18:]
		tgID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Неправильный ID"))
			return
		}

		err = h.db.RemoveFromWhitelist(tgID)
		if err != nil {
			h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Ошибка"))
			return
		}

		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Удален из белого списка"))
		h.db.AddVerificationLog(tgID, "removed_from_whitelist", &msg.From.ID, "")
	}

	// /blacklist_remove 123456
	if len(msg.Text) > 17 && msg.Text[:17] == "/blacklist_remove " {
		idStr := msg.Text[17:]
		tgID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Неправильный ID"))
			return
		}

		err = h.db.RemoveFromBlacklist(tgID)
		if err != nil {
			h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Ошибка"))
			return
		}

		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Удален из чёрного списка"))
		h.db.AddVerificationLog(tgID, "removed_from_blacklist", &msg.From.ID, "")
	}
}
