package utils

import (
	"database/sql"
	"dating_bot/models"
	"fmt"
	"log"
	"time"

	"gopkg.in/telebot.v3"
)

type Delete_Watched struct {
	*models.BotHandlers
}

func DeleteHandler(bot *telebot.Bot, db *sql.DB) *Delete_Watched {
	return &Delete_Watched{
		BotHandlers: models.New(bot, db),
	}
}

func (h *Delete_Watched) CleanOldRecords() error {
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	_, err := h.Db.Exec(`
        DELETE FROM watched 
        WHERE DATE(viewed_at) = ?`, yesterday)
	if err != nil {
		return fmt.Errorf("ошибка при очистке watched: %v", err)
	}

	_, err = h.Db.Exec(`
        DELETE FROM liked_users 
        WHERE DATE(date) = ?`, yesterday)
	if err != nil {
		return fmt.Errorf("ошибка при очистке liked_users: %v", err)
	}

	return nil
}

func (h *Delete_Watched) StartDailyCleaner() {
	go func() {
		for {
			now := time.Now()

			next := now.AddDate(0, 0, 1)
			next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
			duration := next.Sub(now)

			time.Sleep(duration)

			if err := h.CleanOldRecords(); err != nil {
				log.Printf("Ошибка при ежедневной очистке: %v", err)
			} else {
				log.Println("Ежедневная очистка выполнена успешно")
			}
		}
	}()
}
