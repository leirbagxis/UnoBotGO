package main

import (
	"fmt"
	"log"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

func doPlayCard(bot *telego.Bot, player *Player, resultID string) {
	log.Printf("[doPlayCard] resultID: %s, player: %s (cards in hand: %d)", resultID, player.User.FirstName, len(player.Cards))
	card := CardFromStr(resultID)
	if card == nil {
		log.Printf("[doPlayCard] CardFromStr returned nil for resultID: %s", resultID)
		return
	}

	player.Play(card)
	game := player.Game
	chatID := game.ChatID

	if game.ChoosingColor {
		sendMessage(bot, chatID, "Por favor, escolha uma cor")
	}

	if len(player.Cards) == 1 {
		msgID := sendMessage(bot, chatID, "UNO!")
		reactMessage(bot, chatID, msgID, "😱")
	}

	if len(player.Cards) == 0 {
		msgID := sendMessage(bot, chatID, fmt.Sprintf("%s venceu!", displayLink(player.User)))
		reactMessage(bot, chatID, msgID, "🎉")
		game.PlayersWon++

		if game.MatchID != 0 {
			match := gm.GetMatch(chatID)
			if match != nil {
				gm.endMatchGame(bot, match, player)
			}
			return
		}

		rankingStore.RecordWin(player.User, chatID)

		err := gm.LeaveGame(player.User, chatID)
		if err == ErrNotEnoughPlayers {
			game.Started = false
			sendMessage(bot, chatID, "Jogo encerrado!")
			gm.EndGameByGame(chatID, game)
		}
	}
}

func doDraw(bot *telego.Bot, player *Player) {
	game := player.Game
	drawCounterBefore := game.DrawCounter

	err := player.Draw()
	if err != nil {
		sendMessage(bot, game.ChatID, "Não há mais cartas no baralho.")
	}

	if (game.LastCard.Value == DrawTwo || game.LastCard.Special == DrawFour) && drawCounterBefore > 0 {
		game.Turn()
	}
}

func doSkip(bot *telego.Bot, player *Player) {
	game := player.Game
	chatID := game.ChatID
	skippedPlayer := game.CurrentPlayer
	nextPlayer := game.CurrentPlayer.Next

	if skippedPlayer.WaitingTime > 0 {
		skippedPlayer.WaitingTime -= GetTimeRemovalAfterSkip()
		if skippedPlayer.WaitingTime < 0 {
			skippedPlayer.WaitingTime = 0
		}

		err := skippedPlayer.Draw()
		if err != nil {
			log.Printf("Deck empty during skip: %v", err)
		}

		n := skippedPlayer.WaitingTime
		sendNextMessage(bot, chatID, fmt.Sprintf(
			"Tempo de espera para pular este jogador foi reduzido para %d segundos.\nPróximo jogador: %s",
			n, displayLink(nextPlayer.User)))
		log.Printf("%s foi pulado!", displayName(player.User))
		game.Turn()
	} else {
		err := gm.LeaveGame(skippedPlayer.User, chatID)
		if err == ErrNotEnoughPlayers {
			game.Started = false
		sendMessage(bot, chatID, fmt.Sprintf(
			"%s ficou sem tempo e foi removido!\nJogo encerrado.",
			displayLink(skippedPlayer.User)))
			gm.EndGameByGame(chatID, game)
		} else {
		sendNextMessage(bot, chatID, fmt.Sprintf(
			"%s ficou sem tempo e foi removido!\nPróximo jogador: %s",
			displayLink(skippedPlayer.User), displayLink(nextPlayer.User)))
		log.Printf("%s foi pulado!", displayName(player.User))
		}
	}
}

func doCallBluff(bot *telego.Bot, player *Player) {
	game := player.Game
	chatID := game.ChatID
	prevPlayer := player.Prev

	if prevPlayer.Bluffing {
		sendMessage(bot, chatID, fmt.Sprintf(
			"Blefe pego! Dando 4 cartas para %s", prevPlayer.User.FirstName))
		err := prevPlayer.Draw()
		if err != nil {
			sendMessage(bot, chatID, "Não há mais cartas no baralho.")
		}
	} else {
		game.DrawCounter += 2
		sendMessage(bot, chatID, fmt.Sprintf(
			"%s não blefou! Dando 6 cartas para %s",
			prevPlayer.User.FirstName, player.User.FirstName))
		err := player.Draw()
		if err != nil {
			sendMessage(bot, chatID, "Não há mais cartas no baralho.")
		}
	}

	game.Turn()
}

func sendMessage(bot *telego.Bot, chatID int64, text string) int {
	msg, err := bot.SendMessage(botCtx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      text,
		ParseMode: telego.ModeHTML,
	})
	if err != nil {
		log.Printf("Error sending message: %v", err)
		return 0
	}
	return msg.MessageID
}

func reactMessage(bot *telego.Bot, chatID int64, messageID int, emoji string) {
	if messageID == 0 {
		return
	}
	err := bot.SetMessageReaction(botCtx, &telego.SetMessageReactionParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageID,
		Reaction: []telego.ReactionType{
			&telego.ReactionTypeEmoji{
				Type:  telego.ReactionEmoji,
				Emoji: emoji,
			},
		},
	})
	if err != nil {
		log.Printf("Error setting reaction: %v", err)
	}
}

func sendSticker(bot *telego.Bot, chatID int64, fileID string) {
	_, err := bot.SendSticker(botCtx, &telego.SendStickerParams{
		ChatID:  telego.ChatID{ID: chatID},
		Sticker: telego.InputFile{FileID: fileID},
	})
	if err != nil {
		log.Printf("Error sending sticker: %v", err)
	}
}

func sendNextMessage(bot *telego.Bot, chatID int64, text string) {
	_, err := bot.SendMessage(botCtx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      text,
		ParseMode: telego.ModeHTML,
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					tu.InlineKeyboardButton("Suas cartas").
						WithSwitchInlineQueryCurrentChat(""),
				},
			},
		},
	})
	if err != nil {
		log.Printf("Error sending next message: %v", err)
	}
}
