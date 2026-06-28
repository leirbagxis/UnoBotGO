# Backend Structure

**Generated:** 2026-06-25T15:26:00-03:00
**Language:** Go 1.26.3
**Package:** `main` (monolithic, no sub-packages)

## File Inventory (14 source files)

| File       | Lines | Role                         |
|------------|-------|------------------------------|
| main.go    | 106   | Entrypoint, bot init, polling|
| commands.go| 835   | Command handlers, callbacks  |
| game.go    | 148   | Core game logic              |
| player.go  | 190   | Player state & actions       |
| deck.go    | 80    | Deck shuffle/draw/dismiss    |
| card.go    | 155   | Card types, stickers, icons  |
| config.go  | 63    | Env-based configuration      |
| gamemanager.go| 446 | State container & mutation   |
| inline.go  | 263   | Inline query handling        |
| actions.go | 214   | Game action executors        |
| results.go | 288   | Inline result builders       |
| match.go   | 182   | Challenge (1v1) system       |
| errors.go  | 18    | Sentinel errors              |
| ranking.go | 176   | PostgreSQL persistence       |

## Layers

### 1. Entrypoint Layer
- **main.go** — loads config, connects DB, starts bot long polling loop

### 2. Routing Layer
- **commands.go** — `handleMessage()` dispatches by command prefix
- **inline.go** — `handleInlineQuery()` + `handleChosenInlineResult()` for inline UI

### 3. State Layer (GameManager)
- **gamemanager.go** — in-memory state container with mutex-guarded access
  - `ChatIDGames` → `map[int64][]*Game`
  - `UserIDPlayers` → `map[int64][]*Player`
  - `ChatIDMatch` → `map[int64]*Match`

### 4. Domain Layer
- **game.go** — `Game` struct + turn logic, card playing, color choosing
- **player.go** — `Player` + `UserData` structs, draw/hand management
- **deck.go** — `Deck` struct with fill/shuffle/draw/graveyard
- **card.go** — `Card` struct, card constants, sticker maps
- **match.go** — `Match` + `MatchState` for 1v1 challenges
- **errors.go** — sentinel errors for flow control

### 5. Action Layer
- **actions.go** — `doPlayCard`, `doDraw`, `doSkip`, `doCallBluff`, message helpers

### 6. Presentation Layer
- **results.go** — inline query result builders (sticker grid, mode selection, game info)

### 7. Persistence Layer
- **ranking.go** — `RankingStore` wrapping `*sql.DB` for win tracking & head-to-head

### 8. Configuration Layer
- **config.go** — reads env vars with typed fallbacks (TOKEN, DATABASE_URL, WAITING_TIME, etc.)

## Function Catalog (85 functions)

### Message Handlers (commands.go)
| Function | Trigger | Purpose |
|----------|---------|---------|
| `handleMessage` | any text starting with `/` | Command dispatcher |
| `cmdNewGame` | `/novo` | Create game lobby |
| `cmdJoinGame` | `/entrar` | Join existing game |
| `cmdStartGame` | `/iniciar` or `/start` | Start game (needs ≥2 players) |
| `cmdLeaveGame` | `/sair` | Leave game or match |
| `cmdKillGame` | `/kill` | End game (owner/admin only) |
| `cmdCloseGame` | `/fechar` | Close lobby |
| `cmdOpenGame` | `/abrir` | Open lobby |
| `cmdSkipPlayer` | `/pular` | Skip current player |
| `cmdKickPlayer` | `/kick` | Kick player (reply-based) |
| `cmdCleanGames` | `/limpar` | Remove unstarted games |
| `cmdNotifyMe` | `/notificar` | Opt-in for game notifications |
| `cmdHelp` | `/ajuda` | Show help text |
| `cmdModes` | `/modos` | Show game modes |
| `cmdRanking` | `/ranking` | Show monthly ranking |
| `cmdRankingDiario` | `/diario` | Show daily ranking |
| `cmdRankingSemanal` | `/semanal` | Show weekly ranking |
| `cmdDesafio` | `/desafio` | Create 1v1 challenge |
| `cmdRankingX1` | `/rankingx1` | Show head-to-head stats |
| `handleCallbackQuery` | inline button tap | Route to challenge or ranking |
| `handleChallengeCallback` | challenge buttons | Handle challenge lifecycle |

### Game Actions (actions.go)
| Function | Purpose |
|----------|---------|
| `doPlayCard` | Execute card play, check win |
| `doDraw` | Draw card(s) from deck |
| `doSkip` | Auto-skip or remove idle player |
| `doCallBluff` | Challenge +4 bluff |
| `sendMessage` | Send HTML message, return ID |
| `sendNextMessage` | Send message with "Suas cartas" button |
| `sendSticker` | Send card sticker |
| `reactMessage` | Add emoji reaction |

### Inline UI (results.go)
| Function | Purpose |
|----------|---------|
| `addCard` | Card sticker (playable or greyed-out) |
| `addDraw` / `addPass` / `addCallBluff` | Action buttons |
| `addChooseColor` | Color picker for wild cards |
| `addModeClassic/Fast/Wild/Text/Caseiro/Test` | Mode selection (for game starter) |
| `addNoGame` / `addNotStarted` | Status messages |
| `gameInfo` / `playerList` | Game state snapshot |
| `displayLink` / `displayName` | User formatting |

### Database (ranking.go)
| Function | Purpose |
|----------|---------|
| `RecordWin` | Record standard game win |
| `RecordChallengeWin/Loss` | Record challenge results |
| `GetRanking` | Query ranking by period (diario/semanal/mensal) |
| `GetChallengeRanking` | Head-to-head stats |
| `UpdateHeadToHead` | Update 1v1 record |

## External Dependencies

| Module                                   | Purpose                             |
|------------------------------------------|-------------------------------------|
| `github.com/mymmrac/telego v1.10.0`     | Telegram Bot API client             |
| `github.com/joho/godotenv v1.5.1`       | `.env` file loader                  |
| `github.com/lib/pq v1.12.3`             | PostgreSQL driver (blank import)    |

## Internal Dependencies (import graph)

```
main.go ───► commands.go ───► gamemanager.go ───► game.go ───► player.go ───► deck.go ───► card.go
                │                 │                   │
                │                 ├── match.go        └──► errors.go
                │                 └──► inline.go
                │                                    ranking.go ◄── config.go
                ▼
            actions.go ◄── inline.go ◄── results.go
```

## Environment Variables

| Variable            | Default                                  | Used By       |
|---------------------|------------------------------------------|---------------|
| `TOKEN`             | (required)                               | main.go       |
| `DATABASE_URL`      | `postgres://malbs@localhost:5432/...`     | config.go     |
| `WAITING_TIME`      | 120                                      | config.go     |
| `TIME_REMOVAL_AFTER_SKIP` | 20                                 | config.go     |
| `MIN_FAST_TURN_TIME` | 15                                      | config.go     |
| `MIN_PLAYERS`       | 2                                        | config.go     |
| `DEFAULT_GAMEMODE`  | `"fast"`                                 | config.go     |

## Diff from Previous

- **No previous codemap** — initial generation (100% new)
