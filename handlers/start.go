package handlers

import (
	"sync"

	"gopkg.in/telebot.v3"
)

type BotHandlers struct {
	bot    *telebot.Bot
	states map[int64]string
	mu     sync.Mutex
}

func New(bot *telebot.Bot) *BotHandlers {
	return &BotHandlers{
		bot:    bot,
		states: make(map[int64]string),
	}
}

func (h *BotHandlers) RegistrationHandler() {
	h.bot.Handle("/start", h.startHandler)
	h.bot.Handle(telebot.OnText, h.textHandler)
	h.bot.Handle(telebot.OnPhoto, h.photoHandler)
}

func (h *BotHandlers) startHandler(c telebot.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.states[c.Sender().ID] = "name"
	return c.Send("Привет! Я бот для знакомств. Давай создадим тебе анкету\nКак тебя зовут?")
}

func (h *BotHandlers) textHandler(c telebot.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	state := h.states[c.Sender().ID]

	switch state {
	case "name":
		h.states[c.Sender().ID] = "age"
		return c.Send("Сколько тебе лет?")
	case "age":
		h.states[c.Sender().ID] = "photo"
		return c.Send("Теперь отправь свое фото")
	default:
		return c.Send("Что-то пошло не так.../start")
	}
}

func (h *BotHandlers) photoHandler(c telebot.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.states[c.Sender().ID] != "photo" {
		return c.Send("Сначала заполните анкету!")
	}
	return c.Send("Ваша анкета готова!")
}
