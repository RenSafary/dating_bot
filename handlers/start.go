package handlers

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/telebot.v3"
)

type BotHandlers struct {
	bot      *telebot.Bot
	db       *sql.DB
	dataUser map[string]string
	states   map[int64]string
	mu       sync.Mutex
}

func New(bot *telebot.Bot, db *sql.DB) *BotHandlers {
	return &BotHandlers{
		bot:      bot,
		db:       db,
		dataUser: make(map[string]string),
		states:   make(map[int64]string),
	}
}

func (h *BotHandlers) RegistrationHaскуфndler() {
	h.bot.Handle("/myprofile", h.startHandler)
	h.bot.Handle(telebot.OnText, h.textHandler)
	h.bot.Handle(telebot.OnPhoto, h.photoHandler)
}

func (h *BotHandlers) startHandler(c telebot.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := int(c.Sender().ID)

	// getting profile
	query := "SELECT name, age, gender, fav_gen, information, photo FROM users WHERE id_tg = ?"
	row := h.db.QueryRow(query, id)

	var (
		name      string
		age       int
		gender    string
		choice    string
		info      string
		photoPath string
	)

	err := row.Scan(&name, &age, &gender, &choice, &info, &photoPath)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println(err)
		}
		h.states[c.Sender().ID] = "name"
		return c.Send("Привет! Я бот для знакомств. Давай создадим тебе анкету\nКак тебя зовут?")
	}
	// getting photo
	if _, err := os.Stat(photoPath); os.IsNotExist(err) {
		fmt.Println("Фото не найдено")
	}
	// sending profile
	return c.Send(&telebot.Photo{
		File:    telebot.FromDisk(photoPath),
		Caption: fmt.Sprintf("%s, %d\n\n%s", name, age, info),
	})
}

func (h *BotHandlers) textHandler(c telebot.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	state := h.states[c.Sender().ID]

	id := c.Sender().ID

	switch state {
	case "name":
		h.dataUser[fmt.Sprintf("%d_name", id)] = c.Text()

		h.states[id] = "age"
		return c.Send("Сколько тебе лет?")
	case "age":
		h.dataUser[fmt.Sprintf("%d_age", id)] = c.Text()

		h.states[id] = "gender"

		menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		btnMale := menu.Text("Мужской ♂️")
		btnFemale := menu.Text("Женский ♀️")

		menu.Reply(
			menu.Row(btnMale, btnFemale),
		)
		return c.Send("Укажи свой пол:", menu)
	case "gender":
		h.dataUser[fmt.Sprintf("%d_gender", id)] = c.Text()
		h.states[id] = "choice"

		menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		btnMale := menu.Text("М")
		btnFemale := menu.Text("Ж")
		btnEverybody := menu.Text("Без разницы")
		menu.Reply(
			menu.Row(btnMale, btnFemale, btnEverybody),
		)
		return c.Send("Кого хочешь искать?", menu)
	case "choice":
		h.dataUser[fmt.Sprintf("%d_choice", id)] = c.Text()
		h.states[id] = "info"

		menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		btnSkip := menu.Text("Пропустить")
		menu.Reply(
			menu.Row(btnSkip),
		)

		return c.Send("Напиши что-нибудь о себе", menu)
	case "info":
		h.dataUser[fmt.Sprintf("%d_info", id)] = c.Text()
		h.states[id] = "photo"

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
	name := h.dataUser[fmt.Sprintf("%d_name", id)]
	age := h.dataUser[fmt.Sprintf("%d_age", id)]
	gender := h.dataUser[fmt.Sprintf("%d_gender", id)]
	choice := h.dataUser[fmt.Sprintf("%d_choice", id)]
	info := h.dataUser[fmt.Sprintf("%d_info", id)]

	// if info was skipped
	if info == "Пропустить" {
		info = ""
	}

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
	text := fmt.Sprintf("%s, %s\n\n%s", name, age, info)
	return c.Send(&telebot.Photo{
		File:    telebot.File{FileID: photo.FileID},
		Caption: text,
	})
}

func (h *BotHandlers) saveInDB(id, age int, name, gender, choice, info, photo string) error {
	_, err := h.db.Exec("INSERT INTO users (name, age, gender, fav_gen, information, photo, id_tg) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		name, age, gender, choice, info, photo, id)
	return err
}
