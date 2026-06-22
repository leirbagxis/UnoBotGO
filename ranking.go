package main

import (
	"database/sql"
	"log"
	"sync"
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

var (
	rankingStore *RankingStore
	brOnce       sync.Once
	brLoc        *time.Location
)

func brLocation() *time.Location {
	brOnce.Do(func() {
		loc, err := time.LoadLocation("America/Sao_Paulo")
		if err != nil {
			log.Printf("Erro ao carregar America/Sao_Paulo, usando UTC-3 fixo: %v", err)
			loc = time.FixedZone("BRT", -3*60*60)
		}
		brLoc = loc
	})
	return brLoc
}

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
		`CREATE TABLE IF NOT EXISTS challenge_wins (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			first_name TEXT NOT NULL,
			username TEXT NOT NULL DEFAULT '',
			chat_id BIGINT NOT NULL,
			wins INT NOT NULL DEFAULT 0,
			losses INT NOT NULL DEFAULT 0,
			UNIQUE(user_id, chat_id)
		)`,
		`CREATE TABLE IF NOT EXISTS challenge_headtohead (
			id BIGSERIAL PRIMARY KEY,
			chat_id BIGINT NOT NULL,
			player1_id BIGINT NOT NULL,
			player2_id BIGINT NOT NULL,
			p1_wins INT NOT NULL DEFAULT 0,
			p2_wins INT NOT NULL DEFAULT 0,
			UNIQUE(chat_id, player1_id, player2_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_headtohead_chat ON challenge_headtohead (chat_id)`,
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

type HeadToHead struct {
	OpponentID  int64
	OpponentName string
	MyWins      int
	MyLosses    int
}

func (rs *RankingStore) RecordChallengeWin(user *UserData, chatID int64) {
	_, err := rs.db.Exec(
		`INSERT INTO challenge_wins (user_id, first_name, username, chat_id, wins, losses)
		VALUES ($1, $2, $3, $4, 1, 0)
		ON CONFLICT (user_id, chat_id)
		DO UPDATE SET wins = challenge_wins.wins + 1,
			first_name = EXCLUDED.first_name,
			username = EXCLUDED.username`,
		user.ID, user.FirstName, user.Username, chatID,
	)
	if err != nil {
		log.Printf("Erro ao registrar vitória em desafio: %v", err)
	}
}

func (rs *RankingStore) RecordChallengeLoss(user *UserData, chatID int64) {
	_, err := rs.db.Exec(
		`INSERT INTO challenge_wins (user_id, first_name, username, chat_id, wins, losses)
		VALUES ($1, $2, $3, $4, 0, 1)
		ON CONFLICT (user_id, chat_id)
		DO UPDATE SET losses = challenge_wins.losses + 1,
			first_name = EXCLUDED.first_name,
			username = EXCLUDED.username`,
		user.ID, user.FirstName, user.Username, chatID,
	)
	if err != nil {
		log.Printf("Erro ao registrar derrota em desafio: %v", err)
	}
}

func (rs *RankingStore) GetChallengeRanking(chatID int64, userID int64) []HeadToHead {
	rows, err := rs.db.Query(
		`SELECT
			CASE WHEN player1_id = $2 THEN player2_id ELSE player1_id END as opponent_id,
			CASE WHEN player1_id = $2 THEN p1_wins ELSE p2_wins END as my_wins,
			CASE WHEN player1_id = $2 THEN p2_wins ELSE p1_wins END as my_losses
		FROM challenge_headtohead
		WHERE chat_id = $1 AND (player1_id = $2 OR player2_id = $2)
		ORDER BY my_wins DESC`,
		chatID, userID,
	)
	if err != nil {
		log.Printf("Erro ao consultar ranking de desafios: %v", err)
		return nil
	}
	defer rows.Close()

	var result []HeadToHead
	for rows.Next() {
		var h HeadToHead
		if err := rows.Scan(&h.OpponentID, &h.MyWins, &h.MyLosses); err != nil {
			log.Printf("Erro ao ler linha do ranking de desafios: %v", err)
			continue
		}
		result = append(result, h)
	}

	return result
}

func (rs *RankingStore) UpdateHeadToHead(chatID int64, winner, loser *UserData, winnerWins, loserWins int) {
	p1, p2 := winner, loser
	if p1.ID > p2.ID {
		p1, p2 = p2, p1
	}

	w1, w2 := winnerWins, loserWins
	if p1.ID != winner.ID {
		w1, w2 = loserWins, winnerWins
	}

	_, err := rs.db.Exec(
		`INSERT INTO challenge_headtohead (chat_id, player1_id, player2_id, p1_wins, p2_wins)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (chat_id, player1_id, player2_id)
		DO UPDATE SET p1_wins = challenge_headtohead.p1_wins + $4,
			p2_wins = challenge_headtohead.p2_wins + $5`,
		chatID, p1.ID, p2.ID, w1, w2,
	)
	if err != nil {
		log.Printf("Erro ao atualizar head-to-head: %v", err)
	}
}

func (rs *RankingStore) GetRanking(chatID int64, period string) []WinCount {
	loc := brLocation()
	now := time.Now().In(loc)

	var since time.Time
	switch period {
	case "diario":
		since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
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
