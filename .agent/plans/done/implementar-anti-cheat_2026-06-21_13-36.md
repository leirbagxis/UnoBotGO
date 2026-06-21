# Plano: implementar-anti-cheat

## Pedido do usuário
O usuário rejeitou a query string com contador no input e questionou como a atualização/mudança de cartas é feita no repositório original do `jh0ker/mau_mau_bot`.

## Objetivo
Implementar a invalidação de cache e controle de jogadas idêntica ao repositório original do `jh0ker/mau_mau_bot` utilizando o mecanismo de `AntiCheat`:
1. Formatar o ID dos resultados inline enviados ao Telegram com o padrão `<cardKey>:<AntiCheat>` (ex: `r_7:0`, `draw:0`).
2. No handler `handleChosenInlineResult` de `inline.go`, extrair o `resultID` e o `anti_cheat` recebidos.
3. Validar se o `anti_cheat` recebido é igual ao `player.AntiCheat` atual. Se for diferente, ignorar a jogada (evita jogadas obsoletas ou repetidas do histórico).
4. Se for igual, processar a jogada e incrementar o `player.AntiCheat++` de forma a mudar todos os IDs de resultados inline do jogador na rodada seguinte. Isso força o Telegram a invalidar o cache local dele e carregar as cartas reais atualizadas do bot imediatamente.

## Contexto atual
- A struct `Player` já possui o campo `AntiCheat int`.
- No entanto, o `inline.go` atualmente envia os IDs limpos (ex: `"r_7"`), permitindo que o Telegram cacheie a resposta.
- O repositório original do `mau_mau_bot` utiliza o formato `id += ':%d' % player.anti_cheat` para forçar o recarregamento de cartas a cada jogada e validar cheats.

## Arquivos analisados
- [inline.go](file:///home/gabriel/OpenCode/UnoBotGO/inline.go)
- [player.go](file:///home/gabriel/OpenCode/UnoBotGO/player.go)
- [results.go](file:///home/gabriel/OpenCode/UnoBotGO/results.go)

## Arquivos que poderão ser modificados
- [inline.go](file:///home/gabriel/OpenCode/UnoBotGO/inline.go)

## Estratégia de implementação
1. **Formatação de IDs das Inline Queries**:
   - Em `inline.go` -> `handleInlineQuery`, após gerar a lista de resultados, percorrer os resultados e formatar o campo `ID` de cada um anexando o `player.AntiCheat` atual (ex: `result.ID = fmt.Sprintf("%s:%d", result.ID, player.AntiCheat)`).
2. **Processamento do Feedback Inline com Validação**:
   - Em `inline.go` -> `handleChosenInlineResult`:
     - Extrair `resultID` e `antiCheat` da string usando `strings.SplitN(result.ResultID, ":", 2)`.
     - Validar se o `antiCheat` extraído coincide com `player.AntiCheat`.
     - Incrementar `player.AntiCheat++`.
     - Prosseguir com a execução do switch original com o `resultID` limpo.

## Passos detalhados

### Passo 1: Modificar `inline.go`
1. Na função `handleInlineQuery`, logo após o preenchimento da slice `results` e antes de chamar `AnswerInlineQuery`:
   ```go
   // Anexar anti-cheat nos IDs dos resultados
   for _, res := range results {
       switch r := res.(type) {
       case *telego.InlineQueryResultCachedSticker:
           r.ID = fmt.Sprintf("%s:%d", r.ID, player.AntiCheat)
       case *telego.InlineQueryResultArticle:
           r.ID = fmt.Sprintf("%s:%d", r.ID, player.AntiCheat)
       }
   }
   ```
2. Na função `handleChosenInlineResult`, no início do processamento:
   ```go
   parts := strings.SplitN(result.ResultID, ":", 2)
   if len(parts) != 2 {
       log.Printf("[ChosenInlineResult] Invalid resultID format: %s", result.ResultID)
       return
   }
   resultID := parts[0]
   antiCheatStr := parts[1]

   antiCheatVal, err := strconv.Atoi(antiCheatStr)
   if err != nil {
       log.Printf("[ChosenInlineResult] Error parsing antiCheatStr: %v", err)
       return
   }

   if antiCheatVal != player.AntiCheat {
       log.Printf("[ChosenInlineResult] Cheat attempt / obsolete action by %s! Got: %d, expected: %d", player.User.FirstName, antiCheatVal, player.AntiCheat)
       return
   }

   player.AntiCheat++
   ```
3. Substituir o uso de `result.ResultID` por `resultID` no switch subsequente do `handleChosenInlineResult`.

## Riscos
A conversão com `strconv.Atoi` exige o import do pacote `strconv`. Nós adicionaremos `"strconv"` de volta aos imports de `inline.go`.

## Impactos esperados
- As cartas na tela do usuário serão atualizadas instantaneamente após cada jogada.
- O campo de mensagem/input continuará 100% limpo (contendo apenas `@NomeDoBot `).
- O bot ficará protegido contra cliques repetidos em mensagens antigas de stickers (jogadas obsoletas).

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
Iniciar uma partida, selecionar as cartas na inline query e verificar se a atualização visual é instantânea e sem vazamentos de estado.

## Rollback
```bash
git checkout -- inline.go
```
