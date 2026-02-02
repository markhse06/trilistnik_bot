package bot

import (
	"context"
	"dorm-bot/internal/models"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api         *tgbotapi.BotAPI
	state       *StateManager
	store       *models.Store
	adminChatID int64
	groupID     int64
}

func NewBot(api *tgbotapi.BotAPI, store *models.Store, adminChatID, groupID int64) *Bot {
	return &Bot{
		api:         api,
		state:       NewStateManager(),
		store:       store,
		adminChatID: adminChatID,
		groupID:     groupID,
	}
}

func (b *Bot) Run() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		switch {
		case update.ChatJoinRequest != nil:
			b.handleJoinRequest(update.ChatJoinRequest)
		case update.Message != nil:
			b.handleMessage(update.Message)
		case update.CallbackQuery != nil:
			b.handleCallback(update.CallbackQuery)
		}
	}
}

// ОБРАБОТКА ЗАЯВОК В ГРУППУ

func (b *Bot) handleJoinRequest(req *tgbotapi.ChatJoinRequest) {
	if req.Chat.ID != b.groupID {
		return
	}

	ctx := context.Background()
	user := req.From

	isWhite, err := b.store.IsWhitelisted(ctx, user.ID)
	if err != nil {
		log.Printf("IsWhitelisted error: %v", err)
	}
	isBlack, err := b.store.IsBlacklisted(ctx, user.ID)
	if err != nil {
		log.Printf("IsBlacklisted error: %v", err)
	}

	_ = b.store.UpsertUser(ctx, user.ID, user.UserName, user.FirstName+" "+user.LastName, models.StatusUnverifed)

	// жёстко забаненные — сразу отклоняем
	if isBlack {
		if err := b.declineJoinRequest(req.Chat.ID, user.ID); err != nil {
			log.Printf("decline join req error: %v", err)
		}
		return
	}

	// белый список — сразу одобряем
	if isWhite {
		cfg := tgbotapi.ApproveChatJoinRequestConfig{
			ChatConfig: tgbotapi.ChatConfig{ChatID: req.Chat.ID},
			UserID:     user.ID,
		}
		if _, err := b.api.Request(cfg); err != nil {
			log.Printf("approve join req error: %v", err)
		} else {
			_ = b.store.UpdateUserStatus(ctx, user.ID, models.StatusApproved)
		}
		return
	}

	// иначе — диалог
	b.state.SetState(user.ID, StateWaitFullName)
	b.state.SetSession(user.ID, &Session{
		UserID:      user.ID,
		GroupID:     req.Chat.ID,
		MediaMsgIDs: make([]int, 0),
	})

	text := "Привет! 🎓 Это бот общежития.\n" +
		"Напиши, пожалуйста, своё ФИО одним сообщением."
	msg := tgbotapi.NewMessage(user.ID, text)
	if _, err := b.api.Send(msg); err != nil {
		warn := tgbotapi.NewMessage(
			b.adminChatID,
			"❗️ Не удалось написать пользователю @"+user.UserName+
				" (ID: "+strconv.FormatInt(user.ID, 10)+"), подавшему заявку в \""+req.Chat.Title+"\".",
		)
		b.api.Send(warn)
	}
}

// ===== ХЕНДЛЕР СООБЩЕНИЙ =====

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	if msg.Chat.IsGroup() || msg.Chat.IsSuperGroup() || msg.Chat.IsChannel() {
		return
	}

	userID := msg.From.ID

	if msg.IsCommand() {
		b.handleCommand(msg)
		return
	}

	state := b.state.GetState(userID)

	switch state {
	case StateWaitFullName:
		b.handleFullName(msg)
	case StateWaitDocs:
		b.handleDocs(msg)
	case StateAdminWaitRejectReason:
		b.handleAdminRejectReason(msg)
	default:
		reply := tgbotapi.NewMessage(userID, "Чтобы начать, подай заявку на вступление в группу общежития, и бот сам тебе напишет.")
		b.api.Send(reply)
	}
}

func (b *Bot) handleAdminRejectReason(msg *tgbotapi.Message) {
	adminID := msg.From.ID
	reason := strings.TrimSpace(msg.Text)
	if reason == "" {
		b.api.Send(tgbotapi.NewMessage(adminID, "Причина не может быть пустой. Напиши коротко, почему отказ."))
		return
	}

	sess, ok := b.state.GetSession(adminID)
	if !ok || sess.RejectTargetUserID == 0 || sess.RejectTargetChatID == 0 {
		b.api.Send(tgbotapi.NewMessage(adminID, "Сессия отказа не найдена. Нажми ещё раз кнопку «Отклонить»."))
		b.state.Clear(adminID)
		return
	}

	targetUserID := sess.RejectTargetUserID
	targetChatID := sess.RejectTargetChatID

	ctx := context.Background()

	// отклоняем join request
	if err := b.declineJoinRequest(targetChatID, targetUserID); err != nil {
		log.Printf("reject error: %v", err)
		b.api.Send(tgbotapi.NewMessage(adminID, "Не удалось отклонить заявку. Попробуй ещё раз."))
		b.state.Clear(adminID)
		return
	}

	// логируем: статус + reason в blacklist (или отдельную таблицу, но у нас уже есть reason)
	if err := b.store.AddToBlacklist(ctx, targetUserID, reason); err != nil {
		log.Printf("blacklist on reject error: %v", err)
	}
	_ = b.store.UpdateUserStatus(ctx, targetUserID, models.StatusRejected)

	// пишем пользователю с причиной
	textForUser := "❌ Твоя заявка в чат общежития отклонена.\n" +
		"Причина: " + reason +
		"\n\nЕсли считаешь, что это ошибка, то напиши админам или повтори попытку."
	b.api.Send(tgbotapi.NewMessage(targetUserID, textForUser))

	// уведомляем админа(ов)
	b.api.Send(tgbotapi.NewMessage(adminID, "Заявка отклонена. Причина отправлена пользователю."))

	// чистим состояние админа
	b.state.Clear(adminID)
}

func (b *Bot) handleFullName(msg *tgbotapi.Message) {
	userID := msg.From.ID
	fullName := strings.TrimSpace(msg.Text)
	if fullName == "" {
		reply := tgbotapi.NewMessage(userID, "Пожалуйста, отправь ФИО текстом одним сообщением.")
		b.api.Send(reply)
		return
	}

	sess, ok := b.state.GetSession(userID)
	if !ok {
		reply := tgbotapi.NewMessage(userID, "Сессия не найдена, попробуй подать заявку заново через группу.")
		b.api.Send(reply)
		return
	}
	sess.FullName = fullName
	b.state.SetSession(userID, sess)

	ctx := context.Background()
	_ = b.store.UpsertUser(ctx, userID, msg.From.UserName, fullName, models.StatusPending)

	b.state.SetState(userID, StateWaitDocs)

	text := "Спасибо, " + fullName + "!\n" +
		"Чтобы попасть в группу тебе нужно подтвердить проживание в общежитие Трилистник. Это можно сделать следующим способом:\n" +
		"1) Фото студенческого, пропуска, паспорта, карты москвича + кружок с этим документом\n\n" +
		"Если ты только заселился(ась)/заселяешься (у тебя не отображается общага в HSE App X), то можешь отправить направление в общежитие + кружок с этим направлением \n\n"
	reply := tgbotapi.NewMessage(userID, text)
	b.api.Send(reply)

	// Отправляем доп инструкцию отдельным сообщением
	text = "Все фото и кружки просто пересылай сюда (можно несколькими сообщениями).\n" +
		"Когда отправишь всё, нажми кнопку «Готово», чтобы отправить анкету админам."
	reply = tgbotapi.NewMessage(userID, text)
	b.api.Send(reply)
}

func (b *Bot) handleDocs(msg *tgbotapi.Message) {
	userID := msg.From.ID
	sess, ok := b.state.GetSession(userID)
	if !ok {
		reply := tgbotapi.NewMessage(userID, "Сессия не найдена, попробуй подать заявку заново через группу.")
		b.api.Send(reply)
		return
	}

	// накапливаем ID всех сообщений
	sess.MediaMsgIDs = append(sess.MediaMsgIDs, msg.MessageID)
	b.state.SetSession(userID, sess)

	// если кнопки ещё нет – показываем
	if sess.ConfirmMsgID == 0 {
		text := "Если это все фото/кружки, нажми «Готово», чтобы отправить анкету админам."
		btn := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Готово, отправить админам", "confirm_docs"),
			),
		)
		m := tgbotapi.NewMessage(userID, text)
		m.ReplyMarkup = btn
		sent, _ := b.api.Send(m)

		sess.ConfirmMsgID = sent.MessageID
		b.state.SetSession(userID, sess)
	}
}

// ОБРАБОТКА КНОПОК

func (b *Bot) handleCallback(cb *tgbotapi.CallbackQuery) {
	data := cb.Data

	// кнопка пользователя "Готово, отправить админам"
	if data == "confirm_docs" {
		b.handleConfirmDocs(cb)
		return
	}

	// дальше — кнопки админов approve/reject
	parts := strings.SplitN(data, ":", 3)
	if len(parts) != 3 {
		b.answerCallback(cb, "Некорректные данные.")
		return
	}

	action := parts[0]
	chatID, err1 := strconv.ParseInt(parts[1], 10, 64)
	userID, err2 := strconv.ParseInt(parts[2], 10, 64)
	if err1 != nil || err2 != nil {
		b.answerCallback(cb, "Ошибка парсинга ID.")
		return
	}

	ctx := context.Background()

	switch action {
	case "approve":
		cfg := tgbotapi.ApproveChatJoinRequestConfig{
			ChatConfig: tgbotapi.ChatConfig{ChatID: chatID},
			UserID:     userID,
		}
		if _, err := b.api.Request(cfg); err != nil {
			log.Printf("approve error: %v", err)
			b.answerCallback(cb, "Не удалось одобрить.")
			return
		}

		_ = b.store.AddToWhitelist(ctx, userID)
		_ = b.store.UpdateUserStatus(ctx, userID, models.StatusApproved)

		msg := tgbotapi.NewMessage(userID, "✅ Твоя заявка одобрена, добро пожаловать в чат общежития!")
		b.api.Send(msg)

		b.answerCallback(cb, "Пользователь одобрен и добавлен в белый список.")

	case "reject":
		// вместо сразу declineJoinRequest → просим причину у админа
		adminID := cb.From.ID

		// сохраняем в сессию админа данные, кого отклоняем
		b.state.SetState(adminID, StateAdminWaitRejectReason)
		b.state.SetSession(adminID, &Session{
			UserID:             adminID,
			RejectTargetUserID: userID,
			RejectTargetChatID: chatID,
		})

		b.answerCallback(cb, "Напиши в ЛС причину отказа этому пользователю.")
		b.api.Send(tgbotapi.NewMessage(adminID, "Напиши причину отказа для пользователя ID "+strconv.FormatInt(userID, 10)+":"))

	default:
		b.answerCallback(cb, "Неизвестное действие.")
	}
}

// ОБРАБОТКА СООБЩЕНИЙ С ДОКУМЕНТАМИ

func (b *Bot) handleConfirmDocs(cb *tgbotapi.CallbackQuery) {
	userID := cb.From.ID

	sess, ok := b.state.GetSession(userID)
	if !ok {
		b.answerCallback(cb, "Сессия не найдена, попробуй подать заявку заново.")
		return
	}

	username := cb.From.UserName
	link := "tg://user?id=" + strconv.FormatInt(userID, 10)
	header := "📝 Новая анкета на проверку\n" +
		"ФИО: " + sess.FullName + "\n" +
		"User: @" + username + " (" + link + ")\n" +
		"ID: " + strconv.FormatInt(userID, 10) + "\n" +
		"Группа: " + strconv.FormatInt(sess.GroupID, 10)
	hdrMsg := tgbotapi.NewMessage(b.adminChatID, header)
	b.api.Send(hdrMsg)

	for _, mid := range sess.MediaMsgIDs {
		fwd := tgbotapi.NewForward(b.adminChatID, userID, mid)
		if _, err := b.api.Send(fwd); err != nil {
			log.Printf("forward error: %v", err)
		}
	}

	approveData := "approve:" + strconv.FormatInt(sess.GroupID, 10) + ":" + strconv.FormatInt(userID, 10)
	rejectData := "reject:" + strconv.FormatInt(sess.GroupID, 10) + ":" + strconv.FormatInt(userID, 10)

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Одобрить", approveData),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", rejectData),
		),
	)

	adminMsg := tgbotapi.NewMessage(b.adminChatID, "Выбери действие для этого пользователя:")
	adminMsg.ReplyMarkup = kb
	b.api.Send(adminMsg)

	b.answerCallback(cb, "Отправил анкету админам ✅")
	b.api.Send(tgbotapi.NewMessage(userID, "Отправил твою анкету на проверку админам. Ожидай решения 🙌"))

	b.state.Clear(userID)
}

// КОМАНДЫ

func (b *Bot) isAdmin(userID int64) bool {
	// если у тебя несколько админов — лучше хранить список ID в конфиге
	return userID == b.adminChatID
}

func (b *Bot) handleCommand(msg *tgbotapi.Message) {
	userID := msg.From.ID
	cmd := msg.Command()
	args := strings.TrimSpace(msg.CommandArguments())

	switch cmd {
	case "start":
		text := "Привет! Я бот для проверки заявок в чат общаги.\n" +
			"Подай заявку на вступление в группу — я сам тебе напишу."
		b.api.Send(tgbotapi.NewMessage(userID, text))

	case "help":
		text := "Я помогаю админам проверять заявки в чат общежития.\n" +
			"1) Подаёшь заявку\n" +
			"2) Я спрашиваю ФИО и документы\n" +
			"3) Отправляю анкету админам\n" +
			"4) Они решают, пускать или нет."

		if b.isAdmin(userID) {
			text = "Привет. Тебе не повезло и ты админ чата." +
				"\nПросто жди когда кто-то подаст заявку на вступление в чат.\n" +
				"Тебе нужно лишь проверить документы пользователя и жмакать на кнопочки.\n\n" +
				"Также у тебя, как у админа, есть список команд:\n" +
				"\\whitelist_add <userID> - добавляет пользователя в белый список\n" +
				"\\whitelist_remove <userID> - удаляет пользователя из белого списка\n" +
				"\\blacklist_add <userID> - добавляет пользователя в черный список\n" +
				"\\blacklist_remove <userID> - удаляет пользователя из черного списка\n\n" +
				"Подробная документация: https://github.com/marhse06/trilistnik_bot/documentation.md"
		}
		b.api.Send(tgbotapi.NewMessage(userID, text))

	case "whitelist_add":
		if !b.isAdmin(userID) {
			b.api.Send(tgbotapi.NewMessage(userID, "Команда только для админов."))
			return
		}
		if args == "" {
			b.api.Send(tgbotapi.NewMessage(userID, "Использование: /whitelist_add <telegram_id>"))
			return
		}
		targetID, err := strconv.ParseInt(args, 10, 64)
		if err != nil {
			b.api.Send(tgbotapi.NewMessage(userID, "Некорректный telegram_id."))
			return
		}
		ctx := context.Background()
		if err := b.store.AddToWhitelist(ctx, targetID); err != nil {
			log.Printf("whitelist_add error: %v", err)
			b.api.Send(tgbotapi.NewMessage(userID, "Ошибка при добавлении в белый список."))
			return
		}
		if err := b.store.RemoveFromBlacklist(ctx, targetID); err != nil {
			log.Printf("blacklist_remove error while adding user to whitelist: %v", err)
		}
		_ = b.store.UpdateUserStatus(ctx, targetID, models.StatusApproved)
		b.api.Send(tgbotapi.NewMessage(userID, "Пользователь добавлен в белый список."))

	case "whitelist_remove":
		if !b.isAdmin(userID) {
			b.api.Send(tgbotapi.NewMessage(userID, "Команда только для админов."))
			return
		}
		if args == "" {
			b.api.Send(tgbotapi.NewMessage(userID, "Использование: /whitelist_remove <telegram_id>"))
			return
		}
		targetID, err := strconv.ParseInt(args, 10, 64)
		if err != nil {
			b.api.Send(tgbotapi.NewMessage(userID, "Некорректный telegram_id."))
			return
		}
		ctx := context.Background()
		if err := b.store.RemoveFromWhitelist(ctx, targetID); err != nil {
			log.Printf("whitelist_remove error: %v", err)
			b.api.Send(tgbotapi.NewMessage(userID, "Ошибка при удалении из белого списка."))
			return
		}
		b.api.Send(tgbotapi.NewMessage(userID, "Пользователь удалён из белого списка."))

	case "blacklist_add":
		if !b.isAdmin(userID) {
			b.api.Send(tgbotapi.NewMessage(userID, "Команда только для админов."))
			return
		}
		if args == "" {
			b.api.Send(tgbotapi.NewMessage(userID, "Использование: /blacklist_add <telegram_id> [причина]"))
			return
		}
		parts := strings.SplitN(args, " ", 2)
		targetID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			b.api.Send(tgbotapi.NewMessage(userID, "Некорректный telegram_id."))
			return
		}
		reason := ""
		if len(parts) == 2 {
			reason = parts[1]
		}
		ctx := context.Background()
		if err := b.store.AddToBlacklist(ctx, targetID, reason); err != nil {
			log.Printf("blacklist_add error: %v", err)
			b.api.Send(tgbotapi.NewMessage(userID, "Ошибка при добавлении в чёрный список."))
			return
		}
		if err := b.store.RemoveFromWhitelist(ctx, targetID); err != nil {
			log.Printf("whitelist_remove error while adding user to blacklist: %v", err)
		}
		_ = b.store.UpdateUserStatus(ctx, targetID, models.StatusBlocked)
		b.api.Send(tgbotapi.NewMessage(userID, "Пользователь добавлен в чёрный список."))

	case "blacklist_remove":
		if !b.isAdmin(userID) {
			b.api.Send(tgbotapi.NewMessage(userID, "Команда только для админов."))
			return
		}
		if args == "" {
			b.api.Send(tgbotapi.NewMessage(userID, "Использование: /blacklist_remove <telegram_id>"))
			return
		}
		targetID, err := strconv.ParseInt(args, 10, 64)
		if err != nil {
			b.api.Send(tgbotapi.NewMessage(userID, "Некорректный telegram_id."))
			return
		}
		ctx := context.Background()
		if err := b.store.RemoveFromBlacklist(ctx, targetID); err != nil {
			log.Printf("blacklist_remove error: %v", err)
			b.api.Send(tgbotapi.NewMessage(userID, "Ошибка при удалении из чёрного списка."))
			return
		}
		b.api.Send(tgbotapi.NewMessage(userID, "Пользователь удалён из чёрного списка."))

	case "status":
		if !b.isAdmin(userID) {
			b.api.Send(tgbotapi.NewMessage(userID, "Команда только для админов."))
			return
		}

		if args == "" {
			b.api.Send(tgbotapi.NewMessage(userID, "Использование: /status <telegram_id>"))
			return
		}
		targetID, err := strconv.ParseInt(args, 10, 64)
		if err != nil {
			b.api.Send(tgbotapi.NewMessage(userID, "Некорректный telegram_id."))
			return
		}
		ctx := context.Background()
		if status, err := b.store.GetUserStatus(ctx, targetID); err != nil {
			log.Printf("blacklist_remove error: %v", err)
			b.api.Send(tgbotapi.NewMessage(userID, "Ошибка при удалении из чёрного списка."))
			return
		} else {
			text := "Статус пользователя: "
			if status == "blacklisted" {
				text += "в черном списке"
			} else if status == "whitelisted" {
				text += "в белом списке"
			} else {
				text += "не установлено"
			}

			b.api.Send(tgbotapi.NewMessage(userID, text))
		}

	default:
		b.api.Send(tgbotapi.NewMessage(userID, "Не знаю такую команду. Попробуй /help."))
	}
}

// ===== UTILS =====

func (b *Bot) answerCallback(cb *tgbotapi.CallbackQuery, text string) {
	resp := tgbotapi.NewCallback(cb.ID, text)
	_, _ = b.api.Request(resp)
}

func (b *Bot) declineJoinRequest(chatID, userID int64) error {
	params := map[string]string{
		"chat_id": strconv.FormatInt(chatID, 10),
		"user_id": strconv.FormatInt(userID, 10),
	}
	_, err := b.api.MakeRequest("declineChatJoinRequest", params)
	return err
}
