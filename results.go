package main

import (
	"fmt"
	"strings"

	"github.com/mymmrac/telego"
)

func playerList(game *Game) string {
	var parts []string
	for _, p := range game.Players() {
		cardWord := "carta"
		n := len(p.Cards)
		if n != 1 {
			cardWord = "cartas"
		}
		parts = append(parts, fmt.Sprintf("%s (%d %s)", p.User.FirstName, n, cardWord))
	}
	return strings.Join(parts, " -> ")
}

func gameInfo(game *Game) *telego.InputTextMessageContent {
	text := fmt.Sprintf("Jogador atual: %s\n", displayName(game.CurrentPlayer.User))
	text += fmt.Sprintf("Última carta: %s\n", game.LastCard.Repr())
	text += fmt.Sprintf("Jogadores: %s", playerList(game))
	return &telego.InputTextMessageContent{
		MessageText: text,
	}
}

func addCard(game *Game, card *Card, results *[]telego.InlineQueryResult, canPlay bool) {
	key := card.String()
	if canPlay {
		stickerID := Stickers[key]
		if stickerID == "" {
			return
		}
		*results = append(*results, &telego.InlineQueryResultCachedSticker{
			Type:                "sticker",
			ID:                  key,
			StickerFileID:       stickerID,
			InputMessageContent: nil,
		})
	} else {
		greyID := StickersGrey[key]
		if greyID == "" {
			return
		}
		info := gameInfo(game)
		*results = append(*results, &telego.InlineQueryResultCachedSticker{
			Type:                "sticker",
			ID:                  fmt.Sprintf("grey_%s", key),
			StickerFileID:       greyID,
			InputMessageContent: info,
		})
	}
}

func addDraw(player *Player, results *[]telego.InlineQueryResult) {
	n := player.Game.DrawCounter
	if n == 0 {
		n = 1
	}
	cardWord := "carta"
	if n != 1 {
		cardWord = "cartas"
	}
	*results = append(*results, &telego.InlineQueryResultCachedSticker{
		Type:          "sticker",
		ID:            "draw",
		StickerFileID: Stickers["option_draw"],
		InputMessageContent: &telego.InputTextMessageContent{
			MessageText: fmt.Sprintf("Comprando %d %s", n, cardWord),
		},
	})
}

func addPass(results *[]telego.InlineQueryResult) {
	*results = append(*results, &telego.InlineQueryResultCachedSticker{
		Type:          "sticker",
		ID:            "pass",
		StickerFileID: Stickers["option_pass"],
		InputMessageContent: &telego.InputTextMessageContent{
			MessageText: "Passar",
		},
	})
}

func addCallBluff(results *[]telego.InlineQueryResult) {
	*results = append(*results, &telego.InlineQueryResultCachedSticker{
		Type:          "sticker",
		ID:            "call_bluff",
		StickerFileID: Stickers["option_bluff"],
		InputMessageContent: &telego.InputTextMessageContent{
			MessageText: "Estou chamando seu blefe!",
		},
	})
}

func addGameInfo(game *Game, results *[]telego.InlineQueryResult) {
	*results = append(*results, &telego.InlineQueryResultCachedSticker{
		Type:          "sticker",
		ID:            "gameinfo",
		StickerFileID: Stickers["option_info"],
		InputMessageContent: gameInfo(game),
	})
}

func addChooseColor(game *Game, results *[]telego.InlineQueryResult) {
	for _, color := range Colors {
		label := colorName(color)
		*results = append(*results, &telego.InlineQueryResultArticle{
			Type:  "article",
			ID:    color,
			Title: "Escolher Cor",
			Description: label,
			InputMessageContent: &telego.InputTextMessageContent{
				MessageText: fmt.Sprintf("%s %s", ColorIcons[color], label),
			},
		})
	}
}

func addNoGame(results *[]telego.InlineQueryResult) {
	*results = append(*results, &telego.InlineQueryResultArticle{
		Type:  "article",
		ID:    "nogame",
		Title: "Você não está jogando",
		InputMessageContent: &telego.InputTextMessageContent{
			MessageText: "Você não está jogando. Use /novo para criar um jogo ou /entrar para entrar.",
		},
	})
}

func addNotStarted(results *[]telego.InlineQueryResult) {
	*results = append(*results, &telego.InlineQueryResultArticle{
		Type:  "article",
		ID:    "nogame",
		Title: "O jogo não foi iniciado",
		InputMessageContent: &telego.InputTextMessageContent{
			MessageText: "Inicie o jogo com /start",
		},
	})
}

func addModeClassic(results *[]telego.InlineQueryResult) {
	*results = append(*results, &telego.InlineQueryResultArticle{
		Type:  "article",
		ID:    "mode_classic",
		Title: "🎻 Modo Classic",
		InputMessageContent: &telego.InputTextMessageContent{
			MessageText: "Classic 🎻",
		},
	})
}

func addModeFast(results *[]telego.InlineQueryResult) {
	*results = append(*results, &telego.InlineQueryResultArticle{
		Type:  "article",
		ID:    "mode_fast",
		Title: "🚀 Modo Sanic",
		InputMessageContent: &telego.InputTextMessageContent{
			MessageText: "Gotta go fast! 🚀",
		},
	})
}

func addModeWild(results *[]telego.InlineQueryResult) {
	*results = append(*results, &telego.InlineQueryResultArticle{
		Type:  "article",
		ID:    "mode_wild",
		Title: "🐉 Modo Wild",
		InputMessageContent: &telego.InputTextMessageContent{
			MessageText: "Into the Wild~ 🐉",
		},
	})
}

func addModeText(results *[]telego.InlineQueryResult) {
	*results = append(*results, &telego.InlineQueryResultArticle{
		Type:  "article",
		ID:    "mode_text",
		Title: "✍️ Modo Text",
		InputMessageContent: &telego.InputTextMessageContent{
			MessageText: "Text ✍️",
		},
	})
}

func colorName(color string) string {
	switch color {
	case Red:
		return "❤️ Vermelho"
	case Blue:
		return "💙 Azul"
	case Green:
		return "💚 Verde"
	case Yellow:
		return "💛 Amarelo"
	}
	return ""
}

func displayName(user *UserData) string {
	name := user.FirstName
	if user.Username != "" {
		name += " (@@" + user.Username + ")"
	}
	return name
}
