package main

import (
	"fmt"
	"log"

	"github.com/mymmrac/telego"
)

func cmdRanking(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	if msg.Chat.Type == telego.ChatTypePrivate {
		showUserRankingAcrossGroups(bot, msg, user, chatID)
		return
	}

	ranking := rankingStore.GetRanking(chatID, "mensal")
	text := formatRankingTexto("Mensal", ranking)

	_, err := bot.SendMessage(botCtx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   text,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: msg.MessageID,
		},
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "📅 Diário", CallbackData: cbRankingDiario},
					{Text: "📆 Semanal", CallbackData: cbRankingSemanal},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Erro ao enviar ranking: %v", err)
	}
}

func cmdRankingDiario(bot *telego.Bot, msg telego.Message, chatID int64) {
	ranking := rankingStore.GetRanking(chatID, "diario")
	text := formatRankingTexto("Diário", ranking)

	_, err := bot.SendMessage(botCtx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   text,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: msg.MessageID,
		},
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "📅 Mensal", CallbackData: cbRankingMensal},
					{Text: "📆 Semanal", CallbackData: cbRankingSemanal},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Erro ao enviar ranking: %v", err)
	}
}

func cmdRankingSemanal(bot *telego.Bot, msg telego.Message, chatID int64) {
	ranking := rankingStore.GetRanking(chatID, "semanal")
	text := formatRankingTexto("Semanal", ranking)

	_, err := bot.SendMessage(botCtx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   text,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: msg.MessageID,
		},
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "📅 Diário", CallbackData: cbRankingDiario},
					{Text: "📆 Mensal", CallbackData: cbRankingMensal},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Erro ao enviar ranking: %v", err)
	}
}

func cmdRankingX1(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	if msg.Chat.Type == telego.ChatTypePrivate {
		sendMessage(bot, chatID, "Use este comando em um grupo.")
		return
	}

	list := rankingStore.GetChallengeRanking(chatID, user.ID)

	if len(list) == 0 {
		sendMessage(bot, chatID, "Nenhum confronto registrado para você neste grupo.")
		return
	}

	text := fmt.Sprintf("Confrontos de %s:\n\n", displayLink(user))
	for _, h := range list {
		opUser := &UserData{
			ID:        h.OpponentID,
			FirstName: h.OpponentName,
			Username:  h.OpponentUsername,
		}
		text += fmt.Sprintf("vs %s — %dV %dD\n", displayLink(opUser), h.MyWins, h.MyLosses)
	}

	_, err := bot.SendMessage(botCtx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      text,
		ParseMode: telego.ModeHTML,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: msg.MessageID,
		},
	})
	if err != nil {
		log.Printf("Erro ao enviar ranking x1: %v", err)
	}
}

func showUserRankingAcrossGroups(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	rankings := rankingStore.GetUserRankingAcrossGroups(user.ID)
	if len(rankings) == 0 {
		sendMessage(bot, chatID, "Você ainda não venceu nenhum jogo.")
		return
	}

	text := "🏆 Sua posição nos grupos:\n\n"
	for _, r := range rankings {
		name := r.GroupName
		if name == "" {
			name = fmt.Sprintf("Grupo %d", r.ChatID)
		}
		text += fmt.Sprintf("#%d — %s (%d vitórias)\n", r.Rank, name, r.Wins)
	}

	_, err := bot.SendMessage(botCtx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   text,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: msg.MessageID,
		},
	})
	if err != nil {
		log.Printf("Erro ao enviar ranking do usuário: %v", err)
	}
}

func formatRankingTexto(periodo string, ranking []WinCount) string {
	if len(ranking) == 0 {
		return fmt.Sprintf("🏆 Ranking %s\n\nNenhuma vitória registrada neste período.", periodo)
	}

	text := fmt.Sprintf("🏆 Ranking %s\n\n", periodo)
	for i, r := range ranking {
		nome := r.FirstName
		if r.Username != "" {
			nome = "@" + r.Username
		}
		text += fmt.Sprintf("%d. %s — %d vitórias\n", i+1, nome, r.Wins)
	}
	return text
}

func showDailyRanking(bot *telego.Bot, chatID int64) {
	ranking := rankingStore.GetRanking(chatID, "diario")
	text := formatRankingTexto("Diário", ranking)
	_, err := bot.SendMessage(botCtx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   text,
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "📅 Mensal", CallbackData: cbRankingMensal},
					{Text: "📆 Semanal", CallbackData: cbRankingSemanal},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Erro ao enviar ranking diário: %v", err)
	}
}

func formatRankingX1Texto(user *UserData, list []HeadToHead) string {
	if len(list) == 0 {
		return "Nenhum confronto registrado para você neste grupo."
	}

	text := fmt.Sprintf("⚔️ Confrontos de %s:\n\n", displayLink(user))
	for _, h := range list {
		opUser := &UserData{
			ID:        h.OpponentID,
			FirstName: h.OpponentName,
			Username:  h.OpponentUsername,
		}
		text += fmt.Sprintf("vs %s — %dV %dD\n", displayLink(opUser), h.MyWins, h.MyLosses)
	}
	return text
}
