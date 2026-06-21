# Plano: adicionar-comando-depuracao

## Pedido do usuário
O usuário relatou novamente que as cartas não atualizam e continuam na mão (deck) do usuário mesmo após jogar.

## Objetivo
Implementar uma ferramenta de auditoria imediata para diagnosticar se a carta está sendo de fato removida da RAM do bot (o que isolaria o problema a um cache visual do Telegram) ou se o processamento da jogada falhou por completo na memória do bot.
1. Criar o comando `/mao` (ou `/cartas`) para que o usuário possa consultar instantaneamente a lista de cartas reais presentes na struct do jogador na memória RAM do bot.
2. Adicionar o comando `/mao` no painel de ajuda (`/ajuda`) do bot.
3. Explicar ao usuário como realizar o teste diagnóstico e ler os logs no seu terminal.

## Contexto atual
- Sem um comando para expor a mão do jogador em formato de texto, não conseguimos saber se a carta jogada foi removida de `p.Cards` (e o Telegram cacheou a inline query visualmente) ou se a função `Play` não foi executada.
- O projeto compila normalmente e já possui logs robustos implementados na etapa anterior.

## Arquivos analisados
- [commands.go](file:///home/gabriel/OpenCode/UnoBotGO/commands.go)

## Arquivos que poderão ser modificados
- [commands.go](file:///home/gabriel/OpenCode/UnoBotGO/commands.go)

## Estratégia de implementação
1. **Adicionar comando `/mao`**:
   - Em `commands.go`, estender o switch de `handleMessage` para suportar `/mao`.
   - Implementar `cmdMyHand` para listar no chat de forma legível (texto) quais cartas o jogador possui na mão no backend.
2. **Atualizar Texto de Ajuda**:
   - Em `commands.go` -> `cmdHelp`, incluir o comando `/mao` na lista de comandos.

## Passos detalhados

### Passo 1: Modificar `commands.go`
1. No switch da função `handleMessage`, adicionar:
   ```go
   case "/mao":
   	cmdMyHand(bot, message, user, chatID)
   ```
2. Adicionar a implementação da função `cmdMyHand` no final do arquivo:
   ```go
   func cmdMyHand(bot *telego.Bot, msg telego.Message, user *UserData, chatID int64) {
   	player := gm.PlayerForUserInChat(user, chatID)
   	if player == nil {
   		sendMessage(bot, chatID, "Você não está em nenhum jogo neste grupo.")
   		return
   	}

   	if len(player.Cards) == 0 {
   		sendMessage(bot, chatID, "Você não tem nenhuma carta na mão.")
   		return
   	}

   	var cardNames []string
   	for _, c := range player.Cards {
   		cardNames = append(cardNames, c.Repr())
   	}

   	sendMessage(bot, chatID, fmt.Sprintf("Suas cartas na memória do bot: %s", strings.Join(cardNames, ", ")))
   }
   ```
3. Na função `cmdHelp` (`commands.go`), adicionar a linha:
   `/mao - Ver cartas na memória (debug)`

## Riscos
Nenhum risco técnico ou de concorrência identificado, pois o método apenas lê dados sob lock implícito já provido por `gm.PlayerForUserInChat` (que usa lock de `gm`).

## Impactos esperados
Ao jogar uma carta, se ela ainda aparecer no inline query, o usuário digita `/mao`.
- Se a carta NÃO estiver no `/mao`, significa que a remoção na RAM funcionou perfeitamente e o problema é apenas cache local do Telegram (podendo ser resolvido alterando configurações de cache ou limpando dados do app).
- Se a carta CONTINUAR no `/mao`, significa que a jogada não foi processada e o bot não está recebendo o `ChosenInlineResult` ou está falhando antes de remover a carta.

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
Digitar `/mao` no chat durante a partida antes e depois de jogar uma carta e observar a atualização no chat.

## Rollback
```bash
git checkout -- commands.go
```
