-- Migration number: 0001 	 runs table for ranking, gacha stats, uniqueness and graves
CREATE TABLE IF NOT EXISTS runs (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  name         TEXT    NOT NULL,
  points       INTEGER NOT NULL,
  age          REAL    NOT NULL,
  achievements INTEGER NOT NULL,
  difficulty   TEXT    NOT NULL,
  family       TEXT    NOT NULL,
  talents      TEXT    NOT NULL, -- sorted talent IDs, comma separated
  combo_key    TEXT    NOT NULL, -- talents|difficulty|family
  look         TEXT    NOT NULL,
  cause        TEXT    NOT NULL,
  play_seconds REAL    NOT NULL,
  ip_hash      TEXT    NOT NULL,
  created_at   INTEGER NOT NULL  -- unix seconds
);
CREATE INDEX IF NOT EXISTS idx_runs_points  ON runs(points DESC);
CREATE INDEX IF NOT EXISTS idx_runs_created ON runs(created_at);
CREATE INDEX IF NOT EXISTS idx_runs_combo   ON runs(combo_key);
CREATE INDEX IF NOT EXISTS idx_runs_age     ON runs(age);
CREATE INDEX IF NOT EXISTS idx_runs_ip      ON runs(ip_hash, created_at);
