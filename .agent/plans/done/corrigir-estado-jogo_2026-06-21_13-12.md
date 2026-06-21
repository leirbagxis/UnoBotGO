# Plano: corrigir-estado-jogo

## Pedido do usuário
O usuário relatou problemas graves com o estado do jogo e o inline mode do Telegram:
1. Cartas continuam atreladas ao usuário mesmo quando não há jogo ativo.
2. Às vezes o usuário está em um jogo mas a inline query diz que ele não está (ou não mostra as cartas).
3. O bot está configurado com 100% de feedback do inline query no BotFather.

## Objetivo
Corrigir os bugs de estado órfão e a sincronização do inline query com o chat correto.
1. Impedir que o comando `/limpar` (cmdCleanGames) deixe jogadores "fantasmas" na memória global, limpando corretamente as referências deles nos maps do `GameManager`.
2. Enviar o ID do chat atual no parâmetro `switch_inline_query_current_chat` do botão de consulta de cartas, extraindo este ID no `handleInlineQuery` e `handleChosenInlineResult` para localizar com precisão o jogo correspondente, eliminando problemas de dessincronização em múltiplos chats.

## Contexto atual
- `GameManager` mantém mapas globais em memória: `ChatIDGames`, `UserIDPlayers` e `UserIDCurrent`.
- Quando um jogo é limpo no lobby usando `/limpar`, os jogadores que haviam entrado nele continuam mapeados no `UserIDPlayers` e `UserIDCurrent`, criando "jogadores fantasmas" permanentes na memória com cartas atreladas a eles.
- Ao abrir o inline query usando o botão "Suas cartas", nenhuma informação sobre qual chat a ação pertence é enviada, fazendo com que o bot use apenas a referência global e possivelmente obsoleta `UserIDCurrent[userID]`.

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
1. **Refatorar limpeza de jogadores no GameManager**:
   - Adicionar o método privado `removeGamePlayers(game *Game)` no `GameManager` para encapsular a remoção de todos os jogadores de um jogo de `UserIDPlayers` e `UserIDCurrent`.
   - Adicionar o método público `CleanGames(chatID int64, user *UserData) (int, error)` que remove jogos não iniciados sob lock, invocando `removeGamePlayers` para cada jogo limpo.
2. **Atualizar `cmdCleanGames`**:
   - Modificar `cmdCleanGames` em `commands.go` para usar o novo método do `GameManager`.
3. **Passagem do ChatID na Inline Query**:
   - Em `actions.go` (função `sendNextMessage`), alterar o parâmetro de `WithSwitchInlineQueryCurrentChat` para passar o `chatID` formatado como string (ex: `fmt.Sprintf("%d", chatID)`).
4. **Resolução de Jogo via ChatID na Inline Query**:
   - Em `inline.go` (função `handleInlineQuery` e `handleChosenInlineResult`), tentar extrair o chatID a partir do texto da query (`query.Query` ou `result.Query`). Se for um ID válido, buscar o jogador correspondente a esse chat na lista do usuário e atualizar `gm.UserIDCurrent[userID]`.

## Passos detalhados

### Passo 1: Modificar `gamemanager.go`
1. Criar o método privado `removeGamePlayers(game *Game)` que realiza a lógica de remover todos os jogadores de `UserIDPlayers` e `UserIDCurrent`.
2. Substituir o bloco duplicado em `EndGame` pela chamada de `gm.removeGamePlayers(game)`.
3. Implementar o método `CleanGames(chatID int64, user *UserData) (int, error)` no `GameManager` para processar a remoção de jogos não iniciados e a limpeza dos seus respectivos jogadores de forma thread-safe.

### Passo 2: Modificar `commands.go`
1. Atualizar o `cmdCleanGames` para usar a nova função `gm.CleanGames(chatID, user)` em vez de realizar a manipulação direta sob lock no controller.

### Passo 3: Modificar `actions.go`
1. Em `sendNextMessage`, alterar:
   ```go
   tu.InlineKeyboardButton("Suas cartas").WithSwitchInlineQueryCurrentChat(fmt.Sprintf("%d", chatID))
   ```

### Passo 4: Modificar `inline.go`
1. Em `handleInlineQuery`, adicionar lógica para:
   - Ler `query.Query` e tentar converter para `int64`.
   - Se for um ID de chat válido, buscar se o usuário está em algum jogo desse chat.
   - Atualizar `gm.UserIDCurrent[userID]` com esse player encontrado.
2. Em `handleChosenInlineResult`, adicionar lógica similar para extrair o chatID de `result.Query` antes de processar a jogada para garantir que a carta seja jogada no jogo correto.

## Riscos
- **Conversão de ChatID**: Chat IDs do Telegram podem ser negativos (ex: grupos, supergrupos). O parse com `strconv.ParseInt` e sinal deve lidar com isso de forma nativa.
- **Tratamento de Concorrência**: Como o estado é compartilhado, todas as alterações devem ser feitas sob a proteção do mutex de `GameManager` (ou delegadas a métodos que utilizam locks apropriados).

## Impactos esperados
- Limpeza perfeita da memória do bot ao remover lobbies.
- Sincronização automática do inline query com o chat em que o usuário está interagindo no momento.
- Fim das mensagens falsas de "você não está em nenhum jogo" e fim das cartas fantasma na tela.

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
Utilizar o bot em múltiplos chats simultaneamente, criar jogos, cancelá-los com `/limpar`, interagir via inline query usando os botões e verificar se o estado do jogo e as cartas correspondem perfeitamente a cada chat sem vazamento de estado.

## Rollback
Caso ocorram erros inesperados, as alterações podem ser desfeitas via git revert:
```bash
git checkout -- gamemanager.go commands.go actions.go inline.go
```

## Observações
O inline feedback (ChosenInlineResult) configurado em 100% no BotFather é importante para garantir que, após selecionar a carta no inline query, a carta de fato seja jogada. O processamento sob o chatID correto no `handleChosenInlineResult` previne bugs caso o jogador clique rapidamente no botão de outro grupo.
