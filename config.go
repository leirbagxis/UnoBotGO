package main

import (
	"os"
	"strconv"
	"sync"
)

var configOnce sync.Once

func loadConfig() {
	configOnce.Do(func() {
		_ = os.Getenv("TOKEN") // ensure initialization
	})
}

func GetToken() string {
	return os.Getenv("TOKEN")
}

func GetBotUsername() string {
	return os.Getenv("BOT_USERNAME")
}

func GetWaitingTime() int {
	return getEnvInt("WAITING_TIME", 120)
}

func GetTimeRemovalAfterSkip() int {
	return getEnvInt("TIME_REMOVAL_AFTER_SKIP", 20)
}

func GetMinFastTurnTime() int {
	return getEnvInt("MIN_FAST_TURN_TIME", 15)
}

func GetMinPlayers() int {
	return getEnvInt("MIN_PLAYERS", 2)
}

func GetDefaultGamemode() string {
	return getEnvStr("DEFAULT_GAMEMODE", "fast")
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
