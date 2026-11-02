package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

// RunRequest is the body of POST /api/runs. Keep in sync with game/api.go.
type RunRequest struct {
	Name         string   `json:"name"`
	Points       int      `json:"points"`
	Age          float64  `json:"age"`
	Achievements int      `json:"achievements"`
	Difficulty   string   `json:"difficulty"`
	Family       string   `json:"family"`
	Talents      []string `json:"talents"`
	Look         string   `json:"look"`
	Cause        string   `json:"cause"`
	PlaySeconds  float64  `json:"playSeconds"`
}

type RunResult struct {
	Rank           int                `json:"rank"`
	Total          int                `json:"total"`
	Unique         bool               `json:"unique"`
	TalentRates    map[string]float64 `json:"talentRates"`
	DifficultyRate float64            `json:"difficultyRate"`
	FamilyRate     float64            `json:"familyRate"`
	Name           string             `json:"name"`
}

type Run struct {
	RunRequest
	ComboKey  string
	IPHash    string
	CreatedAt int64
}

type RankEntry struct {
	Name       string   `json:"name"`
	Points     int      `json:"points"`
	Age        float64  `json:"age"`
	Difficulty string   `json:"difficulty"`
	Talents    []string `json:"talents"`
}

type Grave struct {
	Name  string  `json:"name"`
	Age   float64 `json:"age"`
	Cause string  `json:"cause"`
}

// Store abstracts the database so handlers can be tested without D1.
type Store interface {
	InsertRun(ctx context.Context, r Run) error
	ComboExists(ctx context.Context, key string) (bool, error)
	CountRuns(ctx context.Context) (int, error)
	CountHigher(ctx context.Context, points int) (int, error)
	CountWithTalent(ctx context.Context, talent string) (int, error)
	CountWithDifficulty(ctx context.Context, d string) (int, error)
	CountWithFamily(ctx context.Context, f string) (int, error)
	CountRecentByIP(ctx context.Context, ipHash string, since int64) (int, error)
	Ranking(ctx context.Context, since int64, limit int) ([]RankEntry, error)
	Graves(ctx context.Context, from, to float64, limit int) ([]Grave, error)
}

type API struct {
	store Store
	now   func() time.Time
}

const (
	rateLimitPerMinute = 5
	rankingLimit       = 20
	gravesLimit        = 60
	ipSalt             = "jinsei-game"
)

var jst = time.FixedZone("JST", 9*60*60)

func NewAPI(s Store) *API { return &API{store: s, now: time.Now} }

func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/runs", a.postRun)
	mux.HandleFunc("GET /api/ranking", a.getRanking)
	mux.HandleFunc("GET /api/graves", a.getGraves)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Println("encode:", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func hashIP(ip string) string {
	sum := sha256.Sum256([]byte(ipSalt + ip))
	return hex.EncodeToString(sum[:8])
}

func rate(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(int(float64(n)/float64(total)*1000+0.5)) / 10
}

func (a *API) postRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req RunRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := validateRun(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	now := a.now()
	ip := r.Header.Get("CF-Connecting-IP")
	if ip == "" {
		ip, _, _ = strings.Cut(r.RemoteAddr, ":")
	}
	ipHash := hashIP(ip)
	recent, err := a.store.CountRecentByIP(ctx, ipHash, now.Add(-time.Minute).Unix())
	if err != nil {
		a.internal(w, err)
		return
	}
	if recent >= rateLimitPerMinute {
		writeError(w, http.StatusTooManyRequests, "too many requests")
		return
	}

	key := comboKey(req.Talents, req.Difficulty, req.Family)
	exists, err := a.store.ComboExists(ctx, key)
	if err != nil {
		a.internal(w, err)
		return
	}
	run := Run{RunRequest: req, ComboKey: key, IPHash: ipHash, CreatedAt: now.Unix()}
	if err := a.store.InsertRun(ctx, run); err != nil {
		a.internal(w, err)
		return
	}

	total, err := a.store.CountRuns(ctx)
	if err != nil {
		a.internal(w, err)
		return
	}
	higher, err := a.store.CountHigher(ctx, req.Points)
	if err != nil {
		a.internal(w, err)
		return
	}
	res := RunResult{
		Rank:        higher + 1,
		Total:       total,
		Unique:      !exists,
		TalentRates: map[string]float64{},
		Name:        req.Name,
	}
	for _, t := range req.Talents {
		n, err := a.store.CountWithTalent(ctx, t)
		if err != nil {
			a.internal(w, err)
			return
		}
		res.TalentRates[t] = rate(n, total)
	}
	nd, err := a.store.CountWithDifficulty(ctx, req.Difficulty)
	if err != nil {
		a.internal(w, err)
		return
	}
	nf, err := a.store.CountWithFamily(ctx, req.Family)
	if err != nil {
		a.internal(w, err)
		return
	}
	res.DifficultyRate = rate(nd, total)
	res.FamilyRate = rate(nf, total)
	writeJSON(w, res)
}

func (a *API) internal(w http.ResponseWriter, err error) {
	log.Println("internal error:", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

func (a *API) getRanking(w http.ResponseWriter, r *http.Request) {
	var since int64
	switch r.URL.Query().Get("period") {
	case "", "all":
	case "today":
		t := a.now().In(jst)
		since = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, jst).Unix()
	default:
		writeError(w, http.StatusBadRequest, "unknown period")
		return
	}
	entries, err := a.store.Ranking(r.Context(), since, rankingLimit)
	if err != nil {
		a.internal(w, err)
		return
	}
	if entries == nil {
		entries = []RankEntry{}
	}
	writeJSON(w, map[string]any{"entries": entries})
}

func (a *API) getGraves(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, err1 := strconv.ParseFloat(q.Get("from"), 64)
	to, err2 := strconv.ParseFloat(q.Get("to"), 64)
	if err1 != nil || err2 != nil || from < 0 || to < from || to-from > 2000 {
		writeError(w, http.StatusBadRequest, "invalid range")
		return
	}
	graves, err := a.store.Graves(r.Context(), from, to, gravesLimit)
	if err != nil {
		a.internal(w, err)
		return
	}
	if graves == nil {
		graves = []Grave{}
	}
	writeJSON(w, map[string]any{"graves": graves})
}

// splitTalents parses the stored comma-separated talents.
func splitTalents(s string) []string {
	if s == "" {
		return nil
	}
	ts := strings.Split(s, ",")
	slices.Sort(ts)
	return ts
}
