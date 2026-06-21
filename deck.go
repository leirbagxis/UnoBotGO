package main

import "math/rand"

type Deck struct {
	Cards     []*Card
	Graveyard []*Card
}

func NewDeck() *Deck {
	return &Deck{}
}

func (d *Deck) Shuffle() {
	rand.Shuffle(len(d.Cards), func(i, j int) {
		d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i]
	})
}

func (d *Deck) Draw() (*Card, error) {
	if len(d.Cards) == 0 {
		if len(d.Graveyard) == 0 {
			return nil, ErrDeckEmpty
		}
		d.Cards = append(d.Cards, d.Graveyard...)
		d.Graveyard = nil
		d.Shuffle()
	}
	card := d.Cards[len(d.Cards)-1]
	d.Cards = d.Cards[:len(d.Cards)-1]
	return card, nil
}

func (d *Deck) Dismiss(card *Card) {
	if card.Special != "" {
		card.Color = ""
	}
	d.Graveyard = append(d.Graveyard, card)
}

func (d *Deck) FillClassic() {
	d.Cards = nil
	d.Graveyard = nil
	for _, color := range Colors {
		for _, value := range Values {
			d.Cards = append(d.Cards, &Card{Color: color, Value: value})
			if value != Zero {
				d.Cards = append(d.Cards, &Card{Color: color, Value: value})
			}
		}
	}
	for _, special := range Specials {
		for i := 0; i < 4; i++ {
			d.Cards = append(d.Cards, &Card{Special: special})
		}
	}
	d.Shuffle()
}

func (d *Deck) FillWild() {
	d.Cards = nil
	d.Graveyard = nil
	for _, color := range Colors {
		for _, value := range WildValues {
			for i := 0; i < 4; i++ {
				d.Cards = append(d.Cards, &Card{Color: color, Value: value})
			}
		}
	}
	for _, special := range Specials {
		for i := 0; i < 6; i++ {
			d.Cards = append(d.Cards, &Card{Special: special})
		}
	}
	d.Shuffle()
}
