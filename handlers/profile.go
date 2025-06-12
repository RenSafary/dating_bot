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

var (
	name      string
	age       int
	gender    string
	fav_gen   string
	info      string
	photoPath string
	id_tg     int
)

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

	h.States[id] = "action"

	query := "SELECT name, age, gender, information, photo FROM users WHERE id_tg = ?"
	row := h.Db.QueryRow(query, id)

	err := row.Scan(&name, &age, &gender, &info, &photoPath)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println(err)
		}
		h.States[id] = "name"
		return c.Send("Привет! Я бот для знакомств. Давай создадим тебе анкету\nКак тебя зовут?")
	}

	if _, err := os.Stat(photoPath); os.IsNotExist(err) {
		fmt.Println("Фото не найдено")
	}

	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	btnChange := menu.Text("Поиск")
	btnFind := menu.Text("Изменить анкету")
	menu.Reply(menu.Row(btnChange, btnFind))

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

	// buttons for finding profiles

	find_menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	btnLike := find_menu.Text("Like")
	btnDislike := find_menu.Text("Dislike")
	btnClose := find_menu.Text("Закончить")

	find_menu.Reply(find_menu.Row(btnDislike, btnLike, btnClose))

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
		h.States[id] = "fav_gen"

		menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		btnMale := menu.Text("М")
		btnFemale := menu.Text("Ж")
		btnEverybody := menu.Text("Без разницы")
		menu.Reply(menu.Row(btnMale, btnFemale, btnEverybody))

		return c.Send("Кого хочешь искать?", menu)
	case "fav_gen":
		h.DataUser[fmt.Sprintf("%d_fav_gen", id)] = c.Text()
		h.States[id] = "info"

		menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		btnSkip := menu.Text("Пропустить")
		menu.Reply(menu.Row(btnSkip))

		return c.Send("Напиши что-нибудь о себе", menu)
	case "info":
		h.DataUser[fmt.Sprintf("%d_info", id)] = c.Text()
		h.States[id] = "photo"

		return c.Send("Теперь отправь свое фото", &telebot.ReplyMarkup{RemoveKeyboard: true})
	case "action":
		act := c.Text()
		if act == "Поиск" {
			h.States[id] = "find"
			return c.Send("Ищем анкеты...", find_menu)
		} else if act == "Изменить анкету" {
			h.States[id] = "name"
			query := "DELETE FROM users WHERE id_tg = ?"
			_, err := h.Db.Exec(query, int(id))
			if err != nil {
				return fmt.Errorf("failed to delete user: %v", err)
			}
			photoPath := fmt.Sprintf("images/%d_photo.jpg", id)
			err = os.Remove(photoPath)
			if err != nil {
				return fmt.Errorf("Failed to delete user's photo: %v", err)
			}
			return c.Send("Как тебя зовут?", &telebot.ReplyMarkup{RemoveKeyboard: true})
		}
		return c.Send("Неизвестная команда")
	case "find":
		query_users := "SELECT name, age, gender, information, photo, id_tg FROM users WHERE id_tg != ?"
		row := h.Db.QueryRow(query_users, id)

		err := row.Scan(&name, &age, &gender, &info, &photoPath, &id_tg)
		if err != nil {
			if err == sql.ErrNoRows {
				h.States[id] = ""
				return c.Send("Упс... Анкеты закончились /myprofile")
			}
		}
		var text string
		photoPath := fmt.Sprintf("images/%d_photo.jpg", id_tg)
		if info != "" {
			text = fmt.Sprintf("%s, %d", name, age)
		} else {
			text = fmt.Sprintf("%s, %d\n\n%s", name, age, info)
		}

		return c.Send(&telebot.Photo{
			File:    telebot.FromDisk(photoPath),
			Caption: text,
		}, find_menu)
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
	name = h.DataUser[fmt.Sprintf("%d_name", id)]
	age := h.DataUser[fmt.Sprintf("%d_age", id)]
	gender = h.DataUser[fmt.Sprintf("%d_gender", id)]
	fav_gen = h.DataUser[fmt.Sprintf("%d_fav_gen", id)]
	info = h.DataUser[fmt.Sprintf("%d_info", id)]

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

	if err := h.SaveInDB(id, ageInt, name, gender, fav_gen, info, filePath); err != nil {
		fmt.Println("Ошибка сохранения в БД:", err)
		return err
	}

	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	btnChange := menu.Text("Поиск")
	btnFind := menu.Text("Изменить анкету")
	menu.Reply(menu.Row(btnChange, btnFind))

	text := fmt.Sprintf("%s, %s\n\n%s", name, age, info)
	return c.Send(&telebot.Photo{
		File:    telebot.File{FileID: photo.FileID},
		Caption: text,
	}, menu)
}

func (h *Handlers) SaveInDB(id, age int, name, gender, fav_gen, info, photo string) error {
	_, err := h.Db.Exec(
		"INSERT INTO users (name, age, gender, fav_gen, information, photo, id_tg) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		name, age, gender, fav_gen, info, photo, id,
	)
	return err
}
