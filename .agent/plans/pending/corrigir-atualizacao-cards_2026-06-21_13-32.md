# Plano: corrigir-atualizacao-cards

## Pedido do usuário
O usuário relatou irritado que as cartas não atualizam na tela e que o jogador continua com as mesmas cartas para sempre ("a carta jogada simplesmente nao some do deck do usuario").

## Objetivo
Invalidar o cache local das inline queries no aplicativo do Telegram do usuário de forma invisível/elegante, sem expor IDs de chats ou dados feios.
1. Adicionar o campo `PlayCounter int` na struct `Game` para atuar como um contador de rodadas/jogadas.
2. Incrementar o `PlayCounter` a cada ação que altere a mão de cartas do jogo (descarte, compra, pulo, blefe, escolha de cor).
3. Atualizar a função `sendNextMessage` para passar o `PlayCounter` na query do botão "Suas cartas" (gerando `@NomeDoBot <playCounter>`). Como a query string muda a cada turno (ex: `@NomeDoBot 1`, `@NomeDoBot 2`), o Telegram é forçado a invalidar o cache local e solicitar as cartas atualizadas do bot na hora, fazendo a carta jogada sumir da tela instantaneamente.

## Contexto atual
- O Telegram cacheia agressivamente as inline queries que possuem a mesma query string (como a string vazia `@NomeDoBot `).
- Mesmo que o bot remova a carta da memória, ao clicar no botão, o aplicativo do Telegram no celular do usuário não envia uma nova requisição ao bot e re-exibe as cartas antigas cacheadas localmente, impedindo a atualização visual.

## Arquivos analisados
- [game.go](file:///home/gabriel/OpenCode/UnoBotGO/game.go)
- [actions.go](file:///home/gabriel/OpenCode/UnoBotGO/actions.go)
- [commands.go](file:///home/gabriel/OpenCode/UnoBotGO/commands.go)

## Arquivos que poderão ser modificados
- [game.go](file:///home/gabriel/OpenCode/UnoBotGO/game.go)
- [actions.go](file:///home/gabriel/OpenCode/UnoBotGO/actions.go)
- [commands.go](file:///home/gabriel/OpenCode/UnoBotGO/commands.go)

## Estratégia de implementação
1. **Adicionar `PlayCounter` no Game**:
   - Adicionar o campo `PlayCounter int` no arquivo `game.go` -> struct `Game`.
2. **Atualizar Incrementos de Ações**:
   - Em `game.go` -> `PlayCard` e `ChooseColor`, incrementar `PlayCounter++`.
   - Em `actions.go` -> `doDraw`, `doSkip`, `doCallBluff`, incrementar `PlayCounter++` no respectivo jogo.
3. **Mudar Query String em `sendNextMessage`**:
   - Em `actions.go` -> `sendNextMessage`, buscar o jogo ativo do chatID via `gm.ChatIDGames` e injetar o `PlayCounter` atual no switch de inline query do botão (ex: `@NomeDoBot <playCounter>`).

## Passos detalhados

### Passo 1: Modificar `game.go`
1. Na struct `Game`, adicionar o campo:
   ```go
   PlayCounter int
   ```
2. Na função `PlayCard(card *Card)`, adicionar no início:
   ```go
   g.PlayCounter++
   ```
3. Na função `ChooseColor(color string)`, adicionar no início:
   ```go
   g.PlayCounter++
   ```

### Passo 2: Modificar `actions.go`
1. Em `doDraw(bot, player)`, adicionar:
   ```go
   player.Game.PlayCounter++
   ```
2. Em `doSkip(bot, player)`, adicionar no início/fim:
   ```go
   game.PlayCounter++
   ```
3. Em `doCallBluff(bot, player)`, adicionar:
   ```go
   game.PlayCounter++
   ```
4. Em `sendNextMessage`, alterar o botão para carregar o `PlayCounter`:
   ```go
   gm.Lock()
   games := gm.ChatIDGames[chatID]
   var playCounter int
   if len(games) > 0 {
   	playCounter = games[len(games)-1].PlayCounter
   }
   gm.Unlock()

   // e no botão:
   WithSwitchInlineQueryCurrentChat(fmt.Sprintf("%d", playCounter))
   ```

### Passo 3: Modificar `commands.go`
1. Em `cmdStartGame`, definir o início do contador:
   ```go
   game.PlayCounter = 1
   ```

## Riscos
Nenhum risco de segurança ou concorrência, pois as operações sobre `PlayCounter` seguem os locks de Mutex existentes.

## Impactos esperados
- O Telegram invalidará o cache local e carregará as cartas corretas a cada jogada.
- A carta jogada sumirá na hora da tela do usuário.
- O input do usuário exibirá apenas um número pequeno correspondente à rodada atual do jogo (ex: `@UnoGoBot 14`), o que é limpo e não expõe IDs confidenciais.

## Compatibilidade
- Linux
- macOS
- Windows
- Docker

## Como testar

### Build
```bash
go build -o UnoGoBot
```

### Execução e Teste
Iniciar uma partida e verificar que as cartas somem e atualizam na hora ao serem jogadas, e o campo de mensagem exibe apenas o número de rodadas.

## Rollback
```bash
git checkout -- game.go actions.go commands.go
```
