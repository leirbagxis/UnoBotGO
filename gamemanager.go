package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/mymmrac/telego"
)

type GameManager struct {
	mu            sync.Mutex
	ChatIDGames   map[int64][]*Game
	UserIDPlayers map[int64][]*Player
	UserIDCurrent map[int64]*Player
	RemindDict    map[int64]map[int64]bool
	ChatIDMatch   map[int64]*Match
	nextMatchID   int64
}

func NewGameManager() *GameManager {
	return &GameManager{
		ChatIDGames:   make(map[int64][]*Game),
		UserIDPlayers: make(map[int64][]*Player),
		UserIDCurrent: make(map[int64]*Player),
		RemindDict:    make(map[int64]map[int64]bool),
		ChatIDMatch:   make(map[int64]*Match),
	}
}

func (gm *GameManager) Lock()   { gm.mu.Lock() }
func (gm *GameManager) Unlock() { gm.mu.Unlock() }

func (gm *GameManager) NewGame(chatID int64) *Game {
	gm.Lock()
	defer gm.Unlock()

	if gm.ChatIDMatch[chatID] != nil {
		log.Printf("Blocked NewGame: chat %d has active match", chatID)
		return nil
	}

	if games, ok := gm.ChatIDGames[chatID]; ok && len(games) > 0 {
		last := games[len(games)-1]
		if !last.Started {
			log.Printf("Returning existing unstarted game in chat %d", chatID)
			return last
		}
	}

	log.Printf("Creating new game in chat %d", chatID)
	game := NewGame(chatID)

	gm.ChatIDGames[chatID] = append(gm.ChatIDGames[chatID], game)
	return game
}

func (gm *GameManager) JoinGame(user *UserData, chatID int64) error {
	gm.Lock()
	defer gm.Unlock()

	games := gm.ChatIDGames[chatID]
	if len(games) == 0 {
		return ErrNoGameInChat
	}
	game := games[len(games)-1]
	if !game.Open {
		return ErrLobbyClosed
	}

	players := gm.UserIDPlayers[user.ID]
	for _, p := range players {
		if p.Game.ChatID == chatID {
			return ErrAlreadyJoined
		}
	}

	err := gm.leaveGame(user, chatID)
	if err != nil && err != ErrNoGameInChat {
		return err
	}

	player := NewPlayer(game, user)
	if game.Started {
		if err := player.DrawFirstHand(); err != nil {
			return err
		}
	}

	gm.UserIDPlayers[user.ID] = append(gm.UserIDPlayers[user.ID], player)
	gm.UserIDCurrent[user.ID] = player
	return nil
}

func (gm *GameManager) LeaveGame(user *UserData, chatID int64) error {
	gm.Lock()
	defer gm.Unlock()
	return gm.leaveGame(user, chatID)
}

func (gm *GameManager) leaveGame(user *UserData, chatID int64) error {
	player := gm.playerForUserInChat(user, chatID)
	players := gm.UserIDPlayers[user.ID]

	if player == nil {
		games := gm.ChatIDGames[chatID]
		for _, g := range games {
			for _, p := range g.Players() {
				if p.User.ID == user.ID {
					if p == g.CurrentPlayer {
						g.Turn()
					}
					p.Leave()
					return nil
				}
			}
		}
		return ErrNoGameInChat
	}

	game := player.Game

	if player == game.CurrentPlayer {
		game.Turn()
	}

	player.Leave()

	for i, p := range players {
		if p == player {
			gm.UserIDPlayers[user.ID] = append(players[:i], players[i+1:]...)
			break
		}
	}

	if gm.UserIDCurrent[user.ID] == player {
		if len(gm.UserIDPlayers[user.ID]) > 0 {
			gm.UserIDCurrent[user.ID] = gm.UserIDPlayers[user.ID][0]
		} else {
			delete(gm.UserIDCurrent, user.ID)
			delete(gm.UserIDPlayers, user.ID)
		}
	}

	remaining := game.Players()
	if len(remaining) <= 1 {
		if len(remaining) == 1 {
			return ErrLastPlayerWin
		}
		return ErrNotEnoughPlayers
	}

	return nil
}

func (gm *GameManager) removeGamePlayers(game *Game) {
	for _, p := range game.Players() {
		userPlayers := gm.UserIDPlayers[p.User.ID]
		for i, up := range userPlayers {
			if up == p {
				gm.UserIDPlayers[p.User.ID] = append(userPlayers[:i], userPlayers[i+1:]...)
				break
			}
		}
		if len(gm.UserIDPlayers[p.User.ID]) > 0 {
			gm.UserIDCurrent[p.User.ID] = gm.UserIDPlayers[p.User.ID][0]
		} else {
			delete(gm.UserIDPlayers, p.User.ID)
			delete(gm.UserIDCurrent, p.User.ID)
		}
	}
}

func (gm *GameManager) EndGame(chatID int64, user *UserData) error {
	gm.Lock()
	defer gm.Unlock()

	player := gm.playerForUserInChat(user, chatID)
	if player == nil {
		return ErrNoGameInChat
	}
	return gm.endGame(chatID, player.Game)
}

func (gm *GameManager) EndGameByGame(chatID int64, game *Game) {
	gm.Lock()
	defer gm.Unlock()
	gm.endGame(chatID, game)
}

func (gm *GameManager) endGame(chatID int64, game *Game) error {
	game.Started = false
	gm.removeGamePlayers(game)

	games := gm.ChatIDGames[chatID]
	for i, g := range games {
		if g == game {
			gm.ChatIDGames[chatID] = append(games[:i], games[i+1:]...)
			break
		}
	}
	if len(gm.ChatIDGames[chatID]) == 0 {
		delete(gm.ChatIDGames, chatID)
	}

	return nil
}

func (gm *GameManager) NewMatch(chatID int64, challenger *UserData) *Match {
	gm.Lock()
	defer gm.Unlock()

	if gm.ChatIDMatch[chatID] != nil {
		if gm.ChatIDMatch[chatID].State == MatchWaiting {
			delete(gm.ChatIDMatch, chatID)
		} else {
			return nil
		}
	}
	games := gm.ChatIDGames[chatID]
	if len(games) > 0 && games[len(games)-1].Started {
		return nil
	}

	gm.nextMatchID++
	match := &Match{
		ID:         gm.nextMatchID,
		Challenger: challenger,
		ChatID:     chatID,
		BestOf:     3,
		TargetWins: 2,
		Mode:       GetDefaultGamemode(),
		State:      MatchWaiting,
	}
	gm.ChatIDMatch[chatID] = match
	return match
}

func (gm *GameManager) GetMatch(chatID int64) *Match {
	gm.Lock()
	defer gm.Unlock()
	return gm.ChatIDMatch[chatID]
}

func (gm *GameManager) cancelMatch(chatID int64) {
	delete(gm.ChatIDMatch, chatID)
}

func (gm *GameManager) CancelMatch(chatID int64) {
	gm.Lock()
	defer gm.Unlock()
	gm.cancelMatch(chatID)
}

func (gm *GameManager) startMatchGame(bot *telego.Bot, match *Match) {
	gm.Lock()

	game := NewGame(match.ChatID)
	game.MatchID = match.ID

	u1 := match.Challenger
	u2 := match.Challenged

	p1 := NewPlayer(game, u1)
	p2 := NewPlayer(game, u2)

	game.Mode = match.Mode
	if game.Mode == "wild" {
		game.Deck.FillWild()
	} else {
		game.Deck.FillClassic()
	}
	game.firstCard()
	game.Started = true

	err1 := p1.DrawFirstHand()
	err2 := p2.DrawFirstHand()
	if err1 != nil || err2 != nil {
		log.Printf("Erro ao distribuir cartas no match: %v, %v", err1, err2)
		gm.Unlock()
		gm.CancelMatch(match.ChatID)
		return
	}

	gm.ChatIDGames[match.ChatID] = append(gm.ChatIDGames[match.ChatID], game)
	gm.UserIDPlayers[u1.ID] = append(gm.UserIDPlayers[u1.ID], p1)
	gm.UserIDPlayers[u2.ID] = append(gm.UserIDPlayers[u2.ID], p2)
	gm.UserIDCurrent[u1.ID] = p1
	gm.UserIDCurrent[u2.ID] = p2

	match.CurrentGame = game
	match.State = MatchPlaying
	gm.Unlock()

	if game.LastCard != nil {
		sendSticker(bot, match.ChatID, Stickers[game.LastCard.String()])
	}

	firstMsg := fmt.Sprintf("Partida %d! %s vs %s\nPrimeiro jogador: %s",
		match.Wins1+match.Wins2+1,
		displayLink(u1), displayLink(u2),
		displayLink(game.CurrentPlayer.User))
	sendNextMessage(bot, match.ChatID, firstMsg)
	startPlayerCountdown(bot, game)
}

func (gm *GameManager) endMatchGame(bot *telego.Bot, match *Match, winner *Player) {
	gm.Lock()

	game := match.CurrentGame
	if game == nil {
		gm.Unlock()
		return
	}

	gm.removeGamePlayers(game)
	games := gm.ChatIDGames[match.ChatID]
	for i, g := range games {
		if g == game {
			gm.ChatIDGames[match.ChatID] = append(games[:i], games[i+1:]...)
			break
		}
	}
	if len(gm.ChatIDGames[match.ChatID]) == 0 {
		delete(gm.ChatIDGames, match.ChatID)
	}

	game.Started = false
	match.CurrentGame = nil

	if winner.User.ID == match.Challenger.ID {
		match.Wins1++
	} else {
		match.Wins2++
	}

	if match.Wins1 >= match.TargetWins || match.Wins2 >= match.TargetWins {
		if match.Wins1 >= match.TargetWins {
			match.winner = match.Challenger
		} else {
			match.winner = match.Challenged
		}
		match.State = MatchFinished
		gm.Unlock()
		other := match.Challenger
		if match.winner.ID == match.Challenger.ID {
			other = match.Challenged
		}
		rankingStore.RecordChallengeWin(match.winner, match.ChatID)
		rankingStore.RecordChallengeLoss(other, match.ChatID)
		rankingStore.UpdateHeadToHead(match.ChatID, match.winner, other, match.Wins1, match.Wins2)
		sendMatchScore(bot, match)
		msgID := sendMessage(bot, match.ChatID, fmt.Sprintf("🏆 %s venceu o %s contra %s! Placar: %d×%d",
			displayLink(match.winner), match.formatLabel(), displayLink(other), match.Wins1, match.Wins2))
		reactMessage(bot, match.ChatID, msgID, "🎉")
		gm.CancelMatch(match.ChatID)
	} else {
		match.State = MatchBetweenGames
		gm.Unlock()
		sendMatchScore(bot, match)
		total := match.Wins1 + match.Wins2 + 1
		_, _ = bot.SendMessage(botCtx, &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: match.ChatID},
			Text:      fmt.Sprintf("⚔️ %s %d × %d %s\n\nPreparar partida %d?",
				displayLink(match.Challenger), match.Wins1, match.Wins2, displayLink(match.Challenged), total),
			ParseMode: telego.ModeHTML,
			ReplyMarkup: &telego.InlineKeyboardMarkup{
				InlineKeyboard: [][]telego.InlineKeyboardButton{
					{
						{Text: "▶️ Próxima partida", CallbackData: "match_next"},
						{Text: "❌ Cancelar", CallbackData: "match_cancel"},
					},
				},
			},
		})
	}
}

func (gm *GameManager) CleanGames(chatID int64) (int, error) {
	gm.Lock()
	defer gm.Unlock()

	gm.cancelMatch(chatID)

	games := gm.ChatIDGames[chatID]
	if len(games) == 0 {
		return 0, nil
	}

	var remaining []*Game
	removed := 0
	for _, g := range games {
		if !g.Started {
			gm.removeGamePlayers(g)
			removed++
		} else {
			remaining = append(remaining, g)
		}
	}

	if removed > 0 {
		if len(remaining) == 0 {
			delete(gm.ChatIDGames, chatID)
		} else {
			gm.ChatIDGames[chatID] = remaining
		}
	}

	return removed, nil
}

func (gm *GameManager) UpdateCurrentPlayer(game *Game) {
	gm.Lock()
	defer gm.Unlock()
	if game.CurrentPlayer != nil {
		gm.UserIDCurrent[game.CurrentPlayer.User.ID] = game.CurrentPlayer
	}
}

func (gm *GameManager) GetCurrentPlayer(userID int64) *Player {
	gm.Lock()
	defer gm.Unlock()
	return gm.UserIDCurrent[userID]
}

func (gm *GameManager) GetPlayersForUser(userID int64) []*Player {
	gm.Lock()
	defer gm.Unlock()
	return gm.UserIDPlayers[userID]
}

func (gm *GameManager) PlayerForUserInChat(user *UserData, chatID int64) *Player {
	gm.Lock()
	defer gm.Unlock()
	return gm.playerForUserInChat(user, chatID)
}

func (gm *GameManager) playerForUserInChat(user *UserData, chatID int64) *Player {
	players := gm.UserIDPlayers[user.ID]
	for _, p := range players {
		if p.Game.ChatID == chatID {
			return p
		}
	}
	return nil
}
