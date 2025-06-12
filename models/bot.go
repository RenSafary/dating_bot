package models

import (
	"database/sql"
	"sync"

	"gopkg.in/telebot.v3"
)

type BotHandlers struct {
	Bot              *telebot.Bot
	Db               *sql.DB
	DataUser         map[string]string
	States           map[int64]string
	Mu               sync.Mutex
	CurrentProfileID map[int64]int64
}

func New(bot *telebot.Bot, db *sql.DB) *BotHandlers {
	return &BotHandlers{
		Bot:              bot,
		Db:               db,
		DataUser:         make(map[string]string),
		States:           make(map[int64]string),
		CurrentProfileID: make(map[int64]int64),
	}
}
