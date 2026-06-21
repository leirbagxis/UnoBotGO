package main

import (
	"log"
	"sync"
)

type GameManager struct {
	mu            sync.Mutex
	ChatIDGames   map[int64][]*Game
	UserIDPlayers map[int64][]*Player
	UserIDCurrent map[int64]*Player
	RemindDict    map[int64]map[int64]bool
}

func NewGameManager() *GameManager {
	return &GameManager{
		ChatIDGames:   make(map[int64][]*Game),
		UserIDPlayers: make(map[int64][]*Player),
		UserIDCurrent: make(map[int64]*Player),
		RemindDict:    make(map[int64]map[int64]bool),
	}
}

func (gm *GameManager) Lock()   { gm.mu.Lock() }
func (gm *GameManager) Unlock() { gm.mu.Unlock() }

func (gm *GameManager) NewGame(chatID int64) *Game {
	gm.Lock()
	defer gm.Unlock()

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
	allPlayers := game.Players()
	if len(allPlayers) < 3 {
		return ErrNotEnoughPlayers
	}

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

	return nil
}

func (gm *GameManager) EndGame(chatID int64, user *UserData) error {
	gm.Lock()
	defer gm.Unlock()

	player := gm.playerForUserInChat(user, chatID)
	if player == nil {
		return ErrNoGameInChat
	}
	game := player.Game

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
