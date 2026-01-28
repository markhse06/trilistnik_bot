package services

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type GroupManager struct {
	bot      *tgbotapi.BotAPI
	groupIDs []int64
}

func NewGroupManager(bot *tgbotapi.BotAPI, groupIDs []int64) *GroupManager {
	return &GroupManager{
		bot:      bot,
		groupIDs: groupIDs,
	}
}

func (gm *GroupManager) AddUserToGroup(userID int64, firstName string) error {
	for _, groupID := range gm.groupIDs {
		config := tgbotapi.AddChatMemberConfig{
			ChatID: groupID,
			UserID: userID,
		}

		success, err := gm.bot.AddChatMember(config)
		if err != nil {
			log.Printf("Error adding user %d to group %d: %v", userID, groupID, err)
		} else {
			log.Printf("Successfully added user %d to group %d: %v", userID, groupID, success)
		}
	}

	return nil
}

func (gm *GroupManager) RemoveUserFromGroup(userID int64) error {
	for _, groupID := range gm.groupIDs {
		config := tgbotapi.RestrictChatMemberConfig{
			ChatID: groupID,
			UserID: userID,
			Permissions: &tgbotapi.ChatPermissions{
				CanSendMessages: false,
			},
		}

		success, err := gm.bot.RestrictChatMember(config)
		if err != nil {
			log.Printf("Error restricting user: %v", err)
		} else {
			log.Printf("Successfully restricted user: %v", success)
		}
	}

	return nil
}
