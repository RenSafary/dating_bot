package handlers

import (
	"database/sql"
	"dating_bot/models"
	"fmt"
	"log"

	"gopkg.in/telebot.v3"
)

type HandlersNoti struct {
	*models.BotHandlers
}

func NewHandlersNoti(bot *telebot.Bot, db *sql.DB) *Handlers {
	return &Handlers{
		BotHandlers: models.New(bot, db),
	}
}

func (h *HandlersNoti) SetupHandlers() {
	h.Bot.Handle("/myprofile", h.NotificationHandler)
}

func (hn *HandlersNoti) NotificationHandler(c telebot.Context) error {
	id := c.Sender().ID

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
		log.Fatal(err)
	}
	fmt.Println(id_tg)
}
