package handlers

import (
	"database/sql"
	"dating_bot/models"
	"fmt"

	"gopkg.in/telebot.v3"
)

type HandlersNoti struct {
	*models.BotHandlers
}

func NewHandlersNoti(bot *telebot.Bot, db *sql.DB) *HandlersNoti {
	return &HandlersNoti{
		BotHandlers: models.New(bot, db),
	}
}

func (hn *HandlersNoti) SetupHandlers() {
	hn.Bot.Handle("/notifications", hn.NotificationHandler)
}

func (hn *HandlersNoti) NotificationHandler(c telebot.Context) error {
	hn.Mu.Lock()
	defer hn.Mu.Unlock()

	id := c.Sender().ID
	hn.States[id] = "notifications"

	var likerID int64
	err := hn.Db.QueryRow(`
        SELECT user_id FROM liked_users 
        WHERE liked_id = ? AND watched = FALSE 
        LIMIT 1`,
		id).Scan(&likerID)

	if err != nil {
		if err == sql.ErrNoRows {
			return c.Send("У тебя нет уведомлений")
		}
		fmt.Println(err)
	}
	var text string
	photoPath := fmt.Sprintf("%d_photo.jpg", likerID)
	if info != "" {
		text = fmt.Sprintf("%s, %d", name, age)
	} else {
		text = fmt.Sprintf("%s, %d\n\n%s", name, age, info)
	}

	return c.Send(&telebot.Photo{
		File:    telebot.FromDisk(photoPath),
		Caption: text,
	})
}
