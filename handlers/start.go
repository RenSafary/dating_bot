package handlers

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/telebot.v3"
)

type BotHandlers struct {
	bot      *telebot.Bot
	dataUser map[string]string
	states   map[int64]string
	mu       sync.Mutex
}

func New(bot *telebot.Bot) *BotHandlers {
	return &BotHandlers{
		bot:      bot,
		dataUser: make(map[string]string),
		states:   make(map[int64]string),
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
		h.dataUser["name"] = c.Text()

		h.states[c.Sender().ID] = "age"
		return c.Send("Сколько тебе лет?")
	case "age":
		h.dataUser["age"] = c.Text()

		h.states[c.Sender().ID] = "gender"

		menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		btnMale := menu.Text("Мужской ♂️")
		btnFemale := menu.Text("Женский ♀️")

		menu.Reply(
			menu.Row(btnMale, btnFemale),
		)
		return c.Send("Укажи свой пол:", menu)
	case "gender":
		h.dataUser["gender"] = c.Text()
		h.states[c.Sender().ID] = "choice"

		menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		btnMale := menu.Text("М")
		btnFemale := menu.Text("Ж")
		btnEverybody := menu.Text("Без разницы")
		menu.Reply(
			menu.Row(btnMale, btnFemale, btnEverybody),
		)
		return c.Send("Кого хочешь искать?", menu)
	case "choice":
		h.dataUser["choice"] = c.Text()
		h.states[c.Sender().ID] = "info"

		menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		btnSkip := menu.Text("Пропустить")
		menu.Reply(
			menu.Row(btnSkip),
		)

		return c.Send("Напиши что-нибудь о себе", menu)
	case "info":
		h.dataUser["info"] = c.Text()
		h.states[c.Sender().ID] = "photo"

		return c.Send("Теперь отправь свое фото", &telebot.ReplyMarkup{RemoveKeyboard: true})
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
	id := int(c.Sender().ID)
	name := h.dataUser["name"]
	age := h.dataUser["age"]
	gender := h.dataUser["gender"]
	choice := h.dataUser["choice"]
	info := h.dataUser["info"]

	// convertation
	age_int, err := strconv.Atoi(age)
	if err != nil {
		fmt.Println("Couldn't convert", err)
	}

	photo := c.Message().Photo
	if photo == nil {
		return c.Send("Это не фото! Попробуй ещё раз...")
	}

	file, err := h.bot.FileByID(photo.FileID)
	if err != nil {
		fmt.Println(err)
	}
	filesDir := "./images/"
	fileName := fmt.Sprintf("%d_photo.jpg", c.Sender().ID)
	filePath := filepath.Join(filesDir, fileName)

	err = h.bot.Download(&file, filePath)
	if err != nil {
		fmt.Println("File wasn't saved")
	}

	err = h.saveInDB(id, age_int, name, gender, choice, info, filePath)
	if err != nil {
		fmt.Println("Ошибка сохранения в БД:", err)
	}
	return c.Send("Ваша анкета готова!")
}

func (h *BotHandlers) saveInDB(id, age int, name, gender, choice, info, photo string) error {
	db, err := sql.Open("sqlite3", "database/db.db")
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO users (name, age, gender, fav_gen, information, photo, id_tg) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		name, age, gender, choice, info, photo, id)
	return err
}
