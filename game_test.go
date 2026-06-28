package main

import (
	"testing"
)

func TestNewGame(t *testing.T) {
	chatID := int64(-100123456)
	game := NewGame(chatID, "Test Group")
	if game == nil {
		t.Fatal("NewGame() returned nil")
	}
	if game.ChatID != chatID {
		t.Errorf("game.ChatID = %d, want %d", game.ChatID, chatID)
	}
	if game.GroupName != "Test Group" {
		t.Errorf("game.GroupName = %q, want %q", game.GroupName, "Test Group")
	}
	if !game.Open {
		t.Error("new game should be open")
	}
	if game.Started {
		t.Error("new game should not be started")
	}
}

func TestGamePlayers(t *testing.T) {
	game := NewGame(-100123456, "Test Group")

	players := game.Players()
	if players != nil {
		t.Errorf("empty game should return nil, got %d players", len(players))
	}

	u1 := &UserData{ID: 1, FirstName: "Alice"}
	u2 := &UserData{ID: 2, FirstName: "Bob"}

	p1 := NewPlayer(game, u1)
	p2 := NewPlayer(game, u2)

	players = game.Players()
	if len(players) != 2 {
		t.Errorf("expected 2 players, got %d", len(players))
	}
	if players[0].User.ID != 1 || players[1].User.ID != 2 {
		t.Errorf("unexpected player order")
	}

	_ = p1
	_ = p2
}

func TestGameMinPlayers(t *testing.T) {
	game := NewGame(-100123456, "Test Group")

	// With 1 player, should be less than minimum
	u1 := &UserData{ID: 1, FirstName: "Alice"}
	NewPlayer(game, u1)

	if len(game.Players()) >= GetMinPlayers() {
		t.Errorf("expected less than %d players, got %d", GetMinPlayers(), len(game.Players()))
	}
}

func TestSortedCards(t *testing.T) {
	cards := []*Card{
		{Color: "y", Value: "1"},
		{Color: "r", Value: "3"},
		{Color: "b", Value: "2"},
		{Color: "g", Value: "skip"},
	}

	sorted := sortedCards(cards)
	if len(sorted) != 4 {
		t.Fatalf("expected 4 cards, got %d", len(sorted))
	}

	// Sort order: red (r=0), blue (b=1), green (g=2), yellow (y=3)
	if sorted[0].Color != "r" {
		t.Errorf("first card should be red, got %s", sorted[0].Color)
	}
	// Second should be blue (b=1)
	if sorted[1].Color != "b" {
		t.Errorf("second card should be blue, got %s", sorted[1].Color)
	}
}

func TestCardEqual(t *testing.T) {
	c1 := &Card{Color: "r", Value: "5"}
	c2 := &Card{Color: "r", Value: "5"}
	c3 := &Card{Color: "b", Value: "5"}

	if !c1.Equal(c2) {
		t.Error("identical cards should be equal")
	}
	if c1.Equal(c3) {
		t.Error("different color cards should not be equal")
	}
}
