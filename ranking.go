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

type UserGroupRank struct {
	ChatID    int64
	GroupName string
	Wins      int
	Rank      int
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
		`CREATE INDEX IF NOT EXISTS idx_wins_user ON wins (user_id)`,
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
		`CREATE TABLE IF NOT EXISTS group_settings (
			chat_id BIGINT PRIMARY KEY,
			default_mode TEXT NOT NULL DEFAULT 'fast',
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	}
	for _, q := range queries {
		if _, err := rs.db.Exec(q); err != nil {
			log.Printf("Erro ao executar query de init: %v", err)
		}
	}

	// Migrations para tabelas existentes
	rs.migrate()
}

func (rs *RankingStore) migrate() {
	// Adicionar coluna group_name se não existir
	var exists bool
	err := rs.db.QueryRow(
		`SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'wins' AND column_name = 'group_name'
		)`).Scan(&exists)
	if err == nil && !exists {
		_, err = rs.db.Exec(`ALTER TABLE wins ADD COLUMN group_name TEXT NOT NULL DEFAULT ''`)
		if err != nil {
			log.Printf("Erro ao adicionar coluna group_name: %v", err)
		} else {
			log.Println("Migration: coluna group_name adicionada à tabela wins")
		}
	}
}

func (rs *RankingStore) RecordWin(user *UserData, chatID int64, groupName string) {
	_, err := rs.db.Exec(
		`INSERT INTO wins (user_id, first_name, username, chat_id, group_name, created_at) VALUES ($1, $2, $3, $4, $5, NOW())`,
		user.ID, user.FirstName, user.Username, chatID, groupName,
	)
	if err != nil {
		log.Printf("Erro ao registrar vitória: %v", err)
	}
}

type HeadToHead struct {
	OpponentID      int64
	OpponentName    string
	OpponentUsername string
	MyWins          int
	MyLosses        int
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
			opponent_id,
			my_wins,
			my_losses,
			COALESCE(cw.first_name, '') as opponent_first_name,
			COALESCE(cw.username, '') as opponent_username
		FROM (
			SELECT
				CASE WHEN player1_id = $2 THEN player2_id ELSE player1_id END as opponent_id,
				CASE WHEN player1_id = $2 THEN p1_wins ELSE p2_wins END as my_wins,
				CASE WHEN player1_id = $2 THEN p2_wins ELSE p1_wins END as my_losses
			FROM challenge_headtohead
			WHERE chat_id = $1 AND (player1_id = $2 OR player2_id = $2)
		) sub
		LEFT JOIN challenge_wins cw ON cw.user_id = sub.opponent_id AND cw.chat_id = $1
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
		if err := rows.Scan(&h.OpponentID, &h.MyWins, &h.MyLosses, &h.OpponentName, &h.OpponentUsername); err != nil {
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

func (rs *RankingStore) GetGroupDefaultMode(chatID int64) string {
	var mode string
	err := rs.db.QueryRow(
		`SELECT default_mode FROM group_settings WHERE chat_id = $1`,
		chatID,
	).Scan(&mode)
	if err != nil {
		return GetDefaultGamemode()
	}
	return mode
}

func (rs *RankingStore) SetGroupDefaultMode(chatID int64, mode string) {
	_, err := rs.db.Exec(
		`INSERT INTO group_settings (chat_id, default_mode, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (chat_id)
		DO UPDATE SET default_mode = EXCLUDED.default_mode, updated_at = NOW()`,
		chatID, mode,
	)
	if err != nil {
		log.Printf("Erro ao salvar modo do grupo: %v", err)
	}
}

func (rs *RankingStore) GetUserRankingAcrossGroups(userID int64) []UserGroupRank {
	rows, err := rs.db.Query(
		`SELECT chat_id, group_name, COUNT(*) as wins
		FROM wins
		WHERE user_id = $1
		GROUP BY chat_id, group_name
		ORDER BY wins DESC`,
		userID,
	)
	if err != nil {
		log.Printf("Erro ao consultar ranking do usuário: %v", err)
		return nil
	}
	defer rows.Close()

	var result []UserGroupRank
	for rows.Next() {
		var r UserGroupRank
		if err := rows.Scan(&r.ChatID, &r.GroupName, &r.Wins); err != nil {
			log.Printf("Erro ao ler linha do ranking: %v", err)
			continue
		}
		result = append(result, r)
	}

	// Calcular posição de cada grupo
	for i := range result {
		var rank int
		err := rs.db.QueryRow(
			`SELECT COUNT(DISTINCT user_id) + 1
			FROM wins
			WHERE chat_id = $1
			AND user_id != $2
			AND (SELECT COUNT(*) FROM wins w2 WHERE w2.chat_id = $1 AND w2.user_id = wins.user_id) >= $3`,
			result[i].ChatID, userID, result[i].Wins,
		).Scan(&rank)
		if err != nil {
			rank = 1
		}
		result[i].Rank = rank
	}

	return result
}

func (rs *RankingStore) CleanGroupData(chatID int64) {
	var errs []error
	if _, err := rs.db.Exec("DELETE FROM wins WHERE chat_id = $1", chatID); err != nil {
		errs = append(errs, err)
	}
	if _, err := rs.db.Exec("DELETE FROM challenge_wins WHERE chat_id = $1", chatID); err != nil {
		errs = append(errs, err)
	}
	if _, err := rs.db.Exec("DELETE FROM challenge_headtohead WHERE chat_id = $1", chatID); err != nil {
		errs = append(errs, err)
	}
	if _, err := rs.db.Exec("DELETE FROM group_settings WHERE chat_id = $1", chatID); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		log.Printf("Erro ao limpar dados do grupo %d: %v", chatID, errs)
	} else {
		log.Printf("Dados do grupo %d limpos", chatID)
	}
}
