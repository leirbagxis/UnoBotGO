package main

import (
	"fmt"
	"log"

	"github.com/mymmrac/telego"
)

func cmdDesafio(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	if msg.Chat.Type != telego.ChatTypeGroup && msg.Chat.Type != telego.ChatTypeSupergroup {
		sendMessage(bot, chatID, "Use este comando em um grupo.")
		return
	}

	match := gm.NewMatch(chatID, user)
	if match == nil {
		sendMessage(bot, chatID, "Já existe um jogo ou desafio ativo neste grupo.")
		return
	}

	text := fmt.Sprintf("🎮 %s quer um desafio!\nFormato: %s",
		displayLink(user), match.formatLabel())

	sent, err := bot.SendMessage(botCtx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
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
		log.Printf("Erro ao enviar desafio: %v", err)
		gm.CancelMatch(chatID)
		return
	}

	match.MessageID = sent.MessageID
}

func handleChallengeCallback(bot *telego.Bot, query telego.CallbackQuery, chatID int64, messageID int, user *UserData) {
	match := gm.GetMatch(chatID)
	if match == nil {
		_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "Desafio não encontrado ou já encerrado.",
		})
		return
	}

	switch query.Data {
	case cbChallengeConfig:
		formatConfigMenu(bot, chatID, match)

	case cbChallengeMD1:
		match.BestOf = 1
		match.TargetWins = 1
		formatChallengeMenu(bot, chatID, match)

	case cbChallengeMD3:
		match.BestOf = 3
		match.TargetWins = 2
		formatChallengeMenu(bot, chatID, match)

	case cbChallengeMD5:
		match.BestOf = 5
		match.TargetWins = 3
		formatChallengeMenu(bot, chatID, match)

	case cbChallengeModeClassic:
		match.Mode = "classic"
		formatChallengeMenu(bot, chatID, match)

	case cbChallengeModeFast:
		match.Mode = "fast"
		formatChallengeMenu(bot, chatID, match)

	case cbChallengeModeWild:
		match.Mode = "wild"
		formatChallengeMenu(bot, chatID, match)

	case cbChallengeModeCaseiro:
		match.Mode = "caseiro"
		formatChallengeMenu(bot, chatID, match)

	case cbChallengeModeTest:
		match.Mode = "test"
		formatChallengeMenu(bot, chatID, match)

	case cbChallengeAccept:
		if user.ID == match.Challenger.ID {
			_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
				CallbackQueryID: query.ID,
				Text:            "Você não pode aceitar seu próprio desafio!",
			})
			return
		}
		match.Challenged = user
		match.State = MatchBetweenGames
		_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
		})
		sendMatchAccepted(bot, match)

	case cbMatchStart:
		if match.State != MatchBetweenGames || match.Challenged == nil {
			return
		}
		if user.ID != match.Challenger.ID && user.ID != match.Challenged.ID {
			_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
				CallbackQueryID: query.ID,
				Text:            "Apenas os jogadores do desafio podem iniciar.",
			})
			return
		}
		_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
		})
		gm.startMatchGame(bot, match)

	case cbMatchNext:
		if match.State != MatchBetweenGames {
			return
		}
		if user.ID != match.Challenger.ID && user.ID != match.Challenged.ID {
			return
		}
		_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
		})
		gm.startMatchGame(bot, match)

	case cbMatchCancel:
		gm.CancelMatch(chatID)
		_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
		})
		_, _ = bot.EditMessageText(botCtx, &telego.EditMessageTextParams{
			ChatID:    telego.ChatID{ID: chatID},
			MessageID: messageID,
			Text:      "Desafio cancelado.",
		})
	}
}
