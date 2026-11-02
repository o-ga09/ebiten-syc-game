package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestServer() (*http.ServeMux, *MemStore, *API) {
	st := &MemStore{}
	api := NewAPI(st)
	api.now = func() time.Time { return time.Date(2026, 11, 2, 12, 0, 0, 0, jst) }
	mux := http.NewServeMux()
	api.Register(mux)
	return mux, st, api
}

func validReq() RunRequest {
	return RunRequest{
		Name: "ちくわぶ", Points: 4800, Age: 40, Achievements: 1,
		Difficulty: "hard", Family: "rich", Talents: []string{"magnet", "pushup", "double_jump"},
		Look: "h1-c2-x0", Cause: "上司にやられた", PlaySeconds: 90,
	}
}

func post(t *testing.T, mux http.Handler, req RunRequest, ip string) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/api/runs", bytes.NewReader(b))
	r.Header.Set("CF-Connecting-IP", ip)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func TestPostRunUniqueThenNot(t *testing.T) {
	mux, _, _ := newTestServer()
	w := post(t, mux, validReq(), "1.1.1.1")
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body)
	}
	var res RunResult
	json.Unmarshal(w.Body.Bytes(), &res)
	if !res.Unique || res.Rank != 1 || res.Total != 1 || res.TalentRates["magnet"] != 100 {
		t.Fatalf("first: %+v", res)
	}

	r2 := validReq()
	r2.Points = 9000
	r2.Talents = []string{"double_jump", "magnet", "pushup"} // same combo, other order
	w = post(t, mux, r2, "2.2.2.2")
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Unique || res.Rank != 1 || res.Total != 2 {
		t.Fatalf("second: %+v", res)
	}
}

func TestPostRunRejectsInvalid(t *testing.T) {
	mux, _, _ := newTestServer()
	cases := map[string]func(*RunRequest){
		"empty name":     func(r *RunRequest) { r.Name = " " },
		"long name":      func(r *RunRequest) { r.Name = "あいうえおかきくけこさ" },
		"symbol name":    func(r *RunRequest) { r.Name = "<script>" },
		"bad difficulty": func(r *RunRequest) { r.Difficulty = "god" },
		"dup talents":    func(r *RunRequest) { r.Talents = []string{"magnet", "magnet", "pushup"} },
		"two talents":    func(r *RunRequest) { r.Talents = []string{"magnet", "pushup"} },
		"too fast":       func(r *RunRequest) { r.PlaySeconds = 2 },
		"huge points":    func(r *RunRequest) { r.Points = 99999999 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := validReq()
			mutate(&r)
			if w := post(t, mux, r, "3.3.3.3"); w.Code != http.StatusBadRequest {
				t.Fatalf("status=%d", w.Code)
			}
		})
	}
}

func TestNGNameIsReplaced(t *testing.T) {
	mux, st, _ := newTestServer()
	r := validReq()
	r.Name = "ＦＵＣＫ"
	if w := post(t, mux, r, "4.4.4.4"); w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	if st.runs[0].Name != anonymous {
		t.Fatalf("name=%q", st.runs[0].Name)
	}
}

func TestRateLimit(t *testing.T) {
	mux, _, _ := newTestServer()
	for i := range rateLimitPerMinute {
		if w := post(t, mux, validReq(), "5.5.5.5"); w.Code != http.StatusOK {
			t.Fatalf("req %d status=%d", i, w.Code)
		}
	}
	if w := post(t, mux, validReq(), "5.5.5.5"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestRankingAndGraves(t *testing.T) {
	mux, _, _ := newTestServer()
	post(t, mux, validReq(), "6.6.6.6")
	for _, path := range []string{"/api/ranking?period=all", "/api/ranking?period=today", "/api/graves?from=0&to=100"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "ちくわぶ") {
			t.Fatalf("%s: status=%d body=%s", path, w.Code, w.Body)
		}
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/graves?from=10&to=1", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
}
