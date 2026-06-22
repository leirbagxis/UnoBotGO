package main

import (
	"fmt"
	"log"

	"github.com/mymmrac/telego"
)

type MatchState int

const (
	MatchWaiting MatchState = iota
	MatchPlaying
	MatchBetweenGames
	MatchFinished
)

type Match struct {
	ID            int64
	Challenger    *UserData
	Challenged    *UserData
	ChatID        int64
	BestOf        int
	TargetWins    int
	Wins1         int
	Wins2         int
	State         MatchState
	MessageID     int
	CurrentGame   *Game
	winner        *UserData
	configMessage bool
}

func (m *Match) formatLabel() string {
	s := fmt.Sprintf("MD%d", m.BestOf)
	return s
}

func (m *Match) winsNeeded() int {
	return m.TargetWins
}

func (m *Match) winnerName() string {
	if m.winner != nil {
		return displayLink(m.winner)
	}
	return ""
}

func formatChallengeMenu(bot *telego.Bot, chatID int64, match *Match) {
	text := fmt.Sprintf("🎮 %s quer um desafio!\nFormato: %s",
		displayLink(match.Challenger), match.formatLabel())

	_, err := bot.EditMessageText(botCtx, &telego.EditMessageTextParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: match.MessageID,
		Text:      text,
		ParseMode: telego.ModeHTML,
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "✅ Aceitar", CallbackData: "challenge_accept"},
					{Text: "⚙️ Configurar", CallbackData: "challenge_config"},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Erro ao editar mensagem de desafio: %v", err)
	}
}

func formatConfigMenu(bot *telego.Bot, chatID int64, match *Match) {
	_, err := bot.EditMessageText(botCtx, &telego.EditMessageTextParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: match.MessageID,
		Text:      "Selecione o formato:",
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "MD1", CallbackData: "challenge_md1"},
					{Text: "MD3", CallbackData: "challenge_md3"},
					{Text: "MD5", CallbackData: "challenge_md5"},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Erro ao editar menu de config: %v", err)
	}
}

func sendMatchAccepted(bot *telego.Bot, match *Match) {
	text := fmt.Sprintf("✅ %s aceitou o %s contra %s!\nPlacar: 0×0\n\nPreparar partida 1?",
		displayLink(match.Challenged), match.formatLabel(), displayLink(match.Challenger))

	_, err := bot.EditMessageText(botCtx, &telego.EditMessageTextParams{
		ChatID:    telego.ChatID{ID: match.ChatID},
		MessageID: match.MessageID,
		Text:      text,
		ParseMode: telego.ModeHTML,
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "▶️ Começar!", CallbackData: "match_start"},
					{Text: "❌ Cancelar", CallbackData: "match_cancel"},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Erro ao editar mensagem de aceite: %v", err)
	}
}

func sendMatchScore(bot *telego.Bot, match *Match) {
	var text string
	if match.winner != nil {
		text = fmt.Sprintf("🏆 %s venceu o %s!\nPlacar final: %d×%d",
			displayLink(match.winner), match.formatLabel(), match.Wins1, match.Wins2)
	} else {
		p1Name := displayLink(match.Challenger)
		p2Name := displayLink(match.Challenged)
		total := match.Wins1 + match.Wins2 + 1
		text = fmt.Sprintf("%s %d × %d %s\n\nPreparar partida %d?",
			p1Name, match.Wins1, match.Wins2, p2Name, total)
	}

	var keyboard [][]telego.InlineKeyboardButton
	if match.winner == nil {
		keyboard = [][]telego.InlineKeyboardButton{
			{
				{Text: "▶️ Próxima partida", CallbackData: "match_next"},
				{Text: "❌ Cancelar", CallbackData: "match_cancel"},
			},
		}
	}

	_, err := bot.EditMessageText(botCtx, &telego.EditMessageTextParams{
		ChatID:    telego.ChatID{ID: match.ChatID},
		MessageID: match.MessageID,
		Text:      text,
		ParseMode: telego.ModeHTML,
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})
	if err != nil {
		log.Printf("Erro ao editar mensagem de placar: %v", err)
	}
}
