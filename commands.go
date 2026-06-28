package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/mymmrac/telego"
)

// Callback data constants
const (
	cbChallengeAccept     = "challenge_accept"
	cbChallengeConfig     = "challenge_config"
	cbChallengeMD1        = "challenge_md1"
	cbChallengeMD3        = "challenge_md3"
	cbChallengeMD5        = "challenge_md5"
	cbChallengeModeClassic = "challenge_mode_classic"
	cbChallengeModeFast   = "challenge_mode_fast"
	cbChallengeModeWild   = "challenge_mode_wild"
	cbChallengeModeCaseiro = "challenge_mode_caseiro"
	cbChallengeModeTest   = "challenge_mode_test"
	cbMatchStart          = "match_start"
	cbMatchNext           = "match_next"
	cbMatchCancel         = "match_cancel"

	cbSetModeClassic = "set_mode_classic"
	cbSetModeFast    = "set_mode_fast"
	cbSetModeWild    = "set_mode_wild"
	cbSetModeText    = "set_mode_text"
	cbSetModeCaseiro = "set_mode_caseiro"

	cbInfoHelp      = "info_help"
	cbInfoModes     = "info_modes"
	cbInfoRanking   = "info_ranking"
	cbInfoRankingX1 = "info_rankingx1"
	cbInfoBack      = "info_back"

	cbRankingDiario  = "ranking_diario"
	cbRankingSemanal = "ranking_semanal"
	cbRankingMensal  = "ranking_mensal"
)

func handleMessage(bot *telego.Bot, message telego.Message) {
	if message.Text == "" {
		return
	}
	text := message.Text
	chatID := message.Chat.ID
	user := &UserData{
		ID:        message.From.ID,
		FirstName: message.From.FirstName,
		Username:  message.From.Username,
	}

	gm.Lock()
	if players, ok := gm.UserIDPlayers[user.ID]; ok {
		for _, p := range players {
			if p.Game.ChatID == chatID {
				gm.UserIDCurrent[user.ID] = p
				break
			}
		}
	}
	gm.Unlock()

	if !strings.HasPrefix(text, "/") {
		return
	}

	parts := strings.Fields(text)
	command := parts[0]

	if strings.Contains(command, "@") {
		cmdParts := strings.SplitN(command, "@", 2)
		command = cmdParts[0]
	}

	switch command {
	case "/novo":
		cmdNewGame(bot, message, user, chatID)
	case "/entrar":
		cmdJoinGame(bot, message, user, chatID)
	case "/start":
		cmdStartGame(bot, message, user, chatID)
	case "/iniciar":
		cmdStartGame(bot, message, user, chatID)
	case "/sair":
		cmdLeaveGame(bot, message, user, chatID)
	case "/kill":
		cmdKillGame(bot, message, user, chatID)
	case "/fechar":
		cmdCloseGame(bot, message, user, chatID)
	case "/abrir":
		cmdOpenGame(bot, message, user, chatID)
	case "/pular":
		cmdSkipPlayer(bot, message, user, chatID)
	case "/kick":
		cmdKickPlayer(bot, message, user, chatID)
	case "/notificar":
		cmdNotifyMe(bot, message, user, chatID)
	case "/limpar":
		cmdCleanGames(bot, message, user, chatID)
	case "/ajuda":
		cmdHelp(bot, message, chatID)
	case "/ranking":
		cmdRanking(bot, message, user, chatID)
	case "/diario":
		cmdRankingDiario(bot, message, chatID)
	case "/semanal":
		cmdRankingSemanal(bot, message, chatID)
	case "/desafio":
		cmdDesafio(bot, message, user, chatID)
	case "/modos":
		cmdModes(bot, message, chatID)
	case "/modo":
		cmdModo(bot, message, chatID)
	case "/rankingx1":
		cmdRankingX1(bot, message, user, chatID)
	}
}

func handleCallbackQuery(bot *telego.Bot, query telego.CallbackQuery) {
	chatID := query.Message.GetChat().ID
	messageID := query.Message.GetMessageID()
	data := query.Data
	user := &UserData{
		ID:        query.From.ID,
		FirstName: query.From.FirstName,
		Username:  query.From.Username,
	}

	switch {
	case isChallengeCallback(data):
		handleChallengeCallback(bot, query, chatID, messageID, user)
	case isModeCallback(data):
		handleModeCallback(bot, query, chatID, messageID, data)
	case isInfoCallback(data):
		handleInfoCallback(bot, query, chatID, messageID, user, data)
	case isRankingCallback(data):
		handleRankingCallback(bot, query, chatID, messageID, data)
	}
}

func isChallengeCallback(data string) bool {
	switch data {
	case cbChallengeAccept, cbChallengeConfig,
		cbChallengeMD1, cbChallengeMD3, cbChallengeMD5,
		cbChallengeModeClassic, cbChallengeModeFast, cbChallengeModeWild, cbChallengeModeCaseiro, cbChallengeModeTest,
		cbMatchStart, cbMatchNext, cbMatchCancel:
		return true
	}
	return false
}

func isModeCallback(data string) bool {
	switch data {
	case cbSetModeClassic, cbSetModeFast, cbSetModeWild, cbSetModeText, cbSetModeCaseiro:
		return true
	}
	return false
}

func isInfoCallback(data string) bool {
	switch data {
	case cbInfoHelp, cbInfoModes, cbInfoRanking, cbInfoRankingX1, cbInfoBack:
		return true
	}
	return false
}

func isRankingCallback(data string) bool {
	switch data {
	case cbRankingDiario, cbRankingSemanal, cbRankingMensal:
		return true
	}
	return false
}

func handleModeCallback(bot *telego.Bot, query telego.CallbackQuery, chatID int64, messageID int, data string) {
	modeMap := map[string]string{
		cbSetModeClassic: "classic",
		cbSetModeFast:    "fast",
		cbSetModeWild:    "wild",
		cbSetModeText:    "text",
		cbSetModeCaseiro: "caseiro",
	}
	mode := modeMap[data]

	rankingStore.SetGroupDefaultMode(chatID, mode)
	_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	})

	_, err := bot.EditMessageText(botCtx, &telego.EditMessageTextParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageID,
		Text:      fmt.Sprintf("Modo definido para %s", modeDisplayName(mode)),
	})
	if err != nil {
		log.Printf("Erro ao editar mensagem de modo: %v", err)
	}
}

func handleInfoCallback(bot *telego.Bot, query telego.CallbackQuery, chatID int64, messageID int, user *UserData, data string) {
	_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	})
	reactMessage(bot, chatID, messageID, "👍")

	switch data {
	case cbInfoHelp:
		editWithBack(bot, chatID, messageID, helpText(), "")
	case cbInfoModes:
		editWithBack(bot, chatID, messageID, modesText(), telego.ModeHTML)
	case cbInfoRanking:
		if query.Message.GetChat().Type == telego.ChatTypePrivate {
			editUserRankingAcrossGroups(bot, chatID, messageID, user)
		} else {
			editRankingWithBack(bot, chatID, messageID, "mensal", "Mensal")
		}
	case cbInfoRankingX1:
		editRankingX1WithBack(bot, chatID, messageID, user)
	case cbInfoBack:
		editStartMessage(bot, chatID, messageID)
	}
}

func handleRankingCallback(bot *telego.Bot, query telego.CallbackQuery, chatID int64, messageID int, data string) {
	user := &UserData{
		ID:        query.From.ID,
		FirstName: query.From.FirstName,
		Username:  query.From.Username,
	}

	// No privado, mostra posição do usuário nos grupos
	if query.Message.GetChat().Type == telego.ChatTypePrivate {
		_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
		})
		rankings := rankingStore.GetUserRankingAcrossGroups(user.ID)
		if len(rankings) == 0 {
			_, _ = bot.EditMessageText(botCtx, &telego.EditMessageTextParams{
				ChatID:    telego.ChatID{ID: chatID},
				MessageID: messageID,
				Text:      "Você ainda não venceu nenhum jogo.",
			})
			return
		}

		text := "🏆 Sua posição nos grupos:\n\n"
		for _, r := range rankings {
			name := r.GroupName
			if name == "" {
				name = fmt.Sprintf("Grupo %d", r.ChatID)
			}
			text += fmt.Sprintf("#%d — %s (%d vitórias)\n", r.Rank, name, r.Wins)
		}

		_, _ = bot.EditMessageText(botCtx, &telego.EditMessageTextParams{
			ChatID:    telego.ChatID{ID: chatID},
			MessageID: messageID,
			Text:      text,
			ReplyMarkup: &telego.InlineKeyboardMarkup{
				InlineKeyboard: [][]telego.InlineKeyboardButton{
					{{Text: "◀️ Voltar", CallbackData: cbInfoBack}},
				},
			},
		})
		return
	}

	// No grupo, mostra ranking do grupo
	var periodo, periodoLabel string
	switch data {
	case cbRankingDiario:
		periodo = "diario"
		periodoLabel = "Diário"
	case cbRankingSemanal:
		periodo = "semanal"
		periodoLabel = "Semanal"
	case cbRankingMensal:
		periodo = "mensal"
		periodoLabel = "Mensal"
	default:
		return
	}

	ranking := rankingStore.GetRanking(chatID, periodo)
	text := formatRankingTexto(periodoLabel, ranking)

	var botoes [][]telego.InlineKeyboardButton
	switch periodo {
	case "diario":
		botoes = [][]telego.InlineKeyboardButton{
			{
				{Text: "📅 Mensal", CallbackData: cbRankingMensal},
				{Text: "📆 Semanal", CallbackData: cbRankingSemanal},
			},
		}
	case "semanal":
		botoes = [][]telego.InlineKeyboardButton{
			{
				{Text: "📅 Diário", CallbackData: cbRankingDiario},
				{Text: "📆 Mensal", CallbackData: cbRankingMensal},
			},
		}
	default:
		botoes = [][]telego.InlineKeyboardButton{
			{
				{Text: "📅 Diário", CallbackData: cbRankingDiario},
				{Text: "📆 Semanal", CallbackData: cbRankingSemanal},
			},
		}
	}

	_, err := bot.EditMessageText(botCtx, &telego.EditMessageTextParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageID,
		Text:      text,
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: botoes,
		},
	})
	if err != nil {
		log.Printf("Erro ao editar mensagem de ranking: %v", err)
	}

	_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	})
}

func editWithBack(bot *telego.Bot, chatID int64, messageID int, text string, parseMode string) {
	_, err := bot.EditMessageText(botCtx, &telego.EditMessageTextParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageID,
		Text:      text,
		ParseMode: parseMode,
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{{Text: "◀️ Voltar", CallbackData: cbInfoBack}},
			},
		},
	})
	if err != nil {
		log.Printf("Erro ao editar mensagem: %v", err)
	}
}

func editUserRankingAcrossGroups(bot *telego.Bot, chatID int64, messageID int, user *UserData) {
	rankings := rankingStore.GetUserRankingAcrossGroups(user.ID)
	if len(rankings) == 0 {
		_, _ = bot.EditMessageText(botCtx, &telego.EditMessageTextParams{
			ChatID:    telego.ChatID{ID: chatID},
			MessageID: messageID,
			Text:      "Você ainda não venceu nenhum jogo.",
			ReplyMarkup: &telego.InlineKeyboardMarkup{
				InlineKeyboard: [][]telego.InlineKeyboardButton{
					{{Text: "◀️ Voltar", CallbackData: cbInfoBack}},
				},
			},
		})
		return
	}

	text := "🏆 Sua posição nos grupos:\n\n"
	for _, r := range rankings {
		name := r.GroupName
		if name == "" {
			name = fmt.Sprintf("Grupo %d", r.ChatID)
		}
		text += fmt.Sprintf("#%d — %s (%d vitórias)\n", r.Rank, name, r.Wins)
	}

	_, err := bot.EditMessageText(botCtx, &telego.EditMessageTextParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageID,
		Text:      text,
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{{Text: "◀️ Voltar", CallbackData: cbInfoBack}},
			},
		},
	})
	if err != nil {
		log.Printf("Erro ao editar mensagem de ranking: %v", err)
	}
}

func editRankingWithBack(bot *telego.Bot, chatID int64, messageID int, periodo string, periodoLabel string) {
	ranking := rankingStore.GetRanking(chatID, periodo)
	text := formatRankingTexto(periodoLabel, ranking)

	_, err := bot.EditMessageText(botCtx, &telego.EditMessageTextParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageID,
		Text:      text,
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "📅 Diário", CallbackData: cbRankingDiario},
					{Text: "📆 Semanal", CallbackData: cbRankingSemanal},
				},
				{{Text: "◀️ Voltar", CallbackData: cbInfoBack}},
			},
		},
	})
	if err != nil {
		log.Printf("Erro ao editar mensagem de ranking: %v", err)
	}
}

func editRankingX1WithBack(bot *telego.Bot, chatID int64, messageID int, user *UserData) {
	list := rankingStore.GetChallengeRanking(chatID, user.ID)
	text := formatRankingX1Texto(user, list)

	_, err := bot.EditMessageText(botCtx, &telego.EditMessageTextParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageID,
		Text:      text,
		ParseMode: telego.ModeHTML,
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{{Text: "◀️ Voltar", CallbackData: cbInfoBack}},
			},
		},
	})
	if err != nil {
		log.Printf("Erro ao editar mensagem de ranking x1: %v", err)
	}
}

func editStartMessage(bot *telego.Bot, chatID int64, messageID int) {
	_, err := bot.EditMessageText(botCtx, &telego.EditMessageTextParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: messageID,
		Text:      startText(),
		ParseMode: telego.ModeHTML,
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "📋 Ajuda", CallbackData: cbInfoHelp},
					{Text: "🎮 Modos", CallbackData: cbInfoModes},
				},
				{
					{Text: "🏆 Ranking", CallbackData: cbInfoRanking},
					{Text: "⚔️ Ranking X1", CallbackData: cbInfoRankingX1},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Erro ao editar mensagem inicial: %v", err)
	}
}

func startText() string {
	return fmt.Sprintf(`🎮 <b>UnoBotGO</b>

Olá! Este bot é uma versão em desenvolvimento, inspirado no bot original @unopybot.

<b>Como usar:</b>
1. Adicione o bot a um grupo
2. No grupo, use /novo para criar um jogo
3. Jogadores entram com /entrar
4. Inicie com /iniciar
5. Digite <code>@%s</code> na mensagem para ver suas cartas

Use /ajuda para ver a lista completa de comandos.`, botUsername)
}

func handleMyChatMember(bot *telego.Bot, myChatMember telego.ChatMemberUpdated) {
	status := myChatMember.NewChatMember.MemberStatus()
	if status == telego.MemberStatusLeft || status == telego.MemberStatusBanned {
		chatID := myChatMember.Chat.ID
		log.Printf("Bot removido do grupo %d, limpando dados...", chatID)
		rankingStore.CleanGroupData(chatID)
	}
}

func isAdmin(userID int64) bool {
	return false
}
