package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"dating_bot/database"
	"dating_bot/handlers"

	"github.com/joho/godotenv"
	"gopkg.in/telebot.v3"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Err loading .env file")
	}

	token := os.Getenv("TOKEN")

	bot, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
		return
	}

	db := database.InitDB()

	mainHandlers := handlers.NewHandlers(bot, db)
	mainHandlers.SetupHandlers()

	/*
		findHandlers := handlers.NewHandlersF(bot, db)
		findHandlers.SetupHandlersFind()
	*/

	fmt.Println("Bot is started")
	bot.Start()
}
