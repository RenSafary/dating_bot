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

	query := "SELECT name, age, gender, fav_gen, information, photo FROM users WHERE id_tg = ?"
	row := h.Db.QueryRow(query, id)

	err := row.Scan(&name, &age, &gender, &fav_gen, &info, &photoPath)
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

	findMenu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	btnLike := findMenu.Text("❤️")
	btnDislike := findMenu.Text("👎")
	btnClose := findMenu.Text("Закончить")
	findMenu.Reply(findMenu.Row(btnDislike, btnLike, btnClose))

	if text := c.Text(); text == "❤️" || text == "👎" {
		currentProfileID, err := strconv.ParseInt(h.DataUser[fmt.Sprintf("%d_current_profile", id)], 10, 64)
		if err != nil {
			return c.Send("Ошибка, попробуйте снова /myprofile")
		}

		if text == "❤️" {
			err := h.HandleLike(id, currentProfileID)
			if err != nil {
				return c.Send("Ошибка при сохранении лайка 😢")
			}

			var isMutual bool
			err = h.Db.QueryRow(
				`SELECT mutually FROM liked_users WHERE user_id = ? AND liked_id = ?`,
				id, currentProfileID,
			).Scan(&isMutual)
			if err != nil && err != sql.ErrNoRows {
				return err
			}

			if isMutual {
				c.Send("💕 У вас взаимная симпатия! Теперь вы можете написать друг другу.")
			} else {
				c.Send("❤️ Ты лайкнул этого пользователя!")
			}
		} else {
			c.Send("👎 Дизлайк сохранен, идем дальше!")
		}

		text, photoPath, foundUserID, err := h.FindProfile(id)
		if err != nil {
			if err.Error() == "no more profiles" {
				return c.Send("Упс... Анкеты закончились /myprofile")
			}
			return err
		}

		h.DataUser[fmt.Sprintf("%d_current_profile", id)] = fmt.Sprintf("%d", foundUserID)

		return c.Send(&telebot.Photo{
			File:    telebot.FromDisk(photoPath),
			Caption: text,
		}, findMenu)
	}

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
		menu.Reply(menu.Row(btnMale, btnFemale))

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

			text, photoPath, foundUserID, err := h.FindProfile(id)
			if err != nil {
				if err.Error() == "no more profiles" {
					return c.Send("Упс... Анкеты закончились /myprofile")
				}
				return err
			}

			h.DataUser[fmt.Sprintf("%d_current_profile", id)] = fmt.Sprintf("%d", foundUserID)

			return c.Send(&telebot.Photo{
				File:    telebot.FromDisk(photoPath),
				Caption: text,
			}, findMenu)
		} else if act == "Изменить анкету" {
			h.States[id] = "name"
			_, err := h.Db.Exec("DELETE FROM users WHERE id_tg = ?", id)
			if err != nil {
				return fmt.Errorf("failed to delete user: %v", err)
			}
			photoPath := fmt.Sprintf("images/%d_photo.jpg", id)
			_ = os.Remove(photoPath)
			return c.Send("Как тебя зовут?", &telebot.ReplyMarkup{RemoveKeyboard: true})
		}
		return c.Send("Неизвестная команда")

	case "find":
		text, photoPath, foundUserID, err := h.FindProfile(id)
		if err != nil {
			if err.Error() == "no more profiles" {
				return c.Send("Упс... Анкеты закончились /myprofile")
			}
			return err
		}

		h.DataUser[fmt.Sprintf("%d_current_profile", id)] = fmt.Sprintf("%d", foundUserID)

		return c.Send(&telebot.Photo{
			File:    telebot.FromDisk(photoPath),
			Caption: text,
		}, findMenu)

	default:
		return c.Send("Что-то пошло не так... /myprofile")
	}
}

func (h *Handlers) PhotoHandler(c telebot.Context) error {
	h.Mu.Lock()
	defer h.Mu.Unlock()

	if h.States[c.Sender().ID] != "photo" {
		return c.Send("Что-то пошло не так...")
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
	if gender == "Мужской ♂️" {
		gender = "М"
	} else {
		gender = "Ж"
	}
	_, err := h.Db.Exec(
		"INSERT INTO users (name, age, gender, fav_gen, information, photo, id_tg) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		name, age, gender, fav_gen, info, photo, id,
	)
	return err
}

func (h *Handlers) FindProfile(id int64) (string, string, int64, error) {
	query := `
		SELECT u.name, u.age, u.information, u.photo, u.id_tg 
		FROM users u
		WHERE u.id_tg != ? 
		AND u.gender = ?
		AND u.age BETWEEN ? - 1 AND ? + 1
		AND NOT EXISTS (
			SELECT 1 FROM watched w
			WHERE w.user_id = ?
			AND w.another_user_id = u.id_tg
		)
		LIMIT 1`

	var name, info, photoPath string
	var age int
	var id_tg int64

	err := h.Db.QueryRow(query, id, fav_gen, id).Scan(&name, &age, &info, &photoPath, &id_tg)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", 0, fmt.Errorf("no more profiles")
		}
		return "", "", 0, fmt.Errorf("database error: %v", err)
	}

	_, err = h.Db.Exec(`
        INSERT INTO watched (user_id, another_user_id, viewed_at)
        VALUES (?, ?, CURRENT_TIMESTAMP)`,
		id, id_tg,
	)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to record view: %v", err)
	}

	var text string
	fullPhotoPath := fmt.Sprintf("images/%d_photo.jpg", id_tg)

	if info == "" {
		text = fmt.Sprintf("%s, %d", name, age)
	} else {
		text = fmt.Sprintf("%s, %d\n\n%s", name, age, info)
	}

	return text, fullPhotoPath, id_tg, nil
}

func (h *Handlers) HandleLike(userID, likedUserID int64) error {
	var mutualLike bool
	err := h.Db.QueryRow(`
        SELECT EXISTS(
            SELECT 1 FROM liked_users 
            WHERE user_id = ? AND liked_id = ?
        )`, likedUserID, userID).Scan(&mutualLike)

	if err != nil {
		return fmt.Errorf("failed to check mutual like: %v", err)
	}

	_, err = h.Db.Exec(`
        INSERT INTO liked_users (user_id, liked_id, date, watched, mutually)
        VALUES (?, ?, CURRENT_DATE, ?, ?)`,
		userID, likedUserID, false, mutualLike,
	)
	if err != nil {
		return fmt.Errorf("failed to insert like: %v", err)
	}

	if mutualLike {
		_, err = h.Db.Exec(`
            UPDATE liked_users 
            SET mutually = true
            WHERE user_id = ? AND liked_id = ?`,
			likedUserID, userID,
		)
		if err != nil {
			return fmt.Errorf("failed to update mutual like: %v", err)
		}
	}

	return nil
}

func (h *Handlers) HandleDislike(userID, dislikedUserID int64) error {
	return nil
}
