package main

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// RunRequest is sent to POST /api/runs. Keep in sync with worker/api.go.
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
	Rank       int                `json:"rank"`
	Total      int                `json:"total"`
	Unique     bool               `json:"unique"`
	Talents    map[string]float64 `json:"talentRates"`
	Difficulty float64            `json:"difficultyRate"`
	Family     float64            `json:"familyRate"`
	Name       string             `json:"name"`
}

type RankEntry struct {
	Name       string   `json:"name"`
	Points     int      `json:"points"`
	Age        float64  `json:"age"`
	Difficulty string   `json:"difficulty"`
	Talents    []string `json:"talents"`
}

type GraveDTO struct {
	Name  string  `json:"name"`
	Age   float64 `json:"age"`
	Cause string  `json:"cause"`
}

// Async holds the result of a background request.
type Async[T any] struct {
	ch    chan struct{}
	Value T
	Err   error
}

func goAsync[T any](f func() (T, error)) *Async[T] {
	a := &Async[T]{ch: make(chan struct{})}
	go func() {
		a.Value, a.Err = f()
		close(a.ch)
	}()
	return a
}

// Done reports whether the request finished (non-blocking).
func (a *Async[T]) Done() bool {
	if a == nil {
		return false
	}
	select {
	case <-a.ch:
		return true
	default:
		return false
	}
}

func apiURL(path string) (string, error) {
	base := apiBase()
	if base == "" || base == "null" {
		return "", fmt.Errorf("オフラインモード")
	}
	return base + path, nil
}

func getJSON(path string, out any) error {
	u, err := apiURL(path)
	if err != nil {
		return err
	}
	b, err := httpDo("GET", u, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func SubmitRun(req RunRequest) *Async[RunResult] {
	return goAsync(func() (RunResult, error) {
		var res RunResult
		u, err := apiURL("/api/runs")
		if err != nil {
			return res, err
		}
		body, _ := json.Marshal(req)
		b, err := httpDo("POST", u, body)
		if err != nil {
			return res, err
		}
		err = json.Unmarshal(b, &res)
		return res, err
	})
}

func FetchRanking(period string) *Async[[]RankEntry] {
	return goAsync(func() ([]RankEntry, error) {
		var out struct {
			Entries []RankEntry `json:"entries"`
		}
		err := getJSON("/api/ranking?period="+url.QueryEscape(period), &out)
		return out.Entries, err
	})
}

func FetchGraves(from, to float64) *Async[[]GraveDTO] {
	return goAsync(func() ([]GraveDTO, error) {
		var out struct {
			Graves []GraveDTO `json:"graves"`
		}
		err := getJSON(fmt.Sprintf("/api/graves?from=%.0f&to=%.0f", from, to), &out)
		return out.Graves, err
	})
}
