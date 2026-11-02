package main

import (
	"errors"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The ID lists must match game/data.go.
var (
	validTalents = []string{
		"double_jump", "tough", "magnet", "early_bird", "bug_proof", "foresight", "lucky",
		"pushup", "loud", "lost", "oversleep", "dodge", "scholar", "thrifty", "gambler",
		"tall", "cat_lover", "sprinter", "phoenix", "debugger",
	}
	validDifficulties = []string{"easy", "normal", "hard", "very_hard", "lunatic"}
	validFamilies     = []string{"rich", "free", "protective", "mover", "normal", "big"}
)

const (
	maxNameLen    = 10
	anonymous     = "名無しの人生"
	maxAge        = 5000
	maxCauseLen   = 40
	maxTapsPerSec = 15.0
	rowsPerYear   = 5
)

// ngWords is a small deny list; names containing them are replaced.
var ngWords = []string{
	"死ね", "しね", "殺す", "ころす", "ちんこ", "ちんぽ", "まんこ", "うんこ", "セックス", "せっくす",
	"fuck", "shit", "sex", "nigger", "bitch", "kill",
}

var allowedSymbols = "ー・！？!?_-.☆★♪～~ 　"

// sanitizeName validates a player name. It returns the name to store.
func sanitizeName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(name)
	if n == 0 {
		return "", errors.New("name is empty")
	}
	if n > maxNameLen {
		return "", errors.New("name is too long")
	}
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || strings.ContainsRune(allowedSymbols, r) {
			continue
		}
		return "", errors.New("name contains invalid characters")
	}
	if containsNG(name) {
		return anonymous, nil
	}
	return name, nil
}

func containsNG(s string) bool {
	norm := strings.ToLower(strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '・' || r == '_' || r == '-' || r == '.' {
			return -1
		}
		// Fold full-width ASCII to half-width.
		if r >= '！' && r <= '～' {
			return r - '！' + '!'
		}
		return r
	}, s))
	for _, w := range ngWords {
		if strings.Contains(norm, w) {
			return true
		}
	}
	return false
}

// validateRun checks ranges and normalizes the request in place.
func validateRun(req *RunRequest) error {
	name, err := sanitizeName(req.Name)
	if err != nil {
		return err
	}
	req.Name = name

	if !slices.Contains(validDifficulties, req.Difficulty) {
		return errors.New("unknown difficulty")
	}
	if !slices.Contains(validFamilies, req.Family) {
		return errors.New("unknown family")
	}
	if len(req.Talents) != 3 {
		return errors.New("talents must be 3")
	}
	seen := map[string]bool{}
	for _, t := range req.Talents {
		if !slices.Contains(validTalents, t) || seen[t] {
			return errors.New("invalid talents")
		}
		seen[t] = true
	}
	if req.Age < 0 || req.Age > maxAge {
		return errors.New("age out of range")
	}
	if req.Achievements < 0 || float64(req.Achievements) > req.Age*rowsPerYear+1 {
		return errors.New("achievements out of range")
	}
	// Climbing needs at least one tap per row.
	if req.PlaySeconds < req.Age*rowsPerYear/maxTapsPerSec-1 {
		return errors.New("too fast")
	}
	// Upper bound: max multiplier 2.0*1.2*1.05, achievement 600, rare <= 10000 each.
	maxRare := 10000.0 * (req.Age*rowsPerYear*0.02 + 1)
	maxPoints := (req.Age*100 + float64(req.Achievements)*600 + maxRare) * 2.6
	if req.Points < 0 || float64(req.Points) > maxPoints {
		return errors.New("points out of range")
	}
	if utf8.RuneCountInString(req.Cause) > maxCauseLen {
		req.Cause = string([]rune(req.Cause)[:maxCauseLen])
	}
	if len(req.Look) > 32 {
		req.Look = req.Look[:32]
	}
	return nil
}

// comboKey must match Birth.ComboKey in the game.
func comboKey(talents []string, difficulty, family string) string {
	ids := slices.Clone(talents)
	slices.Sort(ids)
	return strings.Join(ids, ",") + "|" + difficulty + "|" + family
}
