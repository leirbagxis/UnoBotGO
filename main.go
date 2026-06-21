package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/mymmrac/telego"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Arquivo .env não encontrado, usando variáveis de ambiente")
	}

	token := os.Getenv("TOKEN")
	if token == "" {
		log.Fatal("TOKEN environment variable is required")
	}

	bot, err := telego.NewBot(token, telego.WithDefaultDebugLogger())
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	botUser, err := bot.GetMe(botCtx)
	if err != nil {
		log.Printf("Warning: could not get bot info: %v", err)
	} else {
		log.Printf("Bot started: %s (@%s)", botUser.FirstName, botUser.Username)
	}

	updates, err := bot.UpdatesViaLongPolling(botCtx, &telego.GetUpdatesParams{
		AllowedUpdates: []string{
			"message",
			"inline_query",
			"chosen_inline_result",
		},
	})
	if err != nil {
		log.Fatalf("Failed to start long polling: %v", err)
	}

	log.Println("Bot is running...")

	for update := range updates {
		if update.Message != nil {
			handleMessage(bot, *update.Message)
		}
		if update.InlineQuery != nil {
			handleInlineQuery(bot, *update.InlineQuery)
		}
		if update.ChosenInlineResult != nil {
			handleChosenInlineResult(bot, *update.ChosenInlineResult)
		}
	}
}
