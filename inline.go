package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
)

var gm = NewGameManager()
var botCtx = context.Background()

type rawAnswerInlineQuery struct {
	InlineQueryID string                    `json:"inline_query_id"`
	Results       []telego.InlineQueryResult `json:"results"`
	CacheTime     int                       `json:"cache_time"`
	IsPersonal    bool                      `json:"is_personal,omitempty"`
}

func sendAnswerInlineQuery(bot *telego.Bot, params rawAnswerInlineQuery) error {
	body, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	log.Printf("[sendAnswerInlineQuery] Sending answerInlineQuery with cache_time=0, results=%d", len(params.Results))

	url := fmt.Sprintf("https://api.telegram.org/bot%s/answerInlineQuery", botToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[sendAnswerInlineQuery] Telegram response (status=%d): %s", resp.StatusCode, string(respBody))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api status: %s, body: %s", resp.Status, string(respBody))
	}

	// Check ok:true in response body
	var tgResp struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(respBody, &tgResp); err == nil && !tgResp.OK {
		return fmt.Errorf("telegram returned ok:false, body: %s", string(respBody))
	}

	return nil
}

func handleInlineQuery(bot *telego.Bot, query telego.InlineQuery) {
	results := make([]telego.InlineQueryResult, 0)
	userID := query.From.ID
	players := gm.GetPlayersForUser(userID)
	player := gm.GetCurrentPlayer(userID)

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
				addChooseColor(&results)
				addPlayerCards(game, player, &results)
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
				for _, card := range sortedCards(player.Cards) {
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
			for _, card := range sortedCards(player.Cards) {
				addCard(game, card, &results, false)
			}
		} else {
			addGameInfo(game, &results)
		}

	for _, res := range results {
		uniqueID := time.Now().UnixNano()
		switch r := res.(type) {
		case *telego.InlineQueryResultCachedSticker:
			r.ID = fmt.Sprintf("%s:%d:%d", r.ID, uniqueID, player.AntiCheat)
		case *telego.InlineQueryResultArticle:
			r.ID = fmt.Sprintf("%s:%d:%d", r.ID, uniqueID, player.AntiCheat)
		}
	}
	}

	params := rawAnswerInlineQuery{
		InlineQueryID: query.ID,
		Results:       results,
		CacheTime:     0,
		IsPersonal:    true,
	}

	err := sendAnswerInlineQuery(bot, params)
	if err != nil {
		log.Printf("Error answering inline query: %v", err)
	}
	if len(results) == 0 {
		log.Printf("Empty results for inline query")
	}
}

func handleChosenInlineResult(bot *telego.Bot, result telego.ChosenInlineResult) {
	userID := result.From.ID
	log.Printf("[ChosenInlineResult] Received ChosenInlineResult ID: %s for userID: %d, query: %s", result.ResultID, userID, result.Query)

	player := gm.GetCurrentPlayer(userID)
	if player == nil {
		log.Printf("[ChosenInlineResult] Player is NIL for userID: %d", userID)
		return
	}
	game := player.Game

	parts := strings.SplitN(result.ResultID, ":", 3)
	if len(parts) < 2 {
		log.Printf("[ChosenInlineResult] Invalid resultID format: %s", result.ResultID)
		return
	}
	resultID := parts[0]
	antiCheatStr := parts[len(parts)-1]

	antiCheatVal, err := strconv.Atoi(antiCheatStr)
	if err != nil {
		log.Printf("[ChosenInlineResult] Error parsing antiCheatStr: %v", err)
		return
	}

	if antiCheatVal != player.AntiCheat {
		log.Printf("[ChosenInlineResult] Cheat attempt / obsolete action by %s! Got: %d, expected: %d", player.User.FirstName, antiCheatVal, player.AntiCheat)
		// Se for carta e ainda está na mão, tolera anti-cheat defasado (cache do Telegram)
		if resultID != "draw" && resultID != "pass" && resultID != "call_bluff" &&
			!strings.HasPrefix(resultID, "mode_") && player.HasCard(resultID) {
			log.Printf("[ChosenInlineResult] Tolerating stale anti-cheat for card %s (still in hand)", resultID)
		} else {
			sendMessage(bot, player.Game.ChatID, "Ação expirada! Toque em 'Suas cartas' antes de jogar.")
			return
		}
	}

	player.AntiCheat++
	log.Printf("[ChosenInlineResult] Valid action! Player: %s for chatID: %d. Processing resultID: %s, next anti-cheat count: %d", player.User.FirstName, game.ChatID, resultID, player.AntiCheat)
	log.Printf("Selected result: %s", resultID)

	if resultID == "hand" || resultID == "gameinfo" || resultID == "nogame" {
		return
	}

	if strings.HasPrefix(resultID, "grey_") {
		return
	}

	game.Lock()

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
		game.Unlock()
		return
	case resultID == "mode_fast":
		game.SetMode("fast")
		sendMessage(bot, game.ChatID, "Modo alterado para Sanic 🚀")
		game.Unlock()
		return
	case resultID == "mode_wild":
		game.SetMode("wild")
		sendMessage(bot, game.ChatID, "Modo alterado para Wild 🐉")
		game.Unlock()
		return
	case resultID == "mode_text":
		game.SetMode("text")
		sendMessage(bot, game.ChatID, "Modo alterado para Text ✍️")
		game.Unlock()
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
	started := game.Started
	var nextPlayerUser *UserData
	if game.CurrentPlayer != nil {
		nextPlayerUser = game.CurrentPlayer.User
	}
	game.Unlock()

	if started && nextPlayerUser != nil {
		gm.UpdateCurrentPlayer(game)
		nextMsg := fmt.Sprintf("Próximo jogador: %s", displayName(nextPlayerUser))
		sendNextMessage(bot, game.ChatID, nextMsg)
	}
}

func startPlayerCountdown(bot *telego.Bot, game *Game) {
	if game.Mode != "fast" {
		return
	}

	minTime := GetMinFastTurnTime()

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for range ticker.C {
			game.Lock()
			if !game.Started {
				game.Unlock()
				return
			}

			player := game.CurrentPlayer
			delta := int(time.Since(player.TurnStarted).Seconds())

			wait := player.WaitingTime
			if wait < 0 {
				wait = 0
			}
			if wait > 0 && wait < minTime {
				wait = minTime
			}

			if delta >= wait {
				game.Unlock()
				doSkip(bot, player)
			} else {
				game.Unlock()
			}
		}
	}()
}
