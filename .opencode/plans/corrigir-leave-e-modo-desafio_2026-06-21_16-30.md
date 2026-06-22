# Plano: corrigir-leave-e-modo-desafio

## Pedido do usuário
1. Quando um jogador sai, remover sempre. Se restar 1 jogador, ele vence.
2. Poder mudar o modo de jogo no /desafio (menu Configurar).

---

## Issue 1 — Leave: sempre remover, último jogador vence

### Problema atual
`leaveGame` (gamemanager.go:101-149) aborta com `ErrNotEnoughPlayers` se
`len(allPlayers) < 3` sem sequer remover o jogador. O jogo simplesmente
termina com "Jogo encerrado!" sem declarar vencedor.

### Solução
Remover o limite de 3. Sempre executar a remoção (Turn se necessário,
Leave, cleanup). Após, verificar quantos jogadores restam via
`game.Players()`:

- 1 jogador → return `ErrLastPlayerWin`
- 0 jogadores → return `ErrNotEnoughPlayers` (fallback, improvável)
- 2+ → return nil (jogo continua)

### Arquivos

**errors.go** — adicionar sentinela:
```go
ErrLastPlayerWin = errors.New("last player wins")
```

**gamemanager.go** — `leaveGame`:
- Remover bloco `if len(allPlayers) < 3 { return ErrNotEnoughPlayers }`
- Após `player.Leave()` + cleanup, adicionar:
```go
remaining := game.Players()
if len(remaining) <= 1 {
    if len(remaining) == 1 {
        return ErrLastPlayerWin
    }
    return ErrNotEnoughPlayers
}
return nil
```

**commands.go** — `cmdLeaveGame` (linha 185):
- Adicionar guarda no início: se `game.MatchID != 0`, recusar com
  "Você não pode sair de um desafio. Use o botão Cancelar."
- Caso `ErrLastPlayerWin`:
```go
game.Started = false
remaining := game.Players()
if len(remaining) == 1 {
    msgID := sendMessage(bot, chatID, fmt.Sprintf(
        "%s venceu! Último jogador restante.", displayLink(remaining[0].User)))
    reactMessage(bot, chatID, msgID, "🎉")
    rankingStore.RecordWin(remaining[0].User, chatID)
}
gm.EndGameByGame(chatID, game)
```

**commands.go** — `cmdKickPlayer` (linha 301):
- Mesma guarda de match
- Mesmo handler `ErrLastPlayerWin`

**actions.go** — `doSkip` (linha 76):
- Dentro do else (WaitingTime <= 0), adicionar handler `ErrLastPlayerWin`:
```go
if err == ErrLastPlayerWin {
    game.Started = false
    remaining := game.Players()
    if len(remaining) == 1 {
        msgID := sendMessage(bot, chatID, fmt.Sprintf(
            "%s venceu! Último jogador restante.", displayLink(remaining[0].User)))
        reactMessage(bot, chatID, msgID, "🎉")
    }
    gm.EndGameByGame(chatID, game)
} else if err == ErrNotEnoughPlayers {
    // 0 players, só encerrar
    game.Started = false
    sendMessage(bot, chatID, "Jogo encerrado!")
    gm.EndGameByGame(chatID, game)
}
```

**actions.go** — `doPlayCard` (linha 32):
- Handler `ErrLastPlayerWin` igual a `ErrNotEnoughPlayers`: só encerrar
  (já trata o vencedor ali em cima)

---

## Issue 2 — Modo de jogo no /desafio

### Problema atual
`startMatchGame` (gamemanager.go:254) sempre chama `game.Deck.FillClassic()`
ignorando qualquer modo. O menu Configurar só tem MD1/MD3/MD5.

### Solução
Adicionar campo `Mode` no `Match` e botões de modo no menu configurar.

### Arquivos

**match.go** — `Match` struct:
Adicionar:
```go
Mode string
```

**gamemanager.go** — `NewMatch` (linha 205):
No match literal, adicionar `Mode: GetDefaultGamemode()`.

**match.go** — `formatConfigMenu` (linha 72):
Texto:
```
Selecione o formato e o modo:

FORMATO
MD1 | MD3 | MD5

🎮 MODOS
🎻 Classic | 🚀 Fast
🐉 Wild | 🏠 Caseiro
🧪 Test
```

Teclado:
```go
InlineKeyboard: [][]telego.InlineKeyboardButton{
    {
        {Text: "MD1", CallbackData: "challenge_md1"},
        {Text: "MD3", CallbackData: "challenge_md3"},
        {Text: "MD5", CallbackData: "challenge_md5"},
    },
    {
        {Text: "🎻 Classic", CallbackData: "challenge_mode_classic"},
        {Text: "🚀 Fast",   CallbackData: "challenge_mode_fast"},
    },
    {
        {Text: "🐉 Wild",     CallbackData: "challenge_mode_wild"},
        {Text: "🏠 Caseiro", CallbackData: "challenge_mode_caseiro"},
    },
    {
        {Text: "🧪 Test", CallbackData: "challenge_mode_test"},
    },
},
```

**commands.go** — `handleChallengeCallback` (linha 556):
Adicionar 5 novos callbacks:
```go
case "challenge_mode_classic":
    match.Mode = "classic"
    formatConfigMenu(bot, chatID, match)
case "challenge_mode_fast":
    match.Mode = "fast"
    formatConfigMenu(bot, chatID, match)
case "challenge_mode_wild":
    match.Mode = "wild"
    formatConfigMenu(bot, chatID, match)
case "challenge_mode_caseiro":
    match.Mode = "caseiro"
    formatConfigMenu(bot, chatID, match)
case "challenge_mode_test":
    match.Mode = "test"
    formatConfigMenu(bot, chatID, match)
```

**match.go** — `formatChallengeMenu` (linha 51):
Adicionar modo no texto:
```go
text := fmt.Sprintf("🎮 %s quer um desafio!\n%s | %s",
    displayLink(match.Challenger),
    match.formatLabel(),
    modeDisplayName(match.Mode))
```

**match.go** — `sendMatchAccepted` (linha 94):
```go
text := fmt.Sprintf(
    "✅ %s aceitou o %s | %s contra %s!\nPlacar: 0×0\n\nPreparar partida 1?",
    displayLink(match.Challenged),
    match.formatLabel(),
    modeDisplayName(match.Mode),
    displayLink(match.Challenger))
```

**match.go** — nova função auxiliar:
```go
func modeDisplayName(mode string) string {
    switch mode {
    case "classic":  return "Classic 🎻"
    case "fast":     return "Sanic 🚀"
    case "wild":     return "Wild 🐉"
    case "text":     return "Text ✍️"
    case "caseiro":  return "Caseiro 🏠"
    case "test":     return "Test 🧪"
    default:         return mode
    }
}
```

**gamemanager.go** — `startMatchGame` (linha 242):
Trocar `game.Deck.FillClassic()` por:
```go
game.Mode = match.Mode
if game.Mode == "wild" {
    game.Deck.FillWild()
} else {
    game.Deck.FillClassic()
}
```

---

## Riscos
- Nenhum. São mudanças localizadas e testáveis.

## Como testar

### Leave
1. Criar jogo com 2 jogadores
2. Um sai → o outro deve vencer com "venceu! Último jogador restante."
3. Criar jogo com 3 jogadores
4. Um sai → jogo continua com 2 jogadores

### Modo desafio
1. `/desafio` → ⚙️ Configurar → selecionar MD3 + Wild
2. Volta ao menu com "🎮 Fulano quer um desafio! MD3 | Wild 🐉"
3. Aceitar → "Preparar partida 1?" → Começar
4. Deck deve ter cartas do Wild (mais especiais, menos números)
