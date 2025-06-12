package handlers

import (
	"database/sql"
	"dating_bot/models"
	"fmt"
	"log"
	"time"

	"gopkg.in/telebot.v3"
)

type NotificationHandler struct {
	*models.BotHandlers
}

func NewNotificationHandlers(bot *telebot.Bot, db *sql.DB) *NotificationHandler {
	return &NotificationHandler{
		BotHandlers: models.New(bot, db),
	}
}

func (hn *NotificationHandler) RegisterHandlers() {
	hn.Bot.Handle("/notifications", hn.showNotifications)
	hn.Bot.Handle(&telebot.Btn{Text: "❤️ Лайкнуть в ответ"}, hn.handleLikeBack)
	hn.Bot.Handle(&telebot.Btn{Text: "⏭ Пропустить"}, hn.handleSkip)
	//hn.Bot.Handle(&telebot.Btn{Text: "🔔 Уведомления"}, hn.showNotificationMenu)
}

func (hn *NotificationHandler) StartNotificationService() {
	ticker := time.NewTicker(30 * time.Minute)
	go func() {
		for range ticker.C {
			hn.checkAndNotifyUsers()
		}
	}()
}

func (hn *NotificationHandler) showNotifications(c telebot.Context) error {
	hn.Mu.Lock()
	defer hn.Mu.Unlock()

	userID := c.Sender().ID

	liker, err := hn.getNextUnseenLike(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Send("💌 У вас пока нет новых уведомлений")
		}
		log.Printf("Ошибка получения уведомлений: %v", err)
		return c.Send("Произошла ошибка при загрузке уведомлений")
	}

	if err := hn.markAsSeen(liker.ID, userID); err != nil {
		log.Printf("Ошибка обновления статуса: %v", err)
	}

	photoPath := fmt.Sprintf("images/%d_photo.jpg", liker.ID)
	text := fmt.Sprintf("💘 Новый лайк!\n%s, %d", liker.Name, liker.Age)
	if liker.Info != "" {
		text += "\n\n" + liker.Info
	}

	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	btnLikeBack := menu.Text("❤️ Лайкнуть в ответ")
	btnSkip := menu.Text("⏭ Пропустить")
	menu.Reply(menu.Row(btnLikeBack, btnSkip))

	return c.Send(&telebot.Photo{
		File:    telebot.FromDisk(photoPath),
		Caption: text,
	}, menu)
}

// mutual like
func (hn *NotificationHandler) handleLikeBack(c telebot.Context) error {
	hn.Mu.Lock()
	defer hn.Mu.Unlock()

	currentUserID := c.Sender().ID

	var likedUserID int64
	err := hn.Db.QueryRow(`
		SELECT user_id FROM liked_users 
		WHERE liked_id = ? AND watched = TRUE
		ORDER BY date DESC 
		LIMIT 1`,
		currentUserID).Scan(&likedUserID)

	if err != nil {
		if err == sql.ErrNoRows {
			return c.Send("Не удалось найти пользователя для ответа")
		}
		log.Printf("Ошибка поиска пользователя: %v", err)
		return c.Send("Произошла ошибка при обработке лайка")
	}

	_, err = hn.Db.Exec(`
		UPDATE liked_users 
		SET mutually = TRUE 
		WHERE user_id = ? AND liked_id = ?`,
		likedUserID, currentUserID)
	if err != nil {
		log.Printf("Ошибка обновления взаимного лайка: %v", err)
	}

	menu := &telebot.ReplyMarkup{}
	btnMessage := menu.URL("💌 Написать сообщение", fmt.Sprintf("tg://user?id=%d", likedUserID))
	menu.Inline(menu.Row(btnMessage))

	return c.Send(
		fmt.Sprintf("💕 Вы ответили взаимностью! Можете написать [пользователю](tg://user?id=%d).", likedUserID),
		menu,
		telebot.ModeMarkdown,
	)
}

// Skip
func (hn *NotificationHandler) handleSkip(c telebot.Context) error {
	return hn.showNotifications(c)
}

func (hn *NotificationHandler) checkAndNotifyUsers() {
	users, err := hn.getUsersWithUnseenLikes()
	if err != nil {
		log.Printf("Ошибка проверки уведомлений: %v", err)
		return
	}

	for _, userID := range users {
		_, err := hn.Bot.Send(&telebot.User{ID: userID},
			"💌 У вас есть новые лайки! Нажми /notifications")
		if err != nil {
			log.Printf("Ошибка отправки уведомления пользователю %d: %v", userID, err)
		}
	}
}

type Liker struct {
	ID   int64
	Name string
	Age  int
	Info string
}

func (hn *NotificationHandler) getNextUnseenLike(userID int64) (*Liker, error) {
	var liker Liker
	err := hn.Db.QueryRow(`
		SELECT u.id_tg, u.name, u.age, u.information 
		FROM liked_users lu
		JOIN users u ON lu.user_id = u.id_tg
		WHERE lu.liked_id = ? AND lu.watched = FALSE
		LIMIT 1`, userID).Scan(&liker.ID, &liker.Name, &liker.Age, &liker.Info)

	return &liker, err
}

func (hn *NotificationHandler) markAsSeen(likerID, userID int64) error {
	_, err := hn.Db.Exec(`
		UPDATE liked_users 
		SET watched = TRUE 
		WHERE user_id = ? AND liked_id = ?`,
		likerID, userID)
	return err
}

func (hn *NotificationHandler) getUsersWithUnseenLikes() ([]int64, error) {
	var users []int64
	rows, err := hn.Db.Query(`
		SELECT DISTINCT liked_id FROM liked_users 
		WHERE watched = FALSE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err == nil {
			users = append(users, userID)
		}
	}
	return users, nil
}
