# Plano: adicionar-logs-e-seguranca

## Pedido do usuário
O usuário relatou que a carta jogada não sai da mão (deck) do usuário, e ele fica com a mesma carta para sempre.

## Objetivo
Identificar com precisão por que a carta não está saindo da mão do jogador e tornar o acesso aos mapas globais thread-safe para evitar data races.
1. Adicionar logs detalhados em todas as etapas da jogada: recepção do `ChosenInlineResult`, correspondência do jogador, execução de `doPlayCard`, verificação de `c.Equal` e remoção da carta no slice de `Player.Cards`.
2. Implementar métodos de leitura thread-safe em `GameManager` para recuperar jogadores e jogos ativos sob mutex lock, mitigando problemas concorrentes em goroutines de inline query.
3. Explicar e propor a persistência em arquivo local (`state.json`) caso o sumiço das cartas ocorra devido ao reinício do bot durante os testes (o que limpa a RAM e corrompe as referências das partidas ativas).

## Contexto atual
- A recepção do feedback do inline mode (`ChosenInlineResult`) e o processamento da jogada dependem do estado do `gm.UserIDCurrent` na RAM.
- A leitura desses mapas no inline query e chosen inline result atualmente ocorre sem lock de Mutex, o que pode causar data races ou leitura corrompida em Go sob acessos simultâneos.
- Se o bot é reiniciado para atualização de código durante uma partida, o estado na memória RAM zera, impedindo que novas ações inline encontrem o jogador correto daquela partida iniciada anteriormente.

## Arquivos analisados
- [inline.go](file:///home/gabriel/OpenCode/UnoBotGO/inline.go)
- [actions.go](file:///home/gabriel/OpenCode/UnoBotGO/actions.go)
- [player.go](file:///home/gabriel/OpenCode/UnoBotGO/player.go)
- [gamemanager.go](file:///home/gabriel/OpenCode/UnoBotGO/gamemanager.go)

## Arquivos que poderão ser modificados
- [inline.go](file:///home/gabriel/OpenCode/UnoBotGO/inline.go)
- [actions.go](file:///home/gabriel/OpenCode/UnoBotGO/actions.go)
- [player.go](file:///home/gabriel/OpenCode/UnoBotGO/player.go)
- [gamemanager.go](file:///home/gabriel/OpenCode/UnoBotGO/gamemanager.go)

## Estratégia de implementação
1. **Thread-Safe Getters no GameManager**:
   - Adicionar o método `GetCurrentPlayer(userID int64) *Player` em `GameManager` para recuperar com segurança o jogador ativo sob lock.
   - Adicionar o método `GetPlayersForUser(userID int64) []*Player` para recuperar a lista de sessões do usuário de forma segura.
2. **Logue de Diagnóstico nos Handlers**:
   - Em `inline.go` -> `handleChosenInlineResult`, logar o `userID`, `resultID`, e se o `player` foi localizado.
   - Em `actions.go` -> `doPlayCard`, logar a carta interpretada por `CardFromStr`.
   - Em `player.go` -> `Play`, logar a lista de cartas na mão, a comparação `c.Equal(card)` para cada elemento, e o resultado da remoção ou falha.
3. **Persistência de Estado (Opcional/Futuro)**:
   - Apresentar a opção de salvar o estado em arquivo JSON local para persistir partidas entre reinícios do bot.

## Passos detalhados

### Passo 1: Modificar `gamemanager.go`
1. Adicionar o método `GetCurrentPlayer`:
   ```go
   func (gm *GameManager) GetCurrentPlayer(userID int64) *Player {
   	gm.Lock()
   	defer gm.Unlock()
   	return gm.UserIDCurrent[userID]
   }
   ```
2. Adicionar o método `GetPlayersForUser`:
   ```go
   func (gm *GameManager) GetPlayersForUser(userID int64) []*Player {
   	gm.Lock()
   	defer gm.Unlock()
   	return gm.UserIDPlayers[userID]
   }
   ```

### Passo 2: Modificar `inline.go`
1. Substituir acessos diretos a `gm.UserIDCurrent[userID]` por `gm.GetCurrentPlayer(userID)`.
2. Substituir acessos diretos a `gm.UserIDPlayers[userID]` por `gm.GetPlayersForUser(userID)`.
3. Inserir logs detalhados em `handleChosenInlineResult` para podermos ver as ações do Telegram.

### Passo 3: Modificar `actions.go`
1. Adicionar logs em `doPlayCard` para registrar o início da ação de descarte.

### Passo 4: Modificar `player.go`
1. Adicionar logs detalhados em `Play` para verificar se `c.Equal` está batendo com alguma carta na mão do jogador.

## Riscos
- Excesso de logs em produção: os logs adicionados são simples e focados em depuração inicial.

## Impactos esperados
- Teremos visibilidade total no console do bot sobre exatamente onde o fluxo de remoção da carta está falhando ou se o `ChosenInlineResult` não está sendo recebido.
- Eliminação de potenciais crashes por data race em maps de Go.

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

### Execução
Executar o bot e monitorar a saída padrão (stdout) no terminal ao tentar jogar uma carta.

## Rollback
```bash
git checkout -- gamemanager.go inline.go actions.go player.go
```
