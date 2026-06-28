# Plano: Ranking por usuário nos grupos (Opção A)

## Pedido do usuário
Mostrar a posição do usuário em cada grupo que ele jogou, ao invés de apenas o ranking do grupo atual.

## Objetivo
- No privado: `/ranking` lista os grupos onde o usuário jogou com sua posição
- No grupo: `/ranking` continua mostrando ranking completo do grupo
- Bot sai do grupo: limpa todo o histórico (wins, challenges, etc.)

## Contexto atual
- `/ranking` sempre mostra ranking do grupo atual
- Não existe verificação de chat type (funciona no privado, mas retorna vazio)
- Não existe limpeza automática quando bot sai do grupo

## Arquivos que serão modificados
- `ranking.go` — nova query `GetUserRankingAcrossGroups()`
- `cmd_ranking.go` — lógica de exibição baseada no chat type
- `main.go` — handler para `MyChatMember` (bot sai do grupo)

## Estratégia de implementação

### 1. Nova query SQL em ranking.go

```sql
-- Posição do usuário em cada grupo
WITH user_wins AS (
    SELECT chat_id, COUNT(*) as wins
    FROM wins
    WHERE user_id = $1
    GROUP BY chat_id
),
user_ranks AS (
    SELECT 
        uw.chat_id,
        uw.wins,
        (SELECT COUNT(*) + 1 
         FROM wins w2 
         WHERE w2.chat_id = uw.chat_id 
         AND w2.user_id != $1
         GROUP BY w2.user_id
         HAVING COUNT(*) > uw.wins) as rank
    FROM user_wins uw
)
SELECT chat_id, wins, rank FROM user_ranks
```

Simplificado para:
```sql
WITH user_wins AS (
    SELECT chat_id, COUNT(*) as wins
    FROM wins WHERE user_id = $1
    GROUP BY chat_id
)
SELECT 
    uw.chat_id,
    uw.wins,
    (SELECT COUNT(DISTINCT w2.user_id) + 1
     FROM wins w2 
     WHERE w2.chat_id = uw.chat_id
     AND w2.user_id != $1
     AND (SELECT COUNT(*) FROM wins w3 WHERE w3.chat_id = uw.chat_id AND w3.user_id = w2.user_id) > uw.wins
    ) as rank
FROM user_wins uw
ORDER BY uw.wins DESC
```

### 2. Alteração em cmdRanking

```go
func cmdRanking(bot *telego.Bot, msg telego.Message, chatID int64) {
    if msg.Chat.Type == telego.ChatTypePrivate {
        // Mostra posição do usuário nos grupos
        showUserRankingAcrossGroups(bot, msg, chatID, user)
        return
    }
    // Comportamento atual (ranking do grupo)
    ranking := rankingStore.GetRanking(chatID, "mensal")
    // ...
}
```

### 3. Nova função showUserRankingAcrossGroups

```go
func showUserRankingAcrossGroups(bot *telego.Bot, msg telego.Message, chatID int64, user *UserData) {
    rankings := rankingStore.GetUserRankingAcrossGroups(user.ID)
    if len(rankings) == 0 {
        sendMessage(bot, chatID, "Você ainda não venceu nenhum jogo.")
        return
    }
    text := "🏆 Sua posição nos grupos:\n\n"
    for _, r := range rankings {
        text += fmt.Sprintf("#%d — %s (%d vitórias)\n", r.Rank, r.GroupName, r.Wins)
    }
    sendMessage(bot, chatID, text)
}
```

### 4. Limpeza quando bot sai do grupo

Em `main.go`, adicionar handler para `MyChatMember`:

```go
// Detectar quando bot é removido do grupo
if myChatMember.NewChatMember.User.ID == bot.Self.ID {
    if myChatMember.NewChatMember.Status == "left" || myChatMember.NewChatMember.Status == "kicked" {
        rankingStore.CleanGroupData(myChatMember.Chat.ID)
    }
}
```

### 5. Nova função CleanGroupData em ranking.go

```go
func (rs *RankingStore) CleanGroupData(chatID int64) {
    _, _ = rs.db.Exec("DELETE FROM wins WHERE chat_id = $1", chatID)
    _, _ = rs.db.Exec("DELETE FROM challenge_history WHERE chat_id = $1", chatID)
    _, _ = rs.db.Exec("DELETE FROM group_settings WHERE chat_id = $1", chatID)
    log.Printf("Dados do grupo %d limpos", chatID)
}
```

## Passos detalhados

1. Adicionar struct `UserGroupRank` em ranking.go
2. Implementar `GetUserRankingAcrossGroups()` em ranking.go
3. Implementar `CleanGroupData()` em ranking.go
4. Modificar `cmdRanking()` para verificar chat type
5. Modificar `cmdRankingX1()` para verificar chat type
6. Modificar callbacks de ranking para verificar chat type
7. Adicionar handler para `MyChatMember` em main.go
8. Criar migration para limpar dados de grupos antigos (opcional)

## Riscos

- Query SQL complexa pode ser lenta com muitos dados
- Nome do grupo não está no banco — precisa buscar via Telegram API ou armazenar
- Bot precisa de permissão para receber updates de `MyChatMember`

## Impactos esperados

- Usuário vê sua posição geral no privado
- Ranking continua funcionando igual nos grupos
- Dados de grupos abandonados são limpos automaticamente

## Como testar

### Build
```bash
go build -o unobot .
```

### Testes
```bash
go test -v ./...
```

### Execução
1. Mandar `/ranking` no privado — deve mostrar grupos
2. Mandar `/ranking` no grupo — deve mostrar ranking do grupo
3. Remover bot do grupo — dados devem ser limpos

## Observations

- Nome do grupo será armazenado na tabela wins quando usuário vence
- Adicionar coluna `group_name` na tabela wins (migration)
- Atualizar função `RecordWin` para salvar o nome do grupo
