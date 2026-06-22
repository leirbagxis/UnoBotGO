# Plano: corrigir-fluxo-fim-partida-md

## Pedido do usuário
"Do jogando uma partida md3 e quando acabou ele pediu para o proximo jogador enviar uma carta e nao era pra isso acontecer. tbm quero que quando acabe ele mostra a mensagem com o placar e para ir para a proxima rodada"

Também relatou: "Tempo de espera para pular este jogador foi reduzido para 100 segundos. Próximo jogador: NA; tbm apareceu essa mensagem e ele nao pulou o jogador de fato, acabou bugando"

## Objetivo
Corrigir o fluxo de fim de partida MD para que:
1. Quando uma rodada termina, NÃO apareça "Próximo jogador" com botão "Suas cartas"
2. Apareça corretamente o placar com "Preparar partida N?" e botões "Próxima partida" / "Cancelar"
3. O timer de skip pare de rodar para jogos já finalizados

## Contexto atual

### Bug 1 — MD3 pede próximo jogador após fim da rodada
Quando um jogador joga a última carta:
1. `doPlayCard` → `player.Play(card)` → `game.PlayCard(card)` → **`game.Turn()`** avança `CurrentPlayer` para o perdedor ANTES da checagem de vitória
2. `doPlayCard` detecta `len(player.Cards) == 0`, chama `endMatchGame`
3. `endMatchGame` remove jogadores dos maps mas **NÃO seta `game.Started = false`**
4. Controle volta para `handleChosenInlineResult.afterAction`
5. Guard `if game.MatchID != 0 && gm.GetMatch(game.ChatID) == nil` só pega match DELETADA — em `MatchBetweenGames` a match ainda existe → guard FALHA
6. `game.Started` = true → `afterAction` re-adiciona perdedor e envia "Próximo jogador: [perdedor]" com "Suas cartas"
7. Perdedor pode jogar cartas numa rodada encerrada

### Bug 2 — Timer de skip não para
- `endMatchGame` nunca seta `game.Started = false`
- Goroutine do `startPlayerCountdown` continua rodando para o jogo antigo
- Timer dispara `doSkip` em jogo que já deveria estar parado
- Causa "NA" quando `CurrentPlayer.Next.User` está inconsistente

## Arquivos analisados
- gamemanager.go, inline.go, actions.go, game.go, player.go, commands.go

## Arquivos que serão modificados
- **gamemanager.go** (+2 linhas)
- **inline.go** (+5 linhas)

## Estratégia
### Fix 1: `endMatchGame` — parar o jogo
Adicionar `game.Started = false` após remover o jogo dos maps. Isso para a goroutine (que checa `game.Started`) e é consistente com `endGame` (linha 188).

### Fix 2: `afterAction` — verificar match state
Modificar guard para também checar `match.State == MatchPlaying`. Se match existe mas não está Playing, não enviar "Próximo jogador".

## Passos detalhados

### Passo 1: gamemanager.go — endMatchGame
Adicionar `game.Started = false` após a limpeza, antes do unlock.

### Passo 2: inline.go — afterAction
Modificar guard de linha 190 para:
```go
if game.MatchID != 0 {
    match := gm.GetMatch(game.ChatID)
    if match == nil || match.State != MatchPlaying {
        game.Unlock()
        return
    }
}
```

### Passo 3: Build
```bash
go build ./... && go vet ./...
```

## Riscos
Nenhum. `game.Started = false` já é feito em `endGame`. Guard é defensivo.

## Como testar
1. Iniciar MD3 com dois jogadores
2. Jogar até alguém zerar cartas
3. Verificar:
   - "X venceu!" com reação 🎉
   - Placar "Player1 1×0 Player2\n\nPreparar partida 2?" com botões
   - NÃO aparece "Próximo jogador" com "Suas cartas"
   - "Próxima partida" inicia nova rodada

## Rollback
```bash
git checkout -- gamemanager.go inline.go
```
