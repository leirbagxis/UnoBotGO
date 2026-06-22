package main

import (
	"sync"
	"time"
)

type Game struct {
	mu            sync.Mutex
	ChatID        int64
	Deck          *Deck
	LastCard      *Card
	CurrentPlayer *Player
	Reversed      bool
	ChoosingColor bool
	Started       bool
	DrawCounter   int
	PlayersWon    int
	Starter       *UserData
	Owner         []int64
	Mode          string
	Open          bool
	MatchID       int64
}

func NewGame(chatID int64) *Game {
	return &Game{
		ChatID: chatID,
		Deck:   NewDeck(),
		Open:   true,
		Mode:   GetDefaultGamemode(),
		Owner:  make([]int64, 0),
	}
}

func (g *Game) Players() []*Player {
	if g.CurrentPlayer == nil {
		return nil
	}
	var players []*Player
	current := g.CurrentPlayer
	players = append(players, current)
	var it *Player
	if g.Reversed {
		it = current.Prev
	} else {
		it = current.Next
	}
	for it != nil && it != current {
		players = append(players, it)
		if g.Reversed {
			it = it.Prev
		} else {
			it = it.Next
		}
	}
	return players
}

func (g *Game) SetMode(mode string) {
	g.Mode = mode
}

func (g *Game) Lock() {
	g.mu.Lock()
}

func (g *Game) Unlock() {
	g.mu.Unlock()
}

func (g *Game) Start() {
	if g.Mode != "wild" {
		g.Deck.FillClassic()
	} else {
		g.Deck.FillWild()
	}
	g.firstCard()
	g.Started = true
}

func (g *Game) firstCard() {
	if len(g.Deck.Cards) == 0 {
		g.SetMode(GetDefaultGamemode())
	}
	for {
		card, err := g.Deck.Draw()
		if err != nil {
			g.SetMode(GetDefaultGamemode())
			continue
		}
		if card.Special == "" {
			g.LastCard = card
			break
		}
		g.Deck.Dismiss(card)
	}
	g.PlayCard(g.LastCard)
}

func (g *Game) Reverse() {
	g.Reversed = !g.Reversed
}

func (g *Game) Turn() {
	if g.Reversed {
		g.CurrentPlayer = g.CurrentPlayer.Prev
	} else {
		g.CurrentPlayer = g.CurrentPlayer.Next
	}
	g.CurrentPlayer.Drew = false
	g.CurrentPlayer.TurnStarted = time.Now()
	g.ChoosingColor = false
}

func (g *Game) PlayCard(card *Card) {
	if g.LastCard != nil {
		g.Deck.Dismiss(g.LastCard)
	}
	g.LastCard = card

	if card.Value == Skip {
		g.Turn()
	} else if card.Special == DrawFour {
		g.DrawCounter += 4
	} else if card.Value == DrawTwo {
		g.DrawCounter += 2
	} else if card.Value == Reverse {
		players := g.Players()
		if len(players) >= 2 && g.CurrentPlayer.Next != nil &&
			g.CurrentPlayer.Next.Next == g.CurrentPlayer {
			g.Turn()
		} else {
			g.Reverse()
		}
	}

	if card.Special != Choose && card.Special != DrawFour {
		g.Turn()
	} else {
		g.ChoosingColor = true
	}
}

func (g *Game) ChooseColor(color string) {
	g.LastCard.Color = color
	g.Turn()
}
