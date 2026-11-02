package main

import (
	"context"
	"slices"
	"sort"
)

// MemStore is an in-memory Store for tests and local runs without D1.
// It is not safe for concurrent use; callers serialize access.
type MemStore struct{ runs []Run }

func (f *MemStore) InsertRun(_ context.Context, r Run) error { f.runs = append(f.runs, r); return nil }
func (f *MemStore) ComboExists(_ context.Context, k string) (bool, error) {
	return slices.ContainsFunc(f.runs, func(r Run) bool { return r.ComboKey == k }), nil
}
func (f *MemStore) CountRuns(context.Context) (int, error) { return len(f.runs), nil }
func (f *MemStore) countIf(pred func(Run) bool) (int, error) {
	n := 0
	for _, r := range f.runs {
		if pred(r) {
			n++
		}
	}
	return n, nil
}
func (f *MemStore) CountHigher(_ context.Context, p int) (int, error) {
	return f.countIf(func(r Run) bool { return r.Points > p })
}
func (f *MemStore) CountWithTalent(_ context.Context, t string) (int, error) {
	return f.countIf(func(r Run) bool { return slices.Contains(r.Talents, t) })
}
func (f *MemStore) CountWithDifficulty(_ context.Context, d string) (int, error) {
	return f.countIf(func(r Run) bool { return r.Difficulty == d })
}
func (f *MemStore) CountWithFamily(_ context.Context, fam string) (int, error) {
	return f.countIf(func(r Run) bool { return r.Family == fam })
}
func (f *MemStore) CountRecentByIP(_ context.Context, ip string, since int64) (int, error) {
	return f.countIf(func(r Run) bool { return r.IPHash == ip && r.CreatedAt >= since })
}
func (f *MemStore) Ranking(_ context.Context, since int64, limit int) ([]RankEntry, error) {
	var rs []Run
	for _, r := range f.runs {
		if r.CreatedAt >= since {
			rs = append(rs, r)
		}
	}
	sort.SliceStable(rs, func(i, j int) bool { return rs[i].Points > rs[j].Points })
	var out []RankEntry
	for i, r := range rs {
		if i >= limit {
			break
		}
		out = append(out, RankEntry{Name: r.Name, Points: r.Points, Age: r.Age, Difficulty: r.Difficulty, Talents: r.Talents})
	}
	return out, nil
}
func (f *MemStore) Graves(_ context.Context, from, to float64, limit int) ([]Grave, error) {
	var out []Grave
	for _, r := range f.runs {
		if r.Age >= from && r.Age <= to {
			out = append(out, Grave{Name: r.Name, Age: r.Age, Cause: r.Cause})
		}
	}
	return out, nil
}
