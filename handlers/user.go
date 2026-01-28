package handlers

import (
	"log"
	"time"

	"dorm-bot/database"
	"dorm-bot/services"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UserHandler struct {
	bot          *tgbotapi.BotAPI
	db           *database.DB
	groupManager *services.GroupManager
}

func NewUserHandler(bot *tgbotapi.BotAPI, db *database.DB, gm *services.GroupManager) *UserHandler {
	return &UserHandler{
		bot:          bot,
		db:           db,
		groupManager: gm,
	}
}

func (h *UserHandler) HandleMessage(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	tgID := msg.From.ID

	// Определяем состояние юзера
	state, err := h.db.DetermineUserState(tgID)
	if err != nil {
		log.Printf("Error determining user state: %v", err)
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Ошибка при проверке"))
		return
	}

	switch state {
	case "blocked":
		h.handleBlockedUser(msg)
	case "approved":
		h.handleApprovedUser(msg)
	case "unverified":
		h.handleUnverifiedUser(msg)
	}
}

func (h *UserHandler) handleBlockedUser(msg *tgbotapi.Message) {
	h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
		"❌ Ты в черном списке. Свяжись с администрацией."))
}

func (h *UserHandler) handleApprovedUser(msg *tgbotapi.Message) {
	tgID := msg.From.ID

	// Добавляем в группу
	h.groupManager.AddUserToGroup(tgID, msg.From.FirstName)

	h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
		"✅ Добро пожаловать в группу!"))

	h.db.AddVerificationLog(tgID, "auto_approved_from_whitelist", nil, "")
}

func (h *UserHandler) handleUnverifiedUser(msg *tgbotapi.Message) {
	tgID := msg.From.ID

	// Проверяем, есть ли уже активная заявка
	existingApp, err := h.db.GetActiveApplication(tgID)
	if err != nil {
		log.Printf("Error getting application: %v", err)
		return
	}

	if existingApp != nil {
		if existingApp.Status == "pending" {
			h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
				"⏳ Твоя заявка уже на проверке. Жди ответа администратора."))
			return
		}

		if existingApp.Status == "rework_requested" {
			h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
				"⚠️ На доработку: "+existingApp.AdminComment+"\n\nОтправь новые документ и кружок."))
		}
	}

	// Проверяем: отправил ли документ + кружок?
	if msg.Document != nil && msg.Voice != nil {
		h.handleDocumentSubmission(msg, tgID)
	} else if msg.Document != nil && msg.Voice == nil {
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
			"⚠️ Получил фото, но нужен ещё кружок (голосовое сообщение)."))
	} else if msg.Document == nil && msg.Voice != nil {
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
			"⚠️ Получил кружок, но нужен ещё документ (фото паспорта)."))
	} else if msg.Video != nil || msg.Animation != nil {
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
			"❌ Видео и ГИФ не подходят. Отправь фото паспорта и кружок."))
	} else if msg.Sticker != nil {
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
			"❌ Это не документ. Отправляй фото паспорта и кружок."))
	} else if msg.Text != "" {
		if existingApp == nil {
			h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, `
Для вступления в группу подтверди, что ты проживаешь в общаге.

Отправь:
1️⃣ Фото паспорта (или студака / направления / пропуска)
2️⃣ Кружок (голосовое подтверждение)
			`))
		} else {
			h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
				"❌ Отправляй документ и кружок, не текст."))
		}
	} else {
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
			"❌ Что-то пошло не так. Отправляй документ и кружок."))
	}
}

func (h *UserHandler) handleDocumentSubmission(msg *tgbotapi.Message, tgID int64) {
	application := &database.Application{
		TelegramID:      tgID,
		DocumentFileID:  msg.Document.FileID,
		VoiceNoteFileID: msg.Voice.FileID,
		Status:          "pending",
		SubmittedAt:     time.Now(),
	}

	err := h.db.CreateApplication(application)
	if err != nil {
		log.Printf("Error creating application: %v", err)
		h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
			"❌ Ошибка при сохранении документов. Попробуй позже."))
		return
	}

	h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
		"✅ Документы получены! Администратор проверит их в ближайшее время."))

	h.forwardToAdmins(msg, tgID)
	h.db.AddVerificationLog(tgID, "submitted", nil, "")
}

func (h *UserHandler) forwardToAdmins(msg *tgbotapi.Message, tgID int64) {
	log.Printf("New application from user %d", tgID)
}
