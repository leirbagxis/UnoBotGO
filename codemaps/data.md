# Data Models & Schemas

**Generated:** 2026-06-25T15:26:00-03:00

## In-Memory Data Models

### Card (`card.go`)

```go
type Card struct {
    Color   string   // "r","b","g","y","x"
    Value   string   // "0"-"9","draw","reverse","skip"
    Special string   // "colorchooser","draw_four",""
}
```

- **Colors:** Red(`r`), Blue(`b`), Green(`g`), Yellow(`y`), Black(`x`)
- **Values:** 0-9, DrawTwo, Reverse, Skip
- **Specials:** Choose (colorchooser), DrawFour (draw_four)
- 108 cards in classic deck (Wild mode: 80 cards)

### Deck (`deck.go`)

```go
type Deck struct {
    Cards     []*Card   // draw pile
    Graveyard []*Card   // played cards
}
```

### Player (`player.go`)

```go
type Player struct {
    User        *UserData
    Cards       []*Card
    Game        *Game
    Next        *Player     // circular linked list
    Prev        *Player
    Drew        bool
    Bluffing    bool
    TurnStarted time.Time
    WaitingTime int
}
```

### UserData (`player.go`)

```go
type UserData struct {
    ID        int64
    FirstName string
    Username  string
}
```

### Game (`game.go`)

```go
type Game struct {
    mu            sync.Mutex
    ChatID        int64
    Deck          *Deck
    LastCard      *Card
    CurrentPlayer *Player
    Reversed      bool
    ChoosingColor bool
    Started       bool
    DrawCounter   int
    PlayersWon    int
    Starter       *UserData
    Owner         []int64
    Mode          string     // "classic","fast","wild","text","caseiro","test"
    Open          bool
    MatchID       int64      // 0 = normal game, >0 = challenge game
}
```

### GameManager (`gamemanager.go`)

```go
type GameManager struct {
    mu            sync.Mutex
    ChatIDGames   map[int64][]*Game          // chat → game stack
    UserIDPlayers map[int64][]*Player        // user → active games
    UserIDCurrent map[int64]*Player          // user → current game
    RemindDict    map[int64]map[int64]bool   // chat → user → notify
    ChatIDMatch   map[int64]*Match           // chat → active challenge
    nextMatchID   int64
}
```

### Match (`match.go`)

```go
type MatchState int

const (
    MatchWaiting      MatchState = iota  // awaiting opponent
    MatchPlaying                          // game in progress
    MatchBetweenGames                     // between rounds
    MatchFinished                         // complete
)

type Match struct {
    ID            int64
    Challenger    *UserData
    Challenged    *UserData
    ChatID        int64
    BestOf        int          // 1, 3, or 5
    TargetWins    int          // 1, 2, or 3
    Mode          string
    Wins1         int          // challenger wins
    Wins2         int          // challenged wins
    State         MatchState
    MessageID     int          // Telegram message for editing
    CurrentGame   *Game
    winner        *UserData    // unexported
    configMessage bool         // unexported
}
```

### Ranking / Persistence (`ranking.go`)

```go
type WinCount struct {
    UserID    int64
    FirstName string
    Username  string
    Wins      int
}

type RankingStore struct {
    db *sql.DB
}

type HeadToHead struct {
    OpponentID       int64
    OpponentName     string
    OpponentUsername string
    MyWins           int
    MyLosses         int
}
```

## Database Schema (PostgreSQL)

Managed via `ranking.go` — tables are created in `RankingStore.init()`:

### Table: `wins`

```sql
CREATE TABLE IF NOT EXISTS wins (
    id          SERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL,
    first_name  TEXT NOT NULL DEFAULT '',
    username    TEXT NOT NULL DEFAULT '',
    chat_id     BIGINT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Purpose:** Records every game win. Used for period-based rankings.

| Column      | Type         | Notes                    |
|-------------|--------------|--------------------------|
| id          | SERIAL       | Primary key              |
| user_id     | BIGINT       | Telegram user ID         |
| first_name  | TEXT         | Snapshot at win time     |
| username    | TEXT         | Snapshot at win time     |
| chat_id     | BIGINT       | Group ID                 |
| created_at  | TIMESTAMPTZ  | Used for period filters  |

### Table: `challenge_wins`

```sql
CREATE TABLE IF NOT EXISTS challenge_wins (
    id              SERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL,
    opponent_id     BIGINT NOT NULL,
    opponent_name   TEXT NOT NULL DEFAULT '',
    opponent_username TEXT NOT NULL DEFAULT '',
    chat_id         BIGINT NOT NULL,
    won             BOOLEAN NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Purpose:** Records challenge match results for head-to-head tracking.

| Column           | Type         | Notes                    |
|------------------|--------------|--------------------------|
| id               | SERIAL       | Primary key              |
| user_id          | BIGINT       | Winner or loser          |
| opponent_id      | BIGINT       | Opponent ID              |
| opponent_name    | TEXT         | Snapshot                 |
| opponent_username| TEXT         | Snapshot                 |
| chat_id          | BIGINT       | Group ID                 |
| won              | BOOLEAN      | true=win, false=loss     |
| created_at       | TIMESTAMPTZ  |                          |

## Key Relationships

```
Chat ──1:N──> Game
Game ──1:N──> Player (circular linked list)
Game ──1:1──> Deck
Deck ──1:N──> Card
Player ──1:N──> Card (hand)

Chat ──0:1──> Match
Match ──1:1──> Challenger (UserData)
Match ──1:1──> Challenged (UserData)
Match ──0:1──> CurrentGame (Game)

Chat ──1:N──> Win (ranking)
User ──1:N──> ChallengeWin (head-to-head)
```

## Global State

```go
var gm            *GameManager    // defined in inline.go
var botCtx        = context.Background()
var botToken      string          // defined in main.go
var botUsername   string          // defined in main.go
var rankingStore  *RankingStore   // defined in ranking.go
```

All state is in-memory (maps in `GameManager`). Only rankings persist to PostgreSQL.

## Diff from Previous

- **No previous codemap** — initial generation (100% new)
