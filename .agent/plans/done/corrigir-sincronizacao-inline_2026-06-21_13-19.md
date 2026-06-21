# Plano: corrigir-sincronizacao-inline

## Pedido do usuário
O usuário solicitou as seguintes correções/ajustes:
1. Remover a exposição do ID do chat no inline query (ficava feio no input do usuário ao clicar em "Suas cartas").
2. Investigar por que a carta jogada às vezes não saía do deck/mão (explicado pela dessincronização do `UserIDCurrent`, que fazia o bot tentar tirar a carta no jogo errado e falhar).
3. Questionou sobre usar banco de dados em memória (como Redis) ao invés de RAM direta do Go.

## Objetivo
Resolver os problemas sem expor o ID na query inline:
1. Manter a query inline limpa e vazia (`""`), retirando o ID do chat da caixa de texto do usuário.
2. Implementar a atualização automática e segura do ponteiro `UserIDCurrent` nas transições de turnos do jogo e em qualquer interação/mensagem do usuário no grupo.
3. Manter a arquitetura atual em memória RAM do Go para evitar a complexidade técnica e a sobrecarga de gerenciar dependências e serialização de ponteiros e listas circulares no Redis.

## Contexto atual
- Anteriormente tentamos passar o ChatID no botão `WithSwitchInlineQueryCurrentChat`, o que expunha o ID no input.
- O problema da carta não sumir da mão ocorre porque a ação inline era aplicada em um jogo obsoleto/errado do usuário devido ao `UserIDCurrent` dessincronizado.
- O gerenciador de jogos usa estruturas de anel circulares (`Player.Next` e `Player.Prev`) que são complexas de mapear para Redis de forma eficiente.

## Arquivos analisados
- [gamemanager.go](file:///home/gabriel/OpenCode/UnoBotGO/gamemanager.go)
- [commands.go](file:///home/gabriel/OpenCode/UnoBotGO/commands.go)
- [actions.go](file:///home/gabriel/OpenCode/UnoBotGO/actions.go)
- [inline.go](file:///home/gabriel/OpenCode/UnoBotGO/inline.go)

## Arquivos que poderão ser modificados
- [gamemanager.go](file:///home/gabriel/OpenCode/UnoBotGO/gamemanager.go)
- [commands.go](file:///home/gabriel/OpenCode/UnoBotGO/commands.go)
- [actions.go](file:///home/gabriel/OpenCode/UnoBotGO/actions.go)
- [inline.go](file:///home/gabriel/OpenCode/UnoBotGO/inline.go)

## Estratégia de implementação
1. **Reverter a passagem de ChatID na Query**:
   - Reverter em `actions.go` a string `fmt.Sprintf("%d", chatID)` de volta para `""` no botão de switch inline.
   - Remover em `inline.go` a decodificação do chatID nas funções `handleInlineQuery` e `handleChosenInlineResult`, usando apenas `gm.UserIDCurrent[userID]`.
2. **Atualização Automática de Turnos**:
   - Criar o método seguro `UpdateCurrentPlayer(game *Game)` no `GameManager`.
   - Adicionar chamadas para esse método no início do jogo, após jogadas no inline, após skips, kicks, e saídas para atualizar dinamicamente o `UserIDCurrent` de quem está jogando no momento.
3. **Atualização por Mensagens**:
   - Em `commands.go` / `handleMessage`, atualizar o `UserIDCurrent` do usuário remetente sob lock caso ele possua um player ativo correspondente ao chatID da mensagem.

## Passos detalhados

### Passo 1: Modificar `gamemanager.go`
1. Implementar o método thread-safe `UpdateCurrentPlayer(game *Game)` para atualizar o ponteiro `UserIDCurrent` do `CurrentPlayer` no map global:
   ```go
   func (gm *GameManager) UpdateCurrentPlayer(game *Game) {
   	gm.Lock()
   	defer gm.Unlock()
   	if game.CurrentPlayer != nil {
   		gm.UserIDCurrent[game.CurrentPlayer.User.ID] = game.CurrentPlayer
   	}
   }
   ```

### Passo 2: Modificar `commands.go`
1. Em `handleMessage`, logo após a criação do objeto `user`, adicionar um bloco sob lock do `gm` que atualiza o `gm.UserIDCurrent[user.ID]` para o player dele daquele chat se o usuário fizer parte daquele jogo:
   ```go
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
   ```
2. No `cmdStartGame` e `cmdLeaveGame`, invocar `gm.UpdateCurrentPlayer(game)` nas transições de turno bem-sucedidas.

### Passo 3: Modificar `actions.go`
1. Reverter o botão "Suas cartas" de volta para `WithSwitchInlineQueryCurrentChat("")`.

### Passo 4: Modificar `inline.go`
1. Reverter os blocos de parser de query em `handleInlineQuery` e `handleChosenInlineResult`, voltando a referenciar diretamente `gm.UserIDCurrent[userID]`.
2. No `handleChosenInlineResult`, após a execução de ações que mudam o turno, garantir a chamada a `gm.UpdateCurrentPlayer(game)`.

## Riscos
- **Deadlocks**: Garantir que as chamadas a `UpdateCurrentPlayer` não sejam feitas dentro de blocos que já seguram o lock de `gm`.

## Impactos esperados
- A inline query do usuário ficará limpa e bonita (exibindo apenas `@NomeDoBot `).
- O estado de jogo ativo (`UserIDCurrent`) será atualizado de forma proativa sempre que for a vez do jogador ou sempre que ele enviar uma mensagem/comando no grupo, garantindo sincronização sem expor dados no input.
- A carta jogada passará a sumir da mão normalmente, pois as jogadas inline serão aplicadas no jogo correto.

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

### Testes
Abrir o bot em dois grupos, jogar paralelamente e verificar se a abertura do inline query mostra as cartas correspondentes a cada chat sem misturar o estado e sem exibir IDs no input.

## Rollback
Caso ocorram erros inesperados, as alterações podem ser desfeitas via:
```bash
git checkout -- gamemanager.go commands.go actions.go inline.go
```

## Observações
O uso do Redis foi descartado devido à necessidade de serialização complexa e por introduzir dependências externas adicionais no ambiente do usuário.
