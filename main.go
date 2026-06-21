package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/mymmrac/telego"
	_ "github.com/lib/pq"
)

var botToken string

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Arquivo .env não encontrado, usando variáveis de ambiente")
	}

	botToken = os.Getenv("TOKEN")
	if botToken == "" {
		log.Fatal("TOKEN environment variable is required")
	}

	db, err := sql.Open("postgres", GetDatabaseURL())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Conectado ao PostgreSQL")
	defer db.Close()

	bot, err := telego.NewBot(botToken, telego.WithDefaultDebugLogger())
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	botUser, err := bot.GetMe(botCtx)
	if err != nil {
		log.Printf("Warning: could not get bot info: %v", err)
	} else {
		log.Printf("Bot started: %s (@%s)", botUser.FirstName, botUser.Username)
	}

	rankingStore = NewRankingStore(db)

	err = bot.SetMyCommands(botCtx, &telego.SetMyCommandsParams{
		Commands: []telego.BotCommand{
			{Command: "novo", Description: "Criar novo jogo"},
			{Command: "entrar", Description: "Entrar no jogo"},
			{Command: "start", Description: "Iniciar o jogo"},
			{Command: "sair", Description: "Sair do jogo"},
			{Command: "fechar", Description: "Fechar lobby"},
			{Command: "abrir", Description: "Abrir lobby"},
			{Command: "kill", Description: "Encerrar jogo"},
			{Command: "pular", Description: "Pular jogador atual"},
			{Command: "kick", Description: "Expulsar jogador"},
			{Command: "limpar", Description: "Limpar jogos não iniciados"},
			{Command: "notificar", Description: "Notificar quando novo jogo começar"},
			{Command: "ajuda", Description: "Ajuda"},
			{Command: "modos", Description: "Modos de jogo"},
			{Command: "ranking", Description: "Ranking mensal"},
			{Command: "diario", Description: "Ranking diário"},
			{Command: "semanal", Description: "Ranking semanal"},
		},
	})
	if err != nil {
		log.Printf("Warning: could not set bot commands: %v", err)
	}

	updates, err := bot.UpdatesViaLongPolling(botCtx, &telego.GetUpdatesParams{
		AllowedUpdates: []string{
			"message",
			"inline_query",
			"chosen_inline_result",
			"callback_query",
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
		if update.CallbackQuery != nil {
			handleCallbackQuery(bot, *update.CallbackQuery)
		}
	}
}
