package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

var gm = NewGameManager()
var botCtx = context.Background()

func handleInlineQuery(bot *telego.Bot, query telego.InlineQuery) {
	results := make([]telego.InlineQueryResult, 0)
	userID := query.From.ID
	players := gm.UserIDPlayers[userID]
	player := gm.UserIDCurrent[userID]

	if len(players) == 0 || player == nil {
		addNoGame(&results)
	} else {
		game := player.Game
		if !game.Started {
			if userID == game.Starter.ID {
				addModeClassic(&results)
				addModeFast(&results)
				addModeWild(&results)
				addModeText(&results)
			} else {
				addNotStarted(&results)
			}
		} else if userID == game.CurrentPlayer.User.ID {
			if game.ChoosingColor {
				addChooseColor(game, &results)
			} else {
				if !player.Drew {
					addDraw(player, &results)
				} else {
					addPass(&results)
				}
				if game.LastCard.Special == DrawFour && game.DrawCounter > 0 {
					addCallBluff(&results)
				}
				playable := player.PlayableCards()
				addedIDs := make(map[string]bool)
				for _, card := range player.Cards {
					key := card.String()
					if !addedIDs[key] {
						canPlay := false
						for _, pc := range playable {
							if pc.Equal(card) {
								canPlay = true
								break
							}
						}
						addCard(game, card, &results, canPlay)
						addedIDs[key] = true
					}
				}
				addGameInfo(game, &results)
			}
		} else if userID != game.CurrentPlayer.User.ID || !game.Started {
			for _, card := range player.Cards {
				addCard(game, card, &results, false)
			}
		} else {
			addGameInfo(game, &results)
		}
	}

	params := tu.InlineQuery(query.ID, results...)
	params.CacheTime = 0
	params.IsPersonal = true

	err := bot.AnswerInlineQuery(botCtx, params)
	if err != nil {
		log.Printf("Error answering inline query: %v", err)
	}
	if len(results) == 0 {
		log.Printf("Empty results for inline query")
	}
}

func handleChosenInlineResult(bot *telego.Bot, result telego.ChosenInlineResult) {
	userID := result.From.ID
	player := gm.UserIDCurrent[userID]
	if player == nil {
		return
	}
	game := player.Game

	resultID := result.ResultID
	log.Printf("Selected result: %s", resultID)

	if resultID == "hand" || resultID == "gameinfo" || resultID == "nogame" {
		return
	}

	if strings.HasPrefix(resultID, "grey_") {
		return
	}

	game.Lock()
	defer game.Unlock()

	switch {
	case resultID == "call_bluff":
		doCallBluff(bot, player)
	case resultID == "draw":
		doDraw(bot, player)
	case resultID == "pass":
		game.Turn()
	case resultID == "mode_classic":
		game.SetMode("classic")
		sendMessage(bot, game.ChatID, "Modo alterado para Classic 🎻")
		return
	case resultID == "mode_fast":
		game.SetMode("fast")
		sendMessage(bot, game.ChatID, "Modo alterado para Sanic 🚀")
		return
	case resultID == "mode_wild":
		game.SetMode("wild")
		sendMessage(bot, game.ChatID, "Modo alterado para Wild 🐉")
		return
	case resultID == "mode_text":
		game.SetMode("text")
		sendMessage(bot, game.ChatID, "Modo alterado para Text ✍️")
		return
	default:
		for _, color := range Colors {
			if resultID == color {
				game.ChooseColor(color)
				goto afterAction
			}
		}
		doPlayCard(bot, player, resultID)
	}

afterAction:
	if game.Started && game.CurrentPlayer != nil {
		nextMsg := fmt.Sprintf("Próximo jogador: %s", displayName(game.CurrentPlayer.User))
		sendNextMessage(bot, game.ChatID, nextMsg)
	}
}

func startPlayerCountdown(bot *telego.Bot, game *Game) {
	player := game.CurrentPlayer
	waitSec := player.WaitingTime

	if waitSec < GetMinFastTurnTime() {
		waitSec = GetMinFastTurnTime()
	}

	if game.Mode == "fast" {
		go func() {
			time.Sleep(time.Duration(waitSec) * time.Second)
			game.Lock()
			defer game.Unlock()
			if game.Started && game.CurrentPlayer == player {
				doSkip(bot, player)
				if game.Started && game.CurrentPlayer != nil {
					nextMsg := fmt.Sprintf("Próximo jogador: %s", displayName(game.CurrentPlayer.User))
					sendMessage(bot, game.ChatID, nextMsg)
				}
			}
		}()
	}
}
