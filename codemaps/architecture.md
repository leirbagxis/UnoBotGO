# Architecture

**Generated:** 2026-06-25T15:26:00-03:00
**Project:** UnoGoBot — Telegram UNO Bot (Go)
**Module:** `github.com/malbs/UnoGoBot`
**Go Version:** 1.26.3

## Overview

Monolithic Go application (single `main` package) implementing a UNO card game bot for Telegram. All game logic, bot interaction, and persistence live in flat package structure with no sub-packages.

## System Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Telegram API                       │
└──────────┬────────────────────────┬──────────────────┘
           │ updates (long polling) │ inline queries
           ▼                        ▼
┌──────────────────────┐  ┌────────────────────┐
│  handleMessage()     │  │ handleInlineQuery()│
│  (commands.go)       │  │ (inline.go)        │
└──────┬───────────────┘  └─────────┬──────────┘
       │                            │
       ▼                            ▼
┌─────────────────────────────────────────────────────┐
│                   GameManager                        │
│            (gamemanager.go — global `gm`)            │
│                                                      │
│   Manages: Games, Players, Matches, Reminders        │
│   Thread-safe: sync.Mutex on all public ops          │
└───┬──────────┬──────────┬──────────┬─────────────────┘
    │          │          │          │
    ▼          ▼          ▼          ▼
┌──────┐ ┌────────┐ ┌────────┐ ┌──────────┐
│ Game │ │ Player │ │ Match  │ │ Deck/Card│
│.go   │ │ .go    │ │ .go    │ │ .go      │
└──────┘ └────────┘ └────────┘ └──────────┘

┌──────────────────────────────────────────┐
│              actions.go                   │
│   doPlayCard / doDraw / doSkip / send*    │
└──────────────────────────────────────────┘

┌──────────────────────────────────────────┐
│              results.go                   │
│   inline query result builders           │
└──────────────────────────────────────────┘

┌──────────────────────────────────────────┐
│              ranking.go                   │
│   PostgreSQL persistence (RankingStore)   │
└──────────────────────────────────────────┘

┌──────────────────────────────────────────┐
│              config.go                    │
│   Env-based configuration                │
└──────────────────────────────────────────┘
```

## Request Flow

1. **Long polling** receives updates from Telegram
2. **handleMessage()** dispatches by command (`/novo`, `/entrar`, `/iniciar`, ...)
3. **handleInlineQuery()** builds inline button results for card UI
4. **handleChosenInlineResult()** processes card plays, draws, mode changes
5. All mutation goes through **GameManager** (global `gm`) under mutex lock
6. Responses sent via Telegram API (`sendMessage`, `sendNextMessage`, `sendSticker`)

## Data Flow

```
User ──► Telegram ──► Long Poll ──► handleMessage()
                                       │
                                       ▼
                                  GameManager
                                       │
                                       ▼
                                  Game/Player/Match
                                       │
                                       ▼
                                  sendMessage() ──► Telegram ──► User
```

## Key Design Decisions

- **Single package** — all code in `package main`; no sub-packages
- **Global state** — `GameManager` (`gm`), `rankingStore`, `botCtx` are package-level globals
- **Inline UI** — Players interact via Telegram inline queries (type `@bot` + tap), not chat commands
- **Thread safety** — `sync.Mutex` on `GameManager` and `Game`; no channels
- **3-runner modes** — local, docker-db+local-app, full-docker (see Makefile)

## Infrastructure

| Component | Technology |
|-----------|-----------|
| Runtime   | Go 1.26.3 |
| Bot Lib   | telego v1.10.0 |
| Database  | PostgreSQL 17 |
| Deploy    | Docker / Docker Compose |
| Config    | `.env` file + env vars |

## Diff from Previous

- **No previous codemap** — initial generation (100% new)
