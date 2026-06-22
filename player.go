package main

import (
	"log"
	"time"
)

type Player struct {
	User        *UserData
	Cards       []*Card
	Game        *Game
	Next        *Player
	Prev        *Player
	Drew        bool
	Bluffing    bool
	TurnStarted time.Time
	WaitingTime int
}

type UserData struct {
	ID        int64
	FirstName string
	Username  string
}

func NewPlayer(game *Game, user *UserData) *Player {
	p := &Player{
		User:        user,
		Game:        game,
		Cards:       make([]*Card, 0),
		WaitingTime: GetWaitingTime(),
		TurnStarted: time.Now(),
	}
	if game.CurrentPlayer != nil {
		p.Next = game.CurrentPlayer
		p.Prev = game.CurrentPlayer.Prev
		game.CurrentPlayer.Prev.Next = p
		game.CurrentPlayer.Prev = p
	} else {
		p.Next = p
		p.Prev = p
		game.CurrentPlayer = p
	}
	return p
}

func (p *Player) DrawFirstHand() error {
	for i := 0; i < 7; i++ {
		card, err := p.Game.Deck.Draw()
		if err != nil {
			for _, c := range p.Cards {
				p.Game.Deck.Dismiss(c)
			}
			return err
		}
		p.Cards = append(p.Cards, card)
	}
	return nil
}

func (p *Player) Leave() {
	if p.Next == p {
		return
	}
	p.Next.Prev = p.Prev
	p.Prev.Next = p.Next
	p.Next = nil
	p.Prev = nil
	for _, card := range p.Cards {
		p.Game.Deck.Dismiss(card)
	}
	p.Cards = nil
}

func (p *Player) Draw() error {
	amount := p.Game.DrawCounter
	if amount == 0 {
		amount = 1
	}
	for i := 0; i < amount; i++ {
		card, err := p.Game.Deck.Draw()
		if err != nil {
			return err
		}
		p.Cards = append(p.Cards, card)
	}
	p.Game.DrawCounter = 0
	p.Drew = true
	return nil
}

func (p *Player) Play(card *Card) {
	found := false
	for i, c := range p.Cards {
		if c.Equal(card) {
			p.Cards = append(p.Cards[:i], p.Cards[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		log.Printf("[Player.Play] WARNING: Card %s NOT found in player's hand!", card.String())
	}
	p.Game.PlayCard(card)
}

func (p *Player) PlayableCards() []*Card {
	var playable []*Card
	last := p.Game.LastCard

	cards := p.Cards
	if p.Drew {
		if len(p.Cards) > 0 {
			cards = p.Cards[len(p.Cards)-1:]
		} else {
			cards = nil
		}
	}

	p.Bluffing = false
	for _, card := range cards {
		if p.cardPlayable(card) {
			playable = append(playable, card)
			if card.Color == last.Color {
				p.Bluffing = true
			}
		}
	}

	if len(p.Cards) == 1 && p.Cards[0].Special != "" {
		return nil
	}

	return playable
}

func (p *Player) cardPlayable(card *Card) bool {
	last := p.Game.LastCard
	if last == nil {
		return false
	}

	if card.Color != last.Color && card.Value != last.Value && card.Special == "" {
		log.Println("Card's color or value doesn't match")
		return false
	}

	if last.Value == DrawTwo && p.Game.DrawCounter > 0 {
		if p.Game.Mode == "caseiro" && card.Special == DrawFour {
			// caseiro: +4 pode rebater +2
		} else if card.Value != DrawTwo {
			log.Println("Player has to draw and can't counter")
			return false
		}
	}

	if last.Special == DrawFour && p.Game.DrawCounter > 0 {
		if p.Game.Mode == "caseiro" && card.Value == DrawTwo && card.Color == last.Color {
			// caseiro: +2 da cor escolhida pode ser jogado
		} else {
			log.Println("Player has to draw and can't counter")
			return false
		}
	}

	if (last.Special == Choose || last.Special == DrawFour) &&
		(card.Special == Choose || card.Special == DrawFour) {
		log.Println("Can't play colorchooser on another one")
		return false
	}

	if last.Color == "" {
		log.Println("Last card has no color")
		return false
	}

	return true
}
