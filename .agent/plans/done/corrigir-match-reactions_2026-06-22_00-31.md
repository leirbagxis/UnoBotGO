# Plano: corrigir-match-reactions

## Pedido do usuário
1. Partida MD1 encerra mas ainda pede carta para o outro jogador
2. Adicionar reaction de espanto/olhinho quando jogador grita UNO
3. Adicionar reaction de confete quando jogador vence

## Arquivos que serão modificados
- actions.go
- inline.go
- gamemanager.go

## Implementação

### 1. inline.go — afterAction verifica se match encerrou
Logo após label `afterAction:`, checar `game.MatchID != 0 && gm.GetMatch(chatID) == nil`. Se verdadeiro, dar Unlock e return.

### 2. actions.go — sendMessage retorna MessageID + helper reactMessage
- `sendMessage` passa a retornar `int` (MessageID)
- Nova função `reactMessage(bot, chatID, msgID, emoji)` chama `bot.SetMessageReaction`
- UNO: `sendMessage` retorna ID → `reactMessage(bot, chatID, msgID, "😱")`
- Vitória: `sendMessage` retorna ID → `reactMessage(bot, chatID, msgID, "🎉")`

### 3. gamemanager.go — reaction na mensagem final do match
`sendMessage` já retorna ID agora → chamar `reactMessage` com `🎉`
