# Plano: corrigir-logica-reverse-e-deadlock

## Pedido do usuário
O usuário relatou que o bot travou (deadlock) especificamente após ele jogar uma carta de inversão de turno (`reverse`).

## Objetivo
Identificar e corrigir os potenciais deadlocks concorrentes e implementar a lógica correta da inversão do fluxo de turno (`Reversed`):
1. Reduzir o escopo do lock de `game.Lock()` em `handleChosenInlineResult` (`inline.go`), liberando o Mutex do jogo imediatamente após processar a alteração do estado na memória RAM e antes de realizar chamadas de rede lentas (enviar mensagens ao Telegram no `afterAction`), eliminando deadlocks de concorrência.
2. Corrigir as funções `Turn()` e `Players()` em `game.go` para que o turno do jogo siga na direção reversa (`CurrentPlayer.Prev`) quando `Reversed` for `true`, resolvendo a lógica de inversão de fluxo de jogo em UNO que estava quebrada.

## Contexto atual
- Atualmente, o `game.Lock()` é mantido ativo por toda a duração da execução do `handleChosenInlineResult`, inclusive durante chamadas bloqueantes de rede para enviar mensagens ao chat no Telegram.
- A lógica de turnos (`Turn`) e listagem de jogadores (`Players`) em `game.go` sempre assume a direção convencional (`.Next`), ignorando completamente o estado da flag `Reversed` quando a carta reverse é jogada em partidas com mais de 2 jogadores.

## Arquivos analisados
- [game.go](file:///home/gabriel/OpenCode/UnoBotGO/game.go)
- [inline.go](file:///home/gabriel/OpenCode/UnoBotGO/inline.go)

## Arquivos que poderão ser modificados
- [game.go](file:///home/gabriel/OpenCode/UnoBotGO/game.go)
- [inline.go](file:///home/gabriel/OpenCode/UnoBotGO/inline.go)

## Estratégia de implementação
1. **Redução de Mutex no inline.go**:
   - Salvar as propriedades `game.Started` e `game.CurrentPlayer` do jogo em variáveis locais sob lock do game.
   - Chamar `game.Unlock()` antes de entrar no bloco `afterAction`.
   - Executar a lógica de rede do `afterAction` e atualização do GameManager utilizando as variáveis locais, sem segurar o Mutex do jogo.
2. **Atualização da Direção de Jogada em game.go**:
   - Em `Turn()`, verificar se `g.Reversed` é verdadeiro e passar o turno para `g.CurrentPlayer.Prev` (ou `g.CurrentPlayer.Next` caso falso).
   - Em `Players()`, iterar sobre a lista circular na direção correspondente à flag `g.Reversed`.

## Passos detalhados

### Passo 1: Modificar `game.go`
1. Atualizar a função `Players()` para percorrer na direção correta de `g.Reversed`:
   ```go
   func (g *Game) Players() []*Player {
   	if g.CurrentPlayer == nil {
   		return nil
   	}
   	var players []*Player
   	current := g.CurrentPlayer
   	players = append(players, current)
   	var it *Player
   	if g.Reversed {
   		it = current.Prev
   	} else {
   		it = current.Next
   	}
   	for it != nil && it != current {
   		players = append(players, it)
   		if g.Reversed {
   			it = it.Prev
   		} else {
   			it = it.Next
   		}
   	}
   	return players
   }
   ```
2. Atualizar a função `Turn()` para alterar a vez na direção correta:
   ```go
   func (g *Game) Turn() {
   	log.Println("Next Player")
   	if g.Reversed {
   		g.CurrentPlayer = g.CurrentPlayer.Prev
   	} else {
   		g.CurrentPlayer = g.CurrentPlayer.Next
   	}
   	g.CurrentPlayer.Drew = false
   	g.CurrentPlayer.TurnStarted = time.Now()
   	g.ChoosingColor = false
   }
   ```

### Passo 2: Modificar `inline.go`
1. Em `handleChosenInlineResult`, alterar o fim do switch para armazenar variáveis locais, liberar o lock do game e processar o `afterAction` sem lock:
   ```go
   	// ... fim do switch
   	}

   	started := game.Started
   	var nextPlayerUser *UserData
   	if game.CurrentPlayer != nil {
   		nextPlayerUser = game.CurrentPlayer.User
   	}
   	game.Unlock()

   afterAction:
   	if started && nextPlayerUser != nil {
   		gm.UpdateCurrentPlayer(game)
   		nextMsg := fmt.Sprintf("Próximo jogador: %s", displayName(nextPlayerUser))
   		sendNextMessage(bot, game.ChatID, nextMsg)
   	}
   ```
2. Remover o `defer game.Unlock()` da linha 160.

## Riscos
Nenhum risco de inconsistência, visto que as atualizações de estado do jogo na memória RAM ocorrem 100% dentro do escopo protegido do Mutex, e a liberação antecipada apenas remove o bloqueio durante as requisições de rede.

## Impactos esperados
- O jogo UNO passará a rodar no sentido correto (anti-horário) quando a carta `Reverse` for jogada em partidas multi-grupo.
- Fim de travamentos e deadlocks concorrentes.

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
Jogar cartas de Reverse e verificar se o turno muda de direção perfeitamente e se o bot continua operando sem travar ou congelar.

## Rollback
```bash
git checkout -- game.go inline.go
```
