# Plano: corrigir-cache-inline-cards

## Pedido do usuário
A carta jogada não sai do deck do jogador. O usuário quer que funcione igual ao bot original (mau_mau_bot), sem nenhum número no inline — apenas `@usernamebot`.

## Diagnóstico (confirmado)

A carta **é removida corretamente** do servidor (`player.go:100`).

**Causa raiz:** No bot Python original (`results.py:113`), os resultados de cartas cinzas usam **UUIDs aleatórios** como ID:
```python
Sticker(str(uuid4()), sticker_file_id=c.STICKERS_GREY[str(card)], ...)
```

Isso faz com que cada inline query tenha resultados com IDs completamente novos, forçando o Telegram a invalidar o cache client-side e buscar resultados frescos.

No bot Go (`results.go`), os IDs são **fixos** para todos os resultados:
- Cartas jogáveis: `r7`, `b3`, etc. (`results.go:40`)
- Cartas cinzas: `grey_r7` (`results.go:53`)
- Ações: `draw`, `pass`, `call_bluff`, `gameinfo` (`results.go:69-107`)

Como os IDs não mudam, o Telegram client cacheia e reexibe resultados antigos.

## Solução
Gerar IDs únicos (UUIDs) para **todos** os resultados das inline queries, igual ao bot original. Sem adicionar números visíveis ao botão.

## Arquivos que serão modificados
- `results.go` — usar UUIDs para todos os result IDs

## Passos detalhados

### 1. Adicionar import de UUID em `results.go`

Adicionar `"github.com/google/uuid"` aos imports.

### 2. Modificar `addCard` para usar UUID

```go
func addCard(game *Game, card *Card, results *[]telego.InlineQueryResult, canPlay bool) {
    key := card.String()
    if canPlay {
        stickerID := Stickers[key]
        if stickerID == "" {
            return
        }
        *results = append(*results, &telego.InlineQueryResultCachedSticker{
            Type:                "sticker",
            ID:                  uuid.New().String(),
            StickerFileID:       stickerID,
            InputMessageContent: nil,
        })
    } else {
        greyID := StickersGrey[key]
        if greyID == "" {
            return
        }
        info := gameInfo(game)
        *results = append(*results, &telego.InlineQueryResultCachedSticker{
            Type:                "sticker",
            ID:                  uuid.New().String(),
            StickerFileID:       greyID,
            InputMessageContent: info,
        })
    }
}
```

### 3. Modificar `addDraw` para usar UUID

```go
func addDraw(player *Player, results *[]telego.InlineQueryResult) {
    // ... (manter lógica existente)
    *results = append(*results, &telego.InlineQueryResultCachedSticker{
        Type:          "sticker",
        ID:            uuid.New().String(),
        StickerFileID: Stickers["option_draw"],
        // ...
    })
}
```

### 4. Modificar `addPass` para usar UUID

```go
ID: uuid.New().String(),
```

### 5. Modificar `addCallBluff` para usar UUID

```go
ID: uuid.New().String(),
```

### 6. Modificar `addGameInfo` para usar UUID

```go
ID: uuid.New().String(),
```

### 7. Modificar `addChooseColor` para usar UUID

```go
*results = append(*results, &telego.InlineQueryResultArticle{
    Type:  "article",
    ID:    uuid.New().String(),
    // ...
})
```

### 8. Modificar `addNoGame` e `addNotStarted` para usar UUID

```go
ID: uuid.New().String(),
```

### 9. Modificar modos (`addModeClassic`, etc.) para usar UUID

```go
ID: uuid.New().String(),
```

### 10. Adicionar dependência

```bash
go get github.com/google/uuid
```

## Por que isso funciona

No bot Python original:
- Cada inline query retorna resultados com IDs únicos (UUIDs)
- O Telegram client não tem cache para esses IDs → busca resultados frescos do bot
- A carta removida server-side não aparece mais

No bot Go atual:
- Resultados têm IDs fixos (`grey_r7`, `draw`, etc.)
- O Telegram client cacheia por esses IDs → mostra resultados antigos
- A carta removida server-side continua aparecendo

## Riscos
- Nenhum risco de segurança ou concorrência
- UUIDs são gerados em memória, sem I/O

## Impactos esperados
- As cartas atualizarão na hora ao serem jogadas
- O botão continua mostrando apenas `@usernamebot` (sem números)
- Comportamento idêntico ao bot original

## Compatibilidade
- Linux, macOS, Windows, Docker

## Como testar

### Build
```bash
go build -o UnoGoBot
```

### Teste manual
1. Iniciar partida com 2+ jogadores
2. Jogar uma carta
3. Clicar em "Suas cartas" → carta jogada não deve aparecer
4. Verificar que o campo de inline query mostra apenas `@usernamebot`

## Rollback
```bash
git checkout -- results.go
go mod tidy
```
