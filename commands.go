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
	case "/modos":
		cmdModes(bot, message, chatID)
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
		cmdHelp(bot, msg, chatID)
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
		displayName(game.CurrentPlayer.User))
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
			sendNextMessage(bot, chatID, fmt.Sprintf("OK. Próximo jogador: %s", displayName(game.CurrentPlayer.User)))
		} else {
			sendMessage(bot, chatID, fmt.Sprintf("%s saiu do jogo.", displayName(user)))
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
		sendNextMessage(bot, chatID, fmt.Sprintf("Próximo jogador: %s", displayName(game.CurrentPlayer.User)))
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
		sendMessage(bot, chatID, fmt.Sprintf("%s foi expulso por %s", displayName(kicked), displayName(user)))
		sendMessage(bot, chatID, "Jogo encerrado!")
		return
	} else if err != nil {
		sendMessage(bot, chatID, fmt.Sprintf("Jogador %s não encontrado.", displayName(kicked)))
		return
	}

	sendMessage(bot, chatID, fmt.Sprintf("%s foi expulso por %s", displayName(kicked), displayName(user)))
	if game.Started && game.CurrentPlayer != nil {
		sendNextMessage(bot, chatID, fmt.Sprintf("Próximo jogador: %s", displayName(game.CurrentPlayer.User)))
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
/modos - Explicação dos modos de jogo`

	sendMessage(bot, chatID, helpText)
}

func cmdModes(bot *telego.Bot, msg telego.Message, chatID int64) {
	modesText := `Este bot tem quatro modos de jogo:

🎻 Classic - Baralho UNO convencional, sem auto-skip
🚀 Sanic - Baralho UNO convencional, auto-skip
🐉 Wild - Mais cartas especiais, menos números
✍️ Text - Baralho UNO convencional, sem stickers

O criador do jogo pode mudar o modo digitando @ na janela.`
	sendMessage(bot, chatID, modesText)
}

func isAdmin(userID int64) bool {
	return false
}
