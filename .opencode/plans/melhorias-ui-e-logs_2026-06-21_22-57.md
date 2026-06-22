# Plano: melhorias-ui-e-logs

## Pedido do usuário
Remover DEBUG logs, melhorar logs, usar nome com link embutido, `/start` privado aprimorado, `/desafio` na ajuda.

## Arquivos analisados
- main.go
- commands.go
- results.go (displayName)
- actions.go (sendMessage, sendNextMessage)
- inline.go

## Mudanças

### 1. Remover DEBUG logs da telego
- **main.go:36**: `telego.WithDefaultDebugLogger()` → remover o option (default logger é silencioso)
- Efeito: para de spamar `[Sun Jun 21 22:57:10...] DEBUG API call/getUpdates`

### 2. Nome com link embutido (tg://user?id=)
- **results.go `displayName`**: retornar `<a href="tg://user?id={ID}">{FirstName}</a>` usando HTML
- **actions.go `sendMessage`**: adicionar `ParseMode: telego.ModeHTML`
- **actions.go `sendNextMessage`**: adicionar `ParseMode: telego.ModeHTML`
- **commands.go** onde usa `displayName` em `sendMessage`/`sendNextMessage`: já funciona porque sendMessage agora parseia HTML
- Efeito: "João (@joao)" vira "João" com link azul clicável. Chat fica mais limpo.

### 3. `/start` no privado — página inicial bonita
- **commands.go `cmdStartGame`** (`case telego.ChatTypePrivate`): em vez de só chamar `cmdHelp`, enviar mensagem com:
  - Título: "🎮 UnoBotGO"
  - Descrição do bot
  - Como usar (grupo)
  - Lista de comandos com botões inline (novo, entrar, etc.)
  - Pode usar `sendMessage` normal com ParseMode HTML

### 4. `/desafio` na ajuda
- **commands.go `cmdHelp`**: adicionar linha `/desafio - Desafiar alguém para um MD1/MD3/MD5`

### 5. Logs mais limpos
- Remover logs verbosos internos (ex: logs de anti-cheat tolerado podem ser `log.Println` em vez de `log.Printf` ou removidos)
- Manter logs essenciais: conexão, erros, ações de jogo

## Riscos
- HTML ParseMode: nomes do Telegram podem conter `<`, `>`, `&` → precisam de escape. Usar `html.EscapeString(user.FirstName)` no `displayName`.
- Se `sendMessage` for usado em outros contexts sem querer HTML, pode quebrar. Verificar todos os usos.

## Como testar
```bash
make local
```
- Verificar se DEBUG logs sumiram
- Testar "/start" no privado
- Jogar uma partida e ver se nomes aparecem como links
- Ver "/ajuda" tem `/desafio`
