package main

import "errors"

var (
	ErrNoGameInChat      = errors.New("no game in this chat")
	ErrAlreadyJoined     = errors.New("already joined")
	ErrLobbyClosed       = errors.New("lobby is closed")
	ErrNotEnoughPlayers  = errors.New("not enough players")
	ErrLastPlayerWin     = errors.New("last player wins")
	ErrDeckEmpty         = errors.New("deck is empty")
)
