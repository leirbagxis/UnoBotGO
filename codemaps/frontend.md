# Frontend Structure

**Generated:** 2026-06-25T15:26:00-03:00

## Status: Not Applicable

This project is a **Go backend-only application** (Telegram bot). There is no web frontend, no browser UI, and no JavaScript/TypeScript code.

## User Interface

The "frontend" is the **Telegram client** itself. The bot interacts via:

### 1. Chat Commands
Users type commands in group chat:
- `/novo` — create game
- `/entrar` — join game
- `/iniciar` — start game
- etc.

### 2. Inline Queries (Card UI)
Users type `@bot_username` in the message box to see:
- Playable card stickers (tappable)
- Greyed-out cards (info-only, show game state on tap)
- Action buttons: Draw, Pass, Call Bluff
- Color picker (when playing wild cards)
- Mode selector (for game starter)

### 3. Inline Keyboards
- Challenge accept/config/cancel buttons
- Match start/next buttons
- Ranking period toggles

### 4. Rich Messages
- HTML-formatted messages with user links
- Card sticker images sent as Telegram stickers
- Emoji reactions on wins and UNO calls

## UI Flow

```
Chat message flow:
  User: /novo
  Bot: "Novo jogo criado!"

  User: /entrar
  Bot: "Você entrou no jogo! Faltam X jogador(es)..."

  User: @bot [tap card sticker]
  Bot processes play, sends next turn message
```

```
Inline query flow:
  User types @bot in message field
  ─► Telegram sends inline query to bot
  ─► Bot returns ResultList with:
      ├── Card stickers (playable = colored, else grey)
      ├── Draw/Pass/Bluff action buttons
      └── Game info snapshot
  ─► User taps a result
  ─► Telegram sends chosen_inline_result
  ─► Bot executes action (play/draw/pass/color)
```

## No Frontend Code

There is no:
- HTML/CSS/JS
- Web framework
- Mobile app code
- API endpoints
- WebSocket connections
- Client-side state

All UI rendering is handled by the Telegram client based on the structured responses sent by the bot.

## Diff from Previous

- **No previous codemap** — initial generation (100% new)
