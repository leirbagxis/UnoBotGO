package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mymmrac/telego"
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
		cmdRanking(bot, message, chatID)
	case "/diario":
		cmdRankingDiario(bot, message, chatID)
	case "/semanal":
		cmdRankingSemanal(bot, message, chatID)
	case "/desafio":
		cmdDesafio(bot, message, user, chatID)
	case "/modos":
		cmdModes(bot, message, chatID)
	case "/rankingx1":
		cmdRankingX1(bot, message, user, chatID)
	}
}

func cmdNewGame(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	if msg.Chat.Type != telego.ChatTypeGroup && msg.Chat.Type != telego.ChatTypeSupergroup {
		sendMessage(bot, chatID, "Use este comando em um grupo.")
		return
	}

	game := gm.NewGame(chatID)
	if game.Starter != nil {
		sendMessage(bot, chatID, "Já existe um jogo neste grupo!")
		return
	}
	game.Starter = user
	game.Owner = append(game.Owner, user.ID)

	sendMessage(bot, chatID, "Novo jogo criado! Entre com /entrar e inicie com /start")
}

func cmdJoinGame(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	if msg.Chat.Type != telego.ChatTypeGroup && msg.Chat.Type != telego.ChatTypeSupergroup {
		sendMessage(bot, chatID, "Use este comando em um grupo.")
		return
	}

	err := gm.JoinGame(user, chatID)
	switch err {
	case nil:
		sendMessage(bot, chatID, "Você entrou no jogo!")
	case ErrLobbyClosed:
		sendMessage(bot, chatID, "O lobby está fechado.")
	case ErrNoGameInChat:
		sendMessage(bot, chatID, "Nenhum jogo ativo. Crie um com /novo")
	case ErrAlreadyJoined:
		sendMessage(bot, chatID, "Você já está no jogo.")
	case ErrDeckEmpty:
		sendMessage(bot, chatID, "Não há cartas suficientes no baralho.")
	default:
		log.Printf("Error joining game: %v", err)
	}
}

func cmdStartGame(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	if msg.Chat.Type == telego.ChatTypePrivate {
		text := fmt.Sprintf(`<b>🎮 UnoBotGO</b>

Bot para jogar UNO em grupos do Telegram.

<b>Como usar:</b>
1. Adicione o bot a um grupo
2. No grupo, use /novo para criar um jogo
3. Jogadores entram com /entrar
4. Inicie com /start
5. Digite <code>@%s</code> na mensagem para ver suas cartas

<b>Comandos principais:</b>
/novo - Criar novo jogo
/entrar - Entrar no jogo
/start - Iniciar o jogo
/ajuda - Lista completa
/ranking - Rankings`, botUsername)

		sendMessage(bot, chatID, text)
		return
	}

	games := gm.ChatIDGames[chatID]
	if len(games) == 0 {
		sendMessage(bot, chatID, "Nenhum jogo ativo. Crie um com /novo")
		return
	}
	game := games[len(games)-1]

	if game.Started {
		sendMessage(bot, chatID, "O jogo já foi iniciado.")
		return
	}

	if len(game.Players()) < GetMinPlayers() {
		sendMessage(bot, chatID, fmt.Sprintf("Pelo menos %d jogadores devem entrar no jogo.", GetMinPlayers()))
		return
	}

	game.Start()
	gm.UpdateCurrentPlayer(game)
	for _, player := range game.Players() {
		player.DrawFirstHand()
	}

	if game.LastCard != nil {
		sendSticker(bot, chatID, Stickers[game.LastCard.String()])
	}

	firstMsg := fmt.Sprintf(
		"Primeiro jogador: %s\nUse /fechar para impedir que mais pessoas entrem.",
		displayLink(game.CurrentPlayer.User))
	sendNextMessage(bot, chatID, firstMsg)
	startPlayerCountdown(bot, game)
}

func cmdLeaveGame(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	player := gm.PlayerForUserInChat(user, chatID)
	if player == nil {
		sendMessage(bot, chatID, "Você não está em nenhum jogo neste grupo.")
		return
	}
	game := player.Game

	err := gm.LeaveGame(user, chatID)
	switch err {
	case nil:
		if game.Started {
			gm.UpdateCurrentPlayer(game)
			sendNextMessage(bot, chatID, fmt.Sprintf("OK. Próximo jogador: %s", displayLink(game.CurrentPlayer.User)))
		} else {
			sendMessage(bot, chatID, fmt.Sprintf("%s saiu do jogo.", displayLink(user)))
		}
	case ErrNoGameInChat:
		sendMessage(bot, chatID, "Você não está em nenhum jogo neste grupo.")
	case ErrNotEnoughPlayers:
		game.Started = false
		gm.EndGameByGame(chatID, game)
		sendMessage(bot, chatID, "Jogo encerrado!")
	default:
		log.Printf("Error leaving game: %v", err)
	}
}

func cmdKillGame(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	if msg.Chat.Type == telego.ChatTypePrivate {
		cmdHelp(bot, msg, chatID)
		return
	}

	games := gm.ChatIDGames[chatID]
	if len(games) == 0 {
		sendMessage(bot, chatID, "Nenhum jogo ativo neste chat.")
		return
	}
	game := games[len(games)-1]

	if user.ID != game.Starter.ID && !isAdmin(user.ID) {
		sendMessage(bot, chatID, fmt.Sprintf("Apenas o criador do jogo (%s) pode fazer isso.", game.Starter.FirstName))
		return
	}

	err := gm.EndGame(chatID, user)
	if err != nil {
		sendMessage(bot, chatID, "O jogo ainda não foi iniciado. Use /entrar e /start")
	} else {
		game.Started = false
		sendMessage(bot, chatID, "Jogo encerrado!")
	}
}

func cmdCloseGame(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	games := gm.ChatIDGames[chatID]
	if len(games) == 0 {
		sendMessage(bot, chatID, "Nenhum jogo ativo neste chat.")
		return
	}
	game := games[len(games)-1]

	if user.ID != game.Starter.ID && !isAdmin(user.ID) {
		sendMessage(bot, chatID, fmt.Sprintf("Apenas o criador do jogo (%s) pode fazer isso.", game.Starter.FirstName))
		return
	}

	game.Open = false
	sendMessage(bot, chatID, "Lobby fechado. Ninguém mais pode entrar.")
}

func cmdOpenGame(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	games := gm.ChatIDGames[chatID]
	if len(games) == 0 {
		sendMessage(bot, chatID, "Nenhum jogo ativo neste chat.")
		return
	}
	game := games[len(games)-1]

	if user.ID != game.Starter.ID && !isAdmin(user.ID) {
		sendMessage(bot, chatID, fmt.Sprintf("Apenas o criador do jogo (%s) pode fazer isso.", game.Starter.FirstName))
		return
	}

	game.Open = true
	sendMessage(bot, chatID, "Lobby aberto. Novos jogadores podem /entrar")
}

func cmdSkipPlayer(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	player := gm.PlayerForUserInChat(user, chatID)
	if player == nil {
		sendMessage(bot, chatID, "Você não está em nenhum jogo neste grupo.")
		return
	}

	game := player.Game
	skippedPlayer := game.CurrentPlayer
	delta := timeSince(skippedPlayer.TurnStarted)

	if delta < skippedPlayer.WaitingTime && player != skippedPlayer {
		n := skippedPlayer.WaitingTime - delta
		sendMessage(bot, chatID, fmt.Sprintf("Aguarde %d segundos.", n))
		return
	}

	doSkip(bot, player)
	if game.Started && game.CurrentPlayer != nil {
		sendNextMessage(bot, chatID, fmt.Sprintf("Próximo jogador: %s", displayLink(game.CurrentPlayer.User)))
	}
}

func timeSince(t time.Time) int {
	return int(time.Since(t).Seconds())
}

func cmdKickPlayer(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	if msg.Chat.Type == telego.ChatTypePrivate {
		cmdHelp(bot, msg, chatID)
		return
	}

	games := gm.ChatIDGames[chatID]
	if len(games) == 0 {
		sendMessage(bot, chatID, "Nenhum jogo ativo neste chat.")
		return
	}
	game := games[len(games)-1]

	if !game.Started {
		sendMessage(bot, chatID, "O jogo ainda não começou.")
		return
	}

	if user.ID != game.Starter.ID && !isAdmin(user.ID) {
		sendMessage(bot, chatID, fmt.Sprintf("Apenas o criador do jogo (%s) pode fazer isso.", game.Starter.FirstName))
		return
	}

	if msg.ReplyToMessage == nil {
		sendMessage(bot, chatID, "Responda a mensagem de quem você quer expulsar.")
		return
	}

	kicked := &UserData{
		ID:        msg.ReplyToMessage.From.ID,
		FirstName: msg.ReplyToMessage.From.FirstName,
		Username:  msg.ReplyToMessage.From.Username,
	}

		err := gm.LeaveGame(kicked, chatID)
	if err == ErrNotEnoughPlayers {
		kickedPlayer := gm.PlayerForUserInChat(kicked, chatID)
		if kickedPlayer != nil {
			kickedPlayer.Game.Started = false
			gm.EndGameByGame(chatID, kickedPlayer.Game)
		}
		sendMessage(bot, chatID, fmt.Sprintf("%s foi expulso por %s", displayLink(kicked), displayLink(user)))
		sendMessage(bot, chatID, "Jogo encerrado!")
		return
	} else if err != nil {
		sendMessage(bot, chatID, fmt.Sprintf("Jogador %s não encontrado.", displayLink(kicked)))
		return
	}

	sendMessage(bot, chatID, fmt.Sprintf("%s foi expulso por %s", displayLink(kicked), displayLink(user)))
	if game.Started && game.CurrentPlayer != nil {
		sendNextMessage(bot, chatID, fmt.Sprintf("Próximo jogador: %s", displayLink(game.CurrentPlayer.User)))
	}
}

func cmdCleanGames(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	if msg.Chat.Type == telego.ChatTypePrivate {
		sendMessage(bot, chatID, "Use este comando em um grupo.")
		return
	}

	gm.Lock()
	games := gm.ChatIDGames[chatID]
	gm.Unlock()

	if len(games) == 0 {
		sendMessage(bot, chatID, "Nenhum jogo para limpar.")
		return
	}

	last := games[len(games)-1]
	if last.Starter != nil && user.ID != last.Starter.ID && !isAdmin(user.ID) {
		sendMessage(bot, chatID, "Apenas o criador do jogo pode fazer isso.")
		return
	}

	removed, err := gm.CleanGames(chatID)
	if err != nil {
		log.Printf("Error cleaning games: %v", err)
		return
	}

	if removed == 0 {
		sendMessage(bot, chatID, "Nenhum jogo não iniciado para limpar.")
		return
	}

	sendMessage(bot, chatID, fmt.Sprintf("Limpou %d jogo(s) não iniciado(s).", removed))
}

func cmdNotifyMe(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	if msg.Chat.Type == telego.ChatTypePrivate {
		sendMessage(bot, chatID, "Use este comando em um grupo para ser notificado quando um novo jogo começar.")
		return
	}

	if gm.RemindDict[chatID] == nil {
		gm.RemindDict[chatID] = make(map[int64]bool)
	}
	gm.RemindDict[chatID][user.ID] = true
	sendMessage(bot, chatID, "Você será notificado quando um novo jogo começar.")
}

func cmdHelp(bot *telego.Bot, msg telego.Message, chatID int64) {
	helpText := `Siga estes passos:

1. Adicione este bot a um grupo
2. No grupo, crie um novo jogo com /novo ou entre em um jogo com /entrar
3. Após pelo menos 2 jogadores, inicie com /start
4. Digite @ na janela de mensagens e veja suas cartas

Comandos:
/novo - Criar novo jogo
/entrar - Entrar no jogo
/start - Iniciar o jogo
/sair - Sair do jogo
/fechar - Fechar lobby
/abrir - Abrir lobby
/kill - Encerrar jogo
/pular - Pular jogador atual
/kick - Expulsar jogador
/limpar - Limpar jogos não iniciados
/notificar - Notificar quando novo jogo começar
/ajuda - Esta ajuda
/modos - Modos de jogo
/desafio - Desafiar alguém para um MD1/MD3/MD5
/ranking - Ranking mensal
/diario - Ranking diário
/semanal - Ranking semanal`
	sendMessage(bot, chatID, helpText)
}

func cmdModes(bot *telego.Bot, msg telego.Message, chatID int64) {
	text := `<b>🎮 Modos de jogo:</b>

🎻 <b>Classic</b> — UNO padrão. Sem auto-skip.
🚀 <b>Sanic</b> — UNO padrão. Jogador que demora é pulado automaticamente.
🐉 <b>Wild</b> — Mais cartas especiais (+4 e Choose), menos números.
✍️ <b>Text</b> — UNO padrão. Apenas texto, sem stickers.
🏠 <b>Caseiro</b> — +2 pode ser rebatido por +4; após +4 só +2 da cor escolhida.

O criador do jogo pode mudar o modo digitando @ no grupo antes de iniciar.`
	sendMessage(bot, chatID, text)
}

func cmdRanking(bot *telego.Bot, msg telego.Message, chatID int64) {
	ranking := rankingStore.GetRanking(chatID, "mensal")
	text := formatRankingTexto("Mensal", ranking)

	_, err := bot.SendMessage(botCtx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   text,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: msg.MessageID,
		},
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "📅 Diário", CallbackData: "ranking_diario"},
					{Text: "📆 Semanal", CallbackData: "ranking_semanal"},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Erro ao enviar ranking: %v", err)
	}
}

func cmdRankingDiario(bot *telego.Bot, msg telego.Message, chatID int64) {
	ranking := rankingStore.GetRanking(chatID, "diario")
	text := formatRankingTexto("Diário", ranking)

	_, err := bot.SendMessage(botCtx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   text,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: msg.MessageID,
		},
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "📅 Mensal", CallbackData: "ranking_mensal"},
					{Text: "📆 Semanal", CallbackData: "ranking_semanal"},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Erro ao enviar ranking: %v", err)
	}
}

func cmdRankingSemanal(bot *telego.Bot, msg telego.Message, chatID int64) {
	ranking := rankingStore.GetRanking(chatID, "semanal")
	text := formatRankingTexto("Semanal", ranking)

	_, err := bot.SendMessage(botCtx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text:   text,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: msg.MessageID,
		},
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "📅 Diário", CallbackData: "ranking_diario"},
					{Text: "📆 Mensal", CallbackData: "ranking_mensal"},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Erro ao enviar ranking: %v", err)
	}
}

func cmdDesafio(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	if msg.Chat.Type != telego.ChatTypeGroup && msg.Chat.Type != telego.ChatTypeSupergroup {
		sendMessage(bot, chatID, "Use este comando em um grupo.")
		return
	}

	match := gm.NewMatch(chatID, user)
	if match == nil {
		sendMessage(bot, chatID, "Já existe um jogo ou desafio ativo neste grupo.")
		return
	}

	text := fmt.Sprintf("🎮 %s quer um desafio!\nFormato: %s",
		displayLink(user), match.formatLabel())

	sent, err := bot.SendMessage(botCtx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      text,
		ParseMode: telego.ModeHTML,
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "✅ Aceitar", CallbackData: "challenge_accept"},
					{Text: "⚙️ Configurar", CallbackData: "challenge_config"},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Erro ao enviar desafio: %v", err)
		gm.CancelMatch(chatID)
		return
	}

	match.MessageID = sent.MessageID
}

func handleChallengeCallback(bot *telego.Bot, query telego.CallbackQuery, chatID int64, messageID int, user *UserData) {
	match := gm.GetMatch(chatID)
	if match == nil {
		_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "Desafio não encontrado ou já encerrado.",
		})
		return
	}

	switch query.Data {
	case "challenge_config":
		formatConfigMenu(bot, chatID, match)

	case "challenge_md1":
		match.BestOf = 1
		match.TargetWins = 1
		formatChallengeMenu(bot, chatID, match)

	case "challenge_md3":
		match.BestOf = 3
		match.TargetWins = 2
		formatChallengeMenu(bot, chatID, match)

	case "challenge_md5":
		match.BestOf = 5
		match.TargetWins = 3
		formatChallengeMenu(bot, chatID, match)

	case "challenge_accept":
		if user.ID == match.Challenger.ID {
			_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
				CallbackQueryID: query.ID,
				Text:            "Você não pode aceitar seu próprio desafio!",
			})
			return
		}
		match.Challenged = user
		match.State = MatchBetweenGames
		_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
		})
		sendMatchAccepted(bot, match)

	case "match_start":
		if match.State != MatchBetweenGames || match.Challenged == nil {
			return
		}
		if user.ID != match.Challenger.ID && user.ID != match.Challenged.ID {
			_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
				CallbackQueryID: query.ID,
				Text:            "Apenas os jogadores do desafio podem iniciar.",
			})
			return
		}
		_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
		})
		gm.startMatchGame(bot, match)

	case "match_next":
		if match.State != MatchBetweenGames {
			return
		}
		if user.ID != match.Challenger.ID && user.ID != match.Challenged.ID {
			return
		}
		_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
		})
		gm.startMatchGame(bot, match)

	case "match_cancel":
		gm.CancelMatch(chatID)
		_ = bot.AnswerCallbackQuery(botCtx, &telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
		})
		_, _ = bot.EditMessageText(botCtx, &telego.EditMessageTextParams{
			ChatID:    telego.ChatID{ID: chatID},
			MessageID: messageID,
			Text:      "Desafio cancelado.",
		})
	}
}

func cmdRankingX1(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	list := rankingStore.GetChallengeRanking(chatID, user.ID)

	if len(list) == 0 {
		sendMessage(bot, chatID, "Nenhum confronto registrado para você neste grupo.")
		return
	}

	text := fmt.Sprintf("Confrontos de %s:\n\n", displayLink(user))
	for _, h := range list {
		opUser := &UserData{
			ID:        h.OpponentID,
			FirstName: h.OpponentName,
			Username:  h.OpponentUsername,
		}
		text += fmt.Sprintf("vs %s — %dV %dD\n", displayLink(opUser), h.MyWins, h.MyLosses)
	}

	_, err := bot.SendMessage(botCtx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      text,
		ParseMode: telego.ModeHTML,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: msg.MessageID,
		},
	})
	if err != nil {
		log.Printf("Erro ao enviar ranking x1: %v", err)
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

	switch data {
	case "challenge_accept", "challenge_config", "challenge_md1", "challenge_md3", "challenge_md5",
		"match_start", "match_next", "match_cancel":
		handleChallengeCallback(bot, query, chatID, messageID, user)
		return
	}

	var periodo, periodoLabel string
	switch query.Data {
	case "ranking_diario":
		periodo = "diario"
		periodoLabel = "Diário"
	case "ranking_semanal":
		periodo = "semanal"
		periodoLabel = "Semanal"
	case "ranking_mensal":
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
				{Text: "📅 Mensal", CallbackData: "ranking_mensal"},
				{Text: "📆 Semanal", CallbackData: "ranking_semanal"},
			},
		}
	case "semanal":
		botoes = [][]telego.InlineKeyboardButton{
			{
				{Text: "📅 Diário", CallbackData: "ranking_diario"},
				{Text: "📆 Mensal", CallbackData: "ranking_mensal"},
			},
		}
	default:
		botoes = [][]telego.InlineKeyboardButton{
			{
				{Text: "📅 Diário", CallbackData: "ranking_diario"},
				{Text: "📆 Semanal", CallbackData: "ranking_semanal"},
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

func formatRankingTexto(periodo string, ranking []WinCount) string {
	if len(ranking) == 0 {
		return fmt.Sprintf("🏆 Ranking %s\n\nNenhuma vitória registrada neste período.", periodo)
	}

	text := fmt.Sprintf("🏆 Ranking %s\n\n", periodo)
	for i, r := range ranking {
		nome := r.FirstName
		if r.Username != "" {
			nome = "@" + r.Username
		}
		text += fmt.Sprintf("%d. %s — %d vitórias\n", i+1, nome, r.Wins)
	}
	return text
}

func isAdmin(userID int64) bool {
	return false
}
