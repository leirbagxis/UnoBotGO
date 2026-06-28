package main

import (
	"fmt"
	"log"
	"time"

	"github.com/mymmrac/telego"
)

func cmdNewGame(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	if msg.Chat.Type != telego.ChatTypeGroup && msg.Chat.Type != telego.ChatTypeSupergroup {
		sendMessage(bot, chatID, "Use este comando em um grupo.")
		return
	}

	game := gm.NewGame(chatID, msg.Chat.Title)
	if game == nil {
		sendMessage(bot, chatID, "Já existe um desafio ativo neste grupo!")
		return
	}
	if game.Started {
		sendMessage(bot, chatID, "Já existe um jogo em andamento neste grupo!")
		return
	}
	game.Starter = user
	game.Owner = []int64{user.ID}

	sendMessage(bot, chatID, "Novo jogo criado! Entre com /entrar e inicie com /iniciar")
}

func cmdJoinGame(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
	if msg.Chat.Type != telego.ChatTypeGroup && msg.Chat.Type != telego.ChatTypeSupergroup {
		sendMessage(bot, chatID, "Use este comando em um grupo.")
		return
	}

	player, err := gm.JoinGame(user, chatID)
	switch err {
	case nil:
		msg := "Você entrou no jogo!"
		if faltam := GetMinPlayers() - len(player.Game.Players()); faltam > 0 {
			msg = fmt.Sprintf("Você entrou no jogo! Faltam %d jogador(es) para iniciar. Use /iniciar quando todos estiverem prontos!", faltam)
		}
		sendMessage(bot, chatID, msg)
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
		_, err := bot.SendMessage(botCtx, &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: chatID},
			Text:      startText(),
			ParseMode: telego.ModeHTML,
			ReplyParameters: &telego.ReplyParameters{
				MessageID: msg.MessageID,
			},
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
			log.Printf("Erro ao enviar mensagem /start: %v", err)
		}
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
		faltam := GetMinPlayers() - len(game.Players())
		sendMessage(bot, chatID, fmt.Sprintf("Faltam %d jogador(es) para iniciar. Use /iniciar quando todos estiverem prontos!", faltam))
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

	if game.MatchID != 0 {
		sendMessage(bot, chatID, "Você não pode sair de um desafio. Use o botão Cancelar na mensagem do desafio.")
		return
	}

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
	case ErrLastPlayerWin:
		game.Started = false
		remaining := game.Players()
		if len(remaining) == 1 {
			msgID := sendMessage(bot, chatID, fmt.Sprintf("%s venceu! Último jogador restante.", displayLink(remaining[0].User)))
			reactMessage(bot, chatID, msgID, "🎉")
			rankingStore.RecordWin(remaining[0].User, chatID, game.GroupName)
			showDailyRanking(bot, chatID)
		}
		gm.EndGameByGame(chatID, game)
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

	match := gm.GetMatch(chatID)
	if match != nil {
		gm.CancelMatch(chatID)
		sendMessage(bot, chatID, "Desafio cancelado!")
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
		sendMessage(bot, chatID, "O jogo ainda não foi iniciado. Use /entrar e /iniciar")
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

	if game.MatchID != 0 {
		sendMessage(bot, chatID, "Você não pode expulsar alguém de um desafio.")
		return
	}

	err := gm.LeaveGame(kicked, chatID)
	if err == ErrLastPlayerWin {
		kickedPlayer := gm.PlayerForUserInChat(kicked, chatID)
		if kickedPlayer != nil {
			kickedPlayer.Game.Started = false
		}
		remaining := game.Players()
		if len(remaining) == 1 {
			sendMessage(bot, chatID, fmt.Sprintf("%s foi expulso por %s", displayLink(kicked), displayLink(user)))
			msgID := sendMessage(bot, chatID, fmt.Sprintf("%s venceu! Último jogador restante.", displayLink(remaining[0].User)))
			reactMessage(bot, chatID, msgID, "🎉")
			rankingStore.RecordWin(remaining[0].User, chatID, game.GroupName)
			showDailyRanking(bot, chatID)
		}
		gm.EndGameByGame(chatID, game)
		return
	} else if err == ErrNotEnoughPlayers {
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

	match := gm.GetMatch(chatID)
	if match != nil {
		gm.CancelMatch(chatID)
		sendMessage(bot, chatID, "Desafio cancelado!")
		return
	}

	gm.Lock()
	games := gm.ChatIDGames[chatID]
	gm.Unlock()

	if len(games) == 0 {
		sendMessage(bot, chatID, "Nenhum jogo para limpar.")
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
	sendMessage(bot, chatID, helpText())
}

func cmdModes(bot *telego.Bot, msg telego.Message, chatID int64) {
	sendMessage(bot, chatID, modesText())
}

func helpText() string {
	return `Siga estes passos:

1. Adicione este bot a um grupo
2. No grupo, crie um novo jogo com /novo ou entre em um jogo com /entrar
3. Após pelo menos 2 jogadores, inicie com /iniciar
4. Digite @ na janela de mensagens e veja suas cartas

Comandos:
/novo - Criar novo jogo
/entrar - Entrar no jogo
/iniciar - Iniciar o jogo
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
/modo - Definir modo padrão do grupo
/desafio - Desafiar alguém para um MD1/MD3/MD5
/ranking - Ranking mensal
/diario - Ranking diário
/semanal - Ranking semanal`
}

func modesText() string {
	return `<b>🎮 Modos de jogo:</b>

🎻 <b>Classic</b> — UNO padrão. Sem auto-skip.
🚀 <b>Sanic</b> — UNO padrão. Jogador que demora é pulado automaticamente.
🐉 <b>Wild</b> — Mais cartas especiais (+4 e Choose), menos números.
✍️ <b>Text</b> — UNO padrão. Apenas texto, sem stickers.
🏠 <b>Caseiro</b> — +2 pode ser rebatido por +4; após +4 só +2 da cor escolhida.
🧪 <b>Test</b> — Cartas específicas (+4, +2, Reverse). Regras Caseiro. Apenas para teste.

O criador do jogo pode mudar o modo digitando @ no grupo antes de iniciar.`
}

func cmdModo(bot *telego.Bot, msg telego.Message, chatID int64) {
	if msg.Chat.Type != telego.ChatTypeGroup && msg.Chat.Type != telego.ChatTypeSupergroup {
		sendMessage(bot, chatID, "Use este comando em um grupo.")
		return
	}

	currentMode := rankingStore.GetGroupDefaultMode(chatID)
	text := fmt.Sprintf("Modo atual: %s\n\nEscolha o modo padrão para este grupo:", modeDisplayName(currentMode))

	_, err := bot.SendMessage(botCtx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      text,
		ParseMode: telego.ModeHTML,
		ReplyParameters: &telego.ReplyParameters{
			MessageID: msg.MessageID,
		},
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{Text: "🎻 Classic", CallbackData: cbSetModeClassic},
					{Text: "🚀 Sanic", CallbackData: cbSetModeFast},
				},
				{
					{Text: "🐉 Wild", CallbackData: cbSetModeWild},
					{Text: "✍️ Text", CallbackData: cbSetModeText},
				},
				{
					{Text: "🏠 Caseiro", CallbackData: cbSetModeCaseiro},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Erro ao enviar comando /modo: %v", err)
	}
}
