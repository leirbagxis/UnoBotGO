# Decisões

## Data
2026-06-21

## Contexto
O bot mantinha estado global em memória e apresentava problemas de cartas fantasmas ao limpar lobbies não iniciados (vazamento de referências) e exibição incorreta de cartas em inline query ao jogar em múltiplos chats devido à falta de contexto do chat nas requisições inline.

## Decisão tomada
1. Reverter a exibição de ChatID no input do inline query, mantendo a caixa de texto limpa.
2. Sincronizar o ponteiro global de jogo ativo `UserIDCurrent[userID]` de forma proativa e invisível ao usuário:
   - A cada mensagem ou comando recebido do usuário no grupo (em `handleMessage`), caso ele tenha um player associado àquele chat.
   - A cada transição de turno no jogo (ao iniciar a partida, ao passar o turno após uma jogada inline, ao kickar, skippar ou quando um jogador sai).
3. Encapsular a limpeza de referências de jogadores de um jogo na função privada `removeGamePlayers` do `GameManager` e utilizá-la tanto no encerramento normal/forçado quanto na limpeza de lobbies não iniciados (`CleanGames`).
4. Implementar getters thread-safe (`GetCurrentPlayer` e `GetPlayersForUser`) para ler os mapas globais sob Mutex.
5. Adotar o mecanismo de `AntiCheat` idêntico ao repositório original `jh0ker/mau_mau_bot` para invalidar naturalmente o cache local de inline queries no Telegram:
   - Anexar o sufixo `:AntiCheat` nos IDs dos resultados inline (ex: `r_7:0`).
   - Extrair, validar e incrementar o `player.AntiCheat++` no `handleChosenInlineResult`.
6. Corrigir a rotação de turno (`Turn`) e listagem de jogadores (`Players`) em `game.go` para navegar no sentido anterior (`.Prev`) quando `Reversed` for `true`.
7. Otimizar a liberação de locks de `game.Lock()` no `handleChosenInlineResult` para ser realizada de forma antecipada (antes de chamadas de rede do Telegram e do `UpdateCurrentPlayer`), prevenindo deadlocks.

## Motivo
Garante a integridade do estado e evita o descompasso na sincronização do inline query sem expor IDs internos no campo de texto de digitação do Telegram. A liberação antecipada de locks previne deadlocks de Mutex e otimiza a latência. A correção em game.go alinha a rotação ao comportamento real de jogo UNO.

## Impacto
O bot agora é estável em cenários com múltiplos jogos paralelos e limpa totalmente a memória ao cancelar lobbies, sem deixar resíduos de cartas ou sessões fantasma para os usuários. A carta reverse agora muda de fato a rotação do jogo e não causa travamentos concorrentes.
