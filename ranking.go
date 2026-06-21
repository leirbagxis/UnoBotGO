package main

import (
	"database/sql"
	"log"
	"time"
)

type WinCount struct {
	UserID    int64
	FirstName string
	Username  string
	Wins      int
}

type RankingStore struct {
	db *sql.DB
}

var rankingStore *RankingStore

func NewRankingStore(db *sql.DB) *RankingStore {
	store := &RankingStore{db: db}
	store.init()
	return store
}

func (rs *RankingStore) init() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS wins (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			first_name TEXT NOT NULL,
			username TEXT NOT NULL DEFAULT '',
			chat_id BIGINT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_wins_chat_created ON wins (chat_id, created_at)`,
	}
	for _, q := range queries {
		if _, err := rs.db.Exec(q); err != nil {
			log.Printf("Erro ao executar query de init: %v", err)
		}
	}
}

func (rs *RankingStore) RecordWin(user *UserData, chatID int64) {
	_, err := rs.db.Exec(
		`INSERT INTO wins (user_id, first_name, username, chat_id, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		user.ID, user.FirstName, user.Username, chatID,
	)
	if err != nil {
		log.Printf("Erro ao registrar vitória: %v", err)
	}
}

func (rs *RankingStore) GetRanking(chatID int64, period string) []WinCount {
	var since time.Time
	now := time.Now()
	switch period {
	case "diario":
		since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "semanal":
		since = now.AddDate(0, 0, -7)
	default:
		since = now.AddDate(0, -1, 0)
	}

	rows, err := rs.db.Query(
		`SELECT user_id, first_name, username, COUNT(*) as wins
		FROM wins
		WHERE chat_id = $1 AND created_at >= $2
		GROUP BY user_id, first_name, username
		ORDER BY wins DESC`,
		chatID, since,
	)
	if err != nil {
		log.Printf("Erro ao consultar ranking: %v", err)
		return nil
	}
	defer rows.Close()

	var result []WinCount
	for rows.Next() {
		var wc WinCount
		if err := rows.Scan(&wc.UserID, &wc.FirstName, &wc.Username, &wc.Wins); err != nil {
			log.Printf("Erro ao ler linha do ranking: %v", err)
			continue
		}
		result = append(result, wc)
	}

	return result
}
