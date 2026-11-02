package main

import (
	"context"
	"database/sql"
	"slices"
	"strings"
)

// SQLStore implements Store on Cloudflare D1 (SQLite) via database/sql.
type SQLStore struct{ db *sql.DB }

func NewSQLStore(db *sql.DB) *SQLStore { return &SQLStore{db: db} }

func (s *SQLStore) count(ctx context.Context, q string, args ...any) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, q, args...).Scan(&n)
	return n, err
}

func (s *SQLStore) InsertRun(ctx context.Context, r Run) error {
	talents := slices.Clone(r.Talents)
	slices.Sort(talents)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO runs (name, points, age, achievements, difficulty, family, talents, combo_key, look, cause, play_seconds, ip_hash, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.Name, r.Points, r.Age, r.Achievements, r.Difficulty, r.Family,
		strings.Join(talents, ","), r.ComboKey, r.Look, r.Cause, r.PlaySeconds, r.IPHash, r.CreatedAt)
	return err
}

func (s *SQLStore) ComboExists(ctx context.Context, key string) (bool, error) {
	n, err := s.count(ctx, `SELECT COUNT(*) FROM runs WHERE combo_key = ?`, key)
	return n > 0, err
}

func (s *SQLStore) CountRuns(ctx context.Context) (int, error) {
	return s.count(ctx, `SELECT COUNT(*) FROM runs`)
}

func (s *SQLStore) CountHigher(ctx context.Context, points int) (int, error) {
	return s.count(ctx, `SELECT COUNT(*) FROM runs WHERE points > ?`, points)
}

func (s *SQLStore) CountWithTalent(ctx context.Context, talent string) (int, error) {
	return s.count(ctx, `SELECT COUNT(*) FROM runs WHERE ',' || talents || ',' LIKE ?`, "%,"+talent+",%")
}

func (s *SQLStore) CountWithDifficulty(ctx context.Context, d string) (int, error) {
	return s.count(ctx, `SELECT COUNT(*) FROM runs WHERE difficulty = ?`, d)
}

func (s *SQLStore) CountWithFamily(ctx context.Context, f string) (int, error) {
	return s.count(ctx, `SELECT COUNT(*) FROM runs WHERE family = ?`, f)
}

func (s *SQLStore) CountRecentByIP(ctx context.Context, ipHash string, since int64) (int, error) {
	return s.count(ctx, `SELECT COUNT(*) FROM runs WHERE ip_hash = ? AND created_at >= ?`, ipHash, since)
}

func (s *SQLStore) Ranking(ctx context.Context, since int64, limit int) ([]RankEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT name, points, age, difficulty, talents FROM runs
WHERE created_at >= ?
ORDER BY points DESC, created_at ASC
LIMIT ?`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RankEntry
	for rows.Next() {
		var e RankEntry
		var talents string
		if err := rows.Scan(&e.Name, &e.Points, &e.Age, &e.Difficulty, &talents); err != nil {
			return nil, err
		}
		e.Talents = splitTalents(talents)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *SQLStore) Graves(ctx context.Context, from, to float64, limit int) ([]Grave, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT name, age, cause FROM runs
WHERE age >= ? AND age <= ? AND age >= 1
ORDER BY created_at DESC
LIMIT ?`, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Grave
	for rows.Next() {
		var g Grave
		if err := rows.Scan(&g.Name, &g.Age, &g.Cause); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}
