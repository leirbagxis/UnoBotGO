# Plano: comando /modo — modo padrão global por grupo

## Pedido do usuário
Comando `/modo` que seta o modo de jogo para o grupo de forma global, persistida no banco de dados.

## Respostas do usuário
1. **Persistência:** Salvar no banco (global para o grupo)
2. **Escopo:** B — Usado como padrão quando o criador NÃO escolhe modo inline
3. **Permissão:** Qualquer pessoa
4. **Formato:** Inline (botões)
5. **Feedback:** Lista de botões inline para escolher
6. **Modos:** Todos exceto "test" (classic, fast, wild, text, caseiro)

## Objetivo
Permitir que qualquer membro do grupo defina o modo padrão de jogo via `/modo`, com persistência no PostgreSQL. O criador pode sobrescrever escolhendo modo via inline query ao iniciar.

## Contexto atual
- `NewGame()` usa `GetDefaultGamemode()` → lê env var `DEFAULT_GAMEMODE` (default "fast")
- Criador seleciona modo via inline query ao iniciar (override temporário)
- `RankingStore.init()` cria tabelas no startup
- Callbacks inline usam `handleCallbackQuery` em `commands.go`

## Arquivos analisados
- ranking.go (persistência, init de tabelas)
- game.go (NewGame, SetMode)
- inline.go (seleção de modo via inline query)
- commands.go (handleCallbackQuery)
- cmd_game.go (cmdStartGame)
- main.go (registro de comandos)
- config.go (GetDefaultGamemode)

## Arquivos que serão modificados
- ranking.go — nova tabela + funções Get/Set
- game.go — NewGame usa group default
- cmd_game.go — novo comando /modo
- commands.go — callback handler para botões de modo
- main.go — registro do comando

## Estratégia de implementação

### 1. Tabela group_settings (ranking.go)
Adicionar na `init()` do RankingStore:
```sql
CREATE TABLE IF NOT EXISTS group_settings (
    chat_id BIGINT PRIMARY KEY,
    default_mode TEXT NOT NULL DEFAULT 'fast',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)
```

### 2. Funções de persistência (ranking.go)
```go
func (rs *RankingStore) GetGroupDefaultMode(chatID int64) string
func (rs *RankingStore) SetGroupDefaultMode(chatID int64, mode string)
```
- `GetGroupDefaultMode`: SELECT do banco; retorna fallback `GetDefaultGamemode()` se não encontrar
- `SetGroupDefaultMode`: UPSERT (INSERT ON CONFLICT UPDATE)

### 3. Integração no NewGame (game.go)
Modificar `NewGame()` para:
1. Verificar `rankingStore.GetGroupDefaultMode(chatID)` primeiro
2. Usar resultado como modo padrão (em vez de só `GetDefaultGamemode()`)
3. Criador pode sobrescrever via inline query normalmente

### 4. Comando /modo (cmd_game.go)
Nova função `cmdModo`:
- Verifica se é grupo (senão, erro)
- Mostra inline keyboard com 5 modos (sem test)
- Formato: "Modo atual: <modo>\nEscolha o modo padrão para este grupo:"
- Callback data: `set_mode_classic`, `set_mode_fast`, etc.

### 5. Callback handler (commands.go)
Adicionar ao `handleCallbackQuery`:
- `set_mode_classic`, `set_mode_fast`, `set_mode_wild`, `set_mode_text`, `set_mode_caseiro`
- Chama `rankingStore.SetGroupDefaultMode(chatID, mode)`
- Edita a mensagem confirmando: "Modo definido para <modo> 🎯"

### 6. Registro (main.go)
Adicionar `{Command: "modo", Description: "Definir modo padrão do grupo"}`

### 7. Ajuda (cmd_game.go)
Atualizar `cmdHelp` com `/modo - Definir modo padrão do grupo`

## Passos detalhados

1. ranking.go: Adicionar CREATE TABLE na init()
2. ranking.go: Adicionar GetGroupDefaultMode e SetGroupDefaultMode
3. game.go: Modificar NewGame para usar group default
4. cmd_game.go: Adicionar cmdModo com inline keyboard
5. commands.go: Adicionar callback handlers para set_mode_*
6. main.go: Registrar /modo
7. cmd_game.go: Atualizar cmdHelp
8. Build + testes

## Riscos
- Tabela group_settings pode não existir quando bot inicia (resolvido pela init())
- rankingStore pode ser nil na init() do bot (resolvido: rankingStore é inicializado antes do polling)
- Modo "test" não deve aparecer nas opções do /modo (decisão do usuário)

## Impactos esperados
- Grupos podem definir modo padrão sem depender de env var
- Criador continua podendo sobrescrever via inline query
- Persiste entre reinícios do bot

## Como testar

### Build
```bash
go build -o unobot .
```

### Testes
```bash
go test -v ./...
```

### Execução manual
1. Em um grupo, digite `/modo`
2. Verificar que aparecem 5 botões de modo
3. Tocar em um botão → mensagem editada confirmando
4. Criar jogo com `/novo` → verificar que o modo padrão é o escolhido
5. Criador selecionar modo diferente via inline → verificar que prevalece

## Rollback
Remover a CREATE TABLE da init() e as funções Get/SetGroupDefaultMode.

## Observações
- O modo "test" é excluído das opções do /modo (decisão do usuário)
- A tabela group_settings é criada automaticamente na init()
- GetGroupDefaultMode retorna o fallback do env var se não houver registro
