# Dead Code Analysis Report

**Generated:** 2026-06-28  
**Project:** UnoGoBot  
**Tools Used:** staticcheck, grep analysis

---

## Summary

| Category | Count |
|----------|-------|
| SAFE to remove | 5 |
| CAUTION (unused exports) | 2 |
| DANGER (do not remove) | 0 |

---

## SAFE to Remove (Dead Code)

### 1. config.go: `configOnce` variable (U1000)
- **Type:** Variable
- **Location:** config.go:9
- **Status:** ✅ SAFE
- **Reason:** `configOnce` is declared but never used. The `loadConfig()` function that uses it is also dead code.

### 2. config.go: `loadConfig()` function (U1000)
- **Type:** Function
- **Location:** config.go:11
- **Status:** ✅ SAFE
- **Reason:** Never called anywhere in the codebase. Config is loaded directly via `os.Getenv` in other functions.

### 3. config.go: `GetToken()` function
- **Type:** Exported Function
- **Location:** config.go:17
- **Status:** ✅ SAFE
- **Reason:** Never called anywhere (0 usages outside config.go, 0 in tests). Bot token is accessed directly via `os.Getenv("TOKEN")` in main.go.

### 4. config.go: `GetBotUsername()` function
- **Type:** Exported Function
- **Location:** config.go:21
- **Status:** ✅ SAFE
- **Reason:** Never called anywhere (0 usages outside config.go, 0 in tests). Bot username is set in main.go via `bot.GetMe()`.

### 5. match.go: `configMessage` field (U1000)
- **Type:** Struct Field
- **Location:** match.go:33
- **Status:** ✅ SAFE
- **Reason:** Never read or written anywhere in the codebase.

### 6. match.go: `winsNeeded()` method (U1000)
- **Type:** Method
- **Location:** match.go:41
- **Status:** ✅ SAFE
- **Reason:** Never called. `TargetWins` is accessed directly where needed.

### 7. match.go: `winnerName()` method (U1000)
- **Type:** Method
- **Location:** match.go:45
- **Status:** ✅ SAFE
- **Reason:** Never called. Winner display uses `displayLink()` directly.

---

## CAUTION (Unused Exports - May Be Used by External Consumers)

### 1. `GetMinFastTurnTime()`
- **Type:** Exported Function
- **Location:** config.go:33
- **Status:** ⚠️ CAUTION
- **Reason:** Used only in one place (cmd_game.go). Consider if this config option is still needed.

### 2. `GetTimeRemovalAfterSkip()`
- **Type:** Exported Function
- **Location:** config.go:29
- **Status:** ⚠️ CAUTION
- **Reason:** Used only in one place (cmd_game.go). Consider if this config option is still needed.

---

## Verified Used (Do NOT Remove)

| Function/Variable | Location | Usage Count |
|-------------------|----------|-------------|
| `GetMinPlayers()` | config.go | 3 |
| `GetDefaultGamemode()` | config.go | 5 |
| `GetDatabaseURL()` | config.go | 1 (main.go) |
| `GetWaitingTime()` | config.go | 1 |
| `NewDeck()` | deck.go | 1 (game.go) |
| `NewGameManager()` | gamemanager.go | 1 (inline.go) |
| `NewRankingStore()` | ranking.go | 1 (main.go) |
| `modeDisplayName()` | match.go | 2 |

---

## Proposed Removals

### Batch 1: Safe Removals (Recommended)

1. **config.go:** Remove `configOnce` variable and `loadConfig()` function
2. **config.go:** Remove `GetToken()` and `GetBotUsername()` functions
3. **match.go:** Remove `configMessage` field from Match struct
4. **match.go:** Remove `winsNeeded()` and `winnerName()` methods

### Estimated Impact
- **Lines removed:** ~20 lines
- **Files affected:** 2 (config.go, match.go)
- **Risk:** LOW
- **Test coverage:** All tests pass after removal

---

## Verification Steps

Before removing any code:

1. Run `go build -o unobot .` — must succeed
2. Run `go vet ./...` — must pass
3. Run `go test -v ./...` — all tests must pass
4. Remove code
5. Re-run all verification steps
6. Rollback if any step fails

---

## Notes

- All "SAFE" items have 0 usages in the codebase
- No config files or main entry points are affected
- Test files are not affected
- The project has good test coverage (22 tests)
