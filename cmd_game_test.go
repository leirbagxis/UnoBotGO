package main

import (
	"testing"
	"time"
)

func TestTimeSince(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected int // approximate, we check >= 0
	}{
		{"now", time.Now(), 0},
		{"1 second ago", time.Now().Add(-1 * time.Second), 1},
		{"5 seconds ago", time.Now().Add(-5 * time.Second), 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timeSince(tt.input)
			if got < tt.expected {
				t.Errorf("timeSince() = %d, want >= %d", got, tt.expected)
			}
		})
	}
}

func TestIsAdmin(t *testing.T) {
	if isAdmin(0) {
		t.Error("isAdmin(0) should return false")
	}
	if isAdmin(12345) {
		t.Error("isAdmin(12345) should return false")
	}
}

func TestFormatRankingTextoEmpty(t *testing.T) {
	got := formatRankingTexto("Mensal", nil)
	expected := "🏆 Ranking Mensal\n\nNenhuma vitória registrada neste período."
	if got != expected {
		t.Errorf("formatRankingTexto() = %q, want %q", got, expected)
	}
}

func TestFormatRankingTextoWithData(t *testing.T) {
	ranking := []WinCount{
		{UserID: 1, FirstName: "Alice", Wins: 5},
		{UserID: 2, FirstName: "Bob", Username: "bob", Wins: 3},
	}

	got := formatRankingTexto("Diário", ranking)
	expected := "🏆 Ranking Diário\n\n1. Alice — 5 vitórias\n2. @bob — 3 vitórias\n"
	if got != expected {
		t.Errorf("formatRankingTexto() = %q, want %q", got, expected)
	}
}

func TestGetMinPlayers(t *testing.T) {
	min := GetMinPlayers()
	if min < 1 {
		t.Errorf("GetMinPlayers() = %d, want >= 1", min)
	}
}

func TestGetWaitingTime(t *testing.T) {
	wt := GetWaitingTime()
	if wt <= 0 {
		t.Errorf("GetWaitingTime() = %d, want > 0", wt)
	}
}

func TestStartText(t *testing.T) {
	text := startText()
	if text == "" {
		t.Error("startText() returned empty string")
	}
	if !contains(text, "UnoBotGO") {
		t.Error("startText() should contain 'UnoBotGO'")
	}
	if !contains(text, "@unopybot") {
		t.Error("startText() should contain '@unopybot'")
	}
	if !contains(text, "/novo") {
		t.Error("startText() should contain '/novo'")
	}
	if !contains(text, "/iniciar") {
		t.Error("startText() should contain '/iniciar'")
	}
}

func TestHelpText(t *testing.T) {
	text := helpText()
	if text == "" {
		t.Error("helpText() returned empty string")
	}
	if !contains(text, "Comandos") {
		t.Error("helpText() should contain 'Comandos'")
	}
	if !contains(text, "/novo") {
		t.Error("helpText() should contain '/novo'")
	}
	if !contains(text, "/entrar") {
		t.Error("helpText() should contain '/entrar'")
	}
	if !contains(text, "/iniciar") {
		t.Error("helpText() should contain '/iniciar'")
	}
}

func TestModesText(t *testing.T) {
	text := modesText()
	if text == "" {
		t.Error("modesText() returned empty string")
	}
	if !contains(text, "Modos") {
		t.Error("modesText() should contain 'Modos'")
	}
	if !contains(text, "Classic") {
		t.Error("modesText() should contain 'Classic'")
	}
	if !contains(text, "Sanic") {
		t.Error("modesText() should contain 'Sanic'")
	}
	if !contains(text, "Wild") {
		t.Error("modesText() should contain 'Wild'")
	}
}

func TestIsChallengeCallback(t *testing.T) {
	tests := []struct {
		data     string
		expected bool
	}{
		{"challenge_accept", true},
		{"challenge_config", true},
		{"challenge_md1", true},
		{"match_start", true},
		{"match_cancel", true},
		{"set_mode_classic", false},
		{"info_help", false},
		{"ranking_diario", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.data, func(t *testing.T) {
			got := isChallengeCallback(tt.data)
			if got != tt.expected {
				t.Errorf("isChallengeCallback(%q) = %v, want %v", tt.data, got, tt.expected)
			}
		})
	}
}

func TestIsModeCallback(t *testing.T) {
	tests := []struct {
		data     string
		expected bool
	}{
		{"set_mode_classic", true},
		{"set_mode_fast", true},
		{"set_mode_wild", true},
		{"set_mode_text", true},
		{"set_mode_caseiro", true},
		{"challenge_accept", false},
		{"info_help", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.data, func(t *testing.T) {
			got := isModeCallback(tt.data)
			if got != tt.expected {
				t.Errorf("isModeCallback(%q) = %v, want %v", tt.data, got, tt.expected)
			}
		})
	}
}

func TestIsInfoCallback(t *testing.T) {
	tests := []struct {
		data     string
		expected bool
	}{
		{"info_help", true},
		{"info_modes", true},
		{"info_ranking", true},
		{"info_rankingx1", true},
		{"info_back", true},
		{"set_mode_classic", false},
		{"challenge_accept", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.data, func(t *testing.T) {
			got := isInfoCallback(tt.data)
			if got != tt.expected {
				t.Errorf("isInfoCallback(%q) = %v, want %v", tt.data, got, tt.expected)
			}
		})
	}
}

func TestIsRankingCallback(t *testing.T) {
	tests := []struct {
		data     string
		expected bool
	}{
		{"ranking_diario", true},
		{"ranking_semanal", true},
		{"ranking_mensal", true},
		{"info_help", false},
		{"set_mode_classic", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.data, func(t *testing.T) {
			got := isRankingCallback(tt.data)
			if got != tt.expected {
				t.Errorf("isRankingCallback(%q) = %v, want %v", tt.data, got, tt.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
