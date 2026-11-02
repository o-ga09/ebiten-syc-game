package main

import (
	"math/rand/v2"
	"testing"
)

func plainBirth() Birth {
	return Birth{
		Talents:    []TalentID{TPushup, TLoud, TTall},
		Difficulty: difficulties[1],
		Family:     familyByID(FNormal),
	}
}

// setRow overwrites a row with a deterministic layout.
func setRow(w *World, n int, left, right Slot) {
	w.ensureRows(n + 1)
	w.rows[n] = Row{Slots: [2]Slot{left, right}}
}

var floor = Slot{Exists: true}

func TestJumpToExistingSlotAges(t *testing.T) {
	w := NewWorld(plainBirth(), 1, 0)
	for i := 1; i <= 10; i++ {
		setRow(w, i, floor, floor)
	}
	for range 10 {
		w.Jump(1)
		w.Tick += 3
	}
	if w.PlayerRow != 10 || w.Age() != 2 {
		t.Fatalf("row=%d age=%v", w.PlayerRow, w.Age())
	}
	if w.Points() != 200 {
		t.Fatalf("points=%d", w.Points())
	}
}

func TestJumpToMissingSlotLosesLife(t *testing.T) {
	w := NewWorld(plainBirth(), 1, 0)
	setRow(w, 1, floor, Slot{})
	w.Jump(1)
	if w.PlayerRow != 0 {
		t.Fatalf("player should stay, row=%d", w.PlayerRow)
	}
	if !w.Over || w.Lives != 0 || w.Cause == "" {
		t.Fatalf("expected game over with 1 life: over=%v lives=%d", w.Over, w.Lives)
	}
}

func TestToughTalentAndEasyAddLives(t *testing.T) {
	b := plainBirth()
	b.Talents[0] = TTough
	b.Difficulty = difficulties[0] // EASY
	w := NewWorld(b, 1, 0)
	if w.Lives != 3 {
		t.Fatalf("lives=%d", w.Lives)
	}
}

func TestProtectiveFamilySavesOnce(t *testing.T) {
	b := plainBirth()
	b.Family = familyByID(FProtect)
	w := NewWorld(b, 1, 0)
	setRow(w, 1, floor, Slot{})
	w.Jump(1)
	if w.Over || w.Lives != 1 {
		t.Fatalf("protective family should save: over=%v lives=%d", w.Over, w.Lives)
	}
}

func TestBugReverseFlipsControls(t *testing.T) {
	w := NewWorld(plainBirth(), 1, 0)
	setRow(w, 1, Slot{Exists: true, Bug: BugReverse}, floor)
	setRow(w, 2, floor, Slot{})
	w.Jump(0)
	if w.Reversed == 0 {
		t.Fatal("expected reversed")
	}
	w.Tick += 3
	w.Jump(1) // reversed: goes left
	if w.PlayerRow != 2 || w.PlayerLane != 0 {
		t.Fatalf("row=%d lane=%d", w.PlayerRow, w.PlayerLane)
	}
}

func TestBugProofIgnoresOneBug(t *testing.T) {
	b := plainBirth()
	b.Talents[0] = TBugProof
	w := NewWorld(b, 1, 0)
	setRow(w, 1, Slot{Exists: true, Bug: BugReverse}, floor)
	w.Jump(0)
	if w.Reversed != 0 || w.BugShield != 0 {
		t.Fatalf("reversed=%d shield=%d", w.Reversed, w.BugShield)
	}
}

func TestBugFallDropsThreeRows(t *testing.T) {
	w := NewWorld(plainBirth(), 1, 0)
	for i := 1; i <= 5; i++ {
		setRow(w, i, floor, floor)
	}
	setRow(w, 6, Slot{Exists: true, Bug: BugFall}, floor)
	for range 5 {
		w.Jump(0)
		w.Tick += 3
	}
	w.Jump(0)
	if w.PlayerRow != 3 || w.MaxRow != 6 || w.Lives != 1 {
		t.Fatalf("row=%d max=%d lives=%d", w.PlayerRow, w.MaxRow, w.Lives)
	}
}

func TestAchievementAddsPoints(t *testing.T) {
	w := NewWorld(plainBirth(), 1, 0)
	setRow(w, 1, Slot{Exists: true, Item: ItemAchievement, ItemName: "テスト"}, floor)
	w.Jump(0)
	if w.Achievements != 1 {
		t.Fatalf("achievements=%d", w.Achievements)
	}
	if got, want := w.Points(), int(0.2*100+500); got != want {
		t.Fatalf("points=%d want %d", got, want)
	}
}

func TestTimeCatchesIdlePlayer(t *testing.T) {
	w := NewWorld(plainBirth(), 1, 0)
	for range 60 * 30 {
		w.Update()
		if w.Over {
			break
		}
	}
	if !w.Over || w.Cause != "時間に追いつかれた" {
		t.Fatalf("over=%v cause=%q", w.Over, w.Cause)
	}
}

func TestPhoenixRevivesOnce(t *testing.T) {
	b := plainBirth()
	b.Talents[0] = TPhoenix
	w := NewWorld(b, 1, 0)
	w.TimeRow = 0.5
	w.Update()
	if w.Over {
		t.Fatal("phoenix should revive")
	}
	w.TimeRow = 0.5
	w.Update()
	if !w.Over {
		t.Fatal("second catch should end the game")
	}
}

func TestResetRewindsAfterDeath(t *testing.T) {
	w := NewWorld(plainBirth(), 1, 1)
	for range 120 {
		w.Update()
	}
	w.die("テスト")
	if !w.CanReset() {
		t.Fatal("expected reset available")
	}
	if !w.Reset() || w.Over || w.ResetsLeft != 0 {
		t.Fatalf("reset failed: over=%v left=%d", w.Over, w.ResetsLeft)
	}
	if w.Reset() {
		t.Fatal("no resets should remain")
	}
}

func TestGeneratedRowsAlwaysHaveAFloor(t *testing.T) {
	b := plainBirth()
	b.Difficulty = difficulties[4] // LUNATIC
	w := NewWorld(b, 7, 0)
	w.ensureRows(5000)
	for i, r := range w.rows {
		if !r.Slots[0].Exists && !r.Slots[1].Exists {
			t.Fatalf("row %d has no floor", i)
		}
		for lane, s := range r.Slots {
			if s.Bug == BugFall && !r.Slots[1-lane].Exists {
				t.Fatalf("row %d: unavoidable fall bug", i)
			}
		}
	}
}

func TestRollBirth(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for range 200 {
		b := RollBirth(rng)
		if len(b.Talents) != 3 {
			t.Fatalf("talents=%v", b.Talents)
		}
		seen := map[TalentID]bool{}
		for _, id := range b.Talents {
			if seen[id] {
				t.Fatalf("duplicate talent %s", id)
			}
			seen[id] = true
		}
	}
}

func TestComboKeyIsOrderIndependent(t *testing.T) {
	a := plainBirth()
	b := plainBirth()
	b.Talents = []TalentID{TTall, TPushup, TLoud}
	if a.ComboKey() != b.ComboKey() {
		t.Fatalf("%s != %s", a.ComboKey(), b.ComboKey())
	}
}

func TestEarlyRowsHaveBothFloors(t *testing.T) {
	for seed := range uint64(50) {
		w := NewWorld(plainBirth(), seed, 0)
		for n := 1; n <= safeRows; n++ {
			r := w.Row(n)
			if !r.Slots[0].Exists || !r.Slots[1].Exists {
				t.Fatalf("seed %d row %d: early rows must have both floors", seed, n)
			}
		}
	}
}

func TestMissingFloorStartsFallAnimation(t *testing.T) {
	b := plainBirth()
	b.Talents[0] = TTough // survive the fall
	w := NewWorld(b, 1, 0)
	setRow(w, 1, floor, Slot{})
	w.Jump(1)
	if !w.Falling() {
		t.Fatal("expected falling state")
	}
	if w.FallOffset() != 0 {
		t.Fatalf("offset at start = %v", w.FallOffset())
	}
	for range 10 {
		w.Update()
	}
	if w.FallOffset() <= 0 {
		t.Fatalf("player should drop, offset=%v", w.FallOffset())
	}
	for range fallTicks {
		w.Update()
	}
	if w.Falling() {
		t.Fatal("fall animation should end when the player survives")
	}
}
