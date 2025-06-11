package handlers

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"dating_bot/models"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/telebot.v3"
)

type Handlers struct {
	*models.BotHandlers
}

func NewHandlers(bot *telebot.Bot, db *sql.DB) *Handlers {
	return &Handlers{
		BotHandlers: models.New(bot, db),
	}
}

func (h *Handlers) SetupHandlers() {
	h.Bot.Handle("/myprofile", h.StartHandler)
	h.Bot.Handle(telebot.OnText, h.TextHandler)
	h.Bot.Handle(telebot.OnPhoto, h.PhotoHandler)
}

func (h *Handlers) StartHandler(c telebot.Context) error {
	h.Mu.Lock()
	defer h.Mu.Unlock()

	id := c.Sender().ID

	query := "SELECT name, age, gender, fav_gen, information, photo FROM users WHERE id_tg = ?"
	row := h.Db.QueryRow(query, int(id))

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
		h.States[id] = "name"
		fmt.Println(h.States[id] + "here 1")
		return c.Send("Привет! Я бот для знакомств. Давай создадим тебе анкету\nКак тебя зовут?")
	}

	if _, err := os.Stat(photoPath); os.IsNotExist(err) {
		fmt.Println("Фото не найдено")
	}

	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	btnChange := menu.Text("Поиск")
	btnFind := menu.Text("Изменить анкету")
	menu.Reply(menu.Row(btnChange, btnFind))

	h.States[id] = "choice"
	return c.Send(&telebot.Photo{
		File:    telebot.FromDisk(photoPath),
		Caption: fmt.Sprintf("%s, %d\n\n%s", name, age, info),
	}, menu)
}

func (h *Handlers) TextHandler(c telebot.Context) error {
	h.Mu.Lock()
	defer h.Mu.Unlock()
	state := h.States[c.Sender().ID]

	id := c.Sender().ID

	switch state {
	case "name":
		h.DataUser[fmt.Sprintf("%d_name", id)] = c.Text()
		h.States[id] = "age"
		return c.Send("Сколько тебе лет?")
	case "age":
		h.DataUser[fmt.Sprintf("%d_age", id)] = c.Text()
		h.States[id] = "gender"

		menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		btnMale := menu.Text("Мужской ♂️")
		btnFemale := menu.Text("Женский ♀️")
		menu.Reply(menu.Row(btnMale, btnFemale))

		return c.Send("Укажи свой пол:", menu)
	case "gender":
		h.DataUser[fmt.Sprintf("%d_gender", id)] = c.Text()
		h.States[id] = "choice"

		menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		btnMale := menu.Text("М")
		btnFemale := menu.Text("Ж")
		btnEverybody := menu.Text("Без разницы")
		menu.Reply(menu.Row(btnMale, btnFemale, btnEverybody))

		return c.Send("Кого хочешь искать?", menu)
	case "choice":
		h.DataUser[fmt.Sprintf("%d_choice", id)] = c.Text()
		h.States[id] = "info"

		menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		btnSkip := menu.Text("Пропустить")
		menu.Reply(menu.Row(btnSkip))

		return c.Send("Напиши что-нибудь о себе", menu)
	case "info":
		h.DataUser[fmt.Sprintf("%d_info", id)] = c.Text()
		h.States[id] = "photo"

		return c.Send("Теперь отправь свое фото", &telebot.ReplyMarkup{RemoveKeyboard: true})
	default:
		return c.Send("Что-то пошло не так.../myprofile")
	}
}

func (h *Handlers) PhotoHandler(c telebot.Context) error {
	h.Mu.Lock()
	defer h.Mu.Unlock()

	if h.States[c.Sender().ID] != "photo" {
		return c.Send("Сначала заполните анкету!")
	}

	id := int(c.Sender().ID)
	name := h.DataUser[fmt.Sprintf("%d_name", id)]
	age := h.DataUser[fmt.Sprintf("%d_age", id)]
	gender := h.DataUser[fmt.Sprintf("%d_gender", id)]
	choice := h.DataUser[fmt.Sprintf("%d_choice", id)]
	info := h.DataUser[fmt.Sprintf("%d_info", id)]

	if info == "Пропустить" {
		info = ""
	}

	ageInt, err := strconv.Atoi(age)
	if err != nil {
		fmt.Println("Couldn't convert", err)
		return err
	}

	photo := c.Message().Photo
	if photo == nil {
		return c.Send("Это не фото! Попробуй ещё раз...")
	}

	file, err := h.Bot.FileByID(photo.FileID)
	if err != nil {
		fmt.Println(err)
		return err
	}

	filesDir := "./images/"
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		return err
	}

	fileName := fmt.Sprintf("%d_photo.jpg", c.Sender().ID)
	filePath := filepath.Join(filesDir, fileName)

	if err := h.Bot.Download(&file, filePath); err != nil {
		fmt.Println("File wasn't saved")
		return err
	}

	if err := h.SaveInDB(id, ageInt, name, gender, choice, info, filePath); err != nil {
		fmt.Println("Ошибка сохранения в БД:", err)
		return err
	}

	menu := &telebot.ReplyMarkup{}
	btnChange := menu.Text("Поиск")
	btnFind := menu.Text("Изменить анкету")
	menu.Reply(menu.Row(btnChange, btnFind))

	text := fmt.Sprintf("%s, %s\n\n%s", name, age, info)
	return c.Send(&telebot.Photo{
		File:    telebot.File{FileID: photo.FileID},
		Caption: text,
	}, menu)
}

func (h *Handlers) SaveInDB(id, age int, name, gender, choice, info, photo string) error {
	_, err := h.Db.Exec(
		"INSERT INTO users (name, age, gender, fav_gen, information, photo, id_tg) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		name, age, gender, choice, info, photo, id,
	)
	return err
}
