package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand/v2"
)

const (
	// safeRows rows from the start always have floors on both sides so the
	// first taps can never fall (0-3 years old).
	safeRows = 15
	// fallTicks is how long the fall animation lasts after a missed step.
	fallTicks = 30

	rowsPerYear = 5
	rowH        = 26
	playerRowY  = screenH - 130 // screen Y of the player's row
)

var laneX = [2]float64{screenW/2 - 42, screenW/2 + 42}

type BugKind int

const (
	BugNone BugKind = iota
	BugFall
	BugReverse
	BugWarp
)

type ItemKind int

const (
	ItemNone ItemKind = iota
	ItemAchievement
	ItemRare
)

type Slot struct {
	Exists    bool
	Bug       BugKind
	Item      ItemKind
	ItemName  string
	Collected bool
}

type Row struct {
	Slots   [2]Slot
	NoRoute bool
}

type Enemy struct {
	Row   int
	X, VX float64
	Name  string
}

type Grave struct {
	Age   float64
	Name  string
	Cause string
	Lane  int
}

type Message struct {
	Text  string
	Life  int
	Color color.RGBA
}

type Particle struct {
	X, Y, VX, VY float64 // Y is in world pixels (row*rowH)
	Life, Max    float64
	Color        color.RGBA
}

// Event is emitted by World.Update so that the outer layer can play sounds.
type Event int

const (
	EvJump Event = iota
	EvMiss
	EvItem
	EvRare
	EvBug
	EvOver
	EvReset
	EvSave
)

type snapshot struct {
	playerRow, playerLane, maxRow int
	lives, achievements, rare     int
	timeRow                       float64
	enemies                       []Enemy
	reversed                      int
}

// World is the simulation of one life. It has no Ebitengine dependency.
type World struct {
	Birth Birth

	rows []Row

	PlayerRow, PlayerLane int
	MaxRow                int
	VisRow, VisX          float64
	jumpTick              int
	fallTick              int // tick when the player stepped into the void, -1 = not falling
	fallLane              int
	lastJump              int
	pending               int // buffered jump direction, -1 = none

	Lives        int
	TimeRow      float64
	Enemies      []*Enemy
	Achievements int
	RarePoints   int
	Tick         int
	Invincible   int
	Reversed     int
	BugShield    int
	protectUsed  bool
	phoenixUsed  bool
	Shake        float64

	Over  bool
	Cause string

	ResetsLeft int
	snapshots  []snapshot

	Graves    []Grave
	Messages  []*Message
	Particles []*Particle

	enemyTimer  int
	noRouteLeft int
	rng         *rand.Rand
	events      []Event
}

func NewWorld(b Birth, seed uint64, resets int) *World {
	w := &World{
		Birth:      b,
		Lives:      1 + b.Difficulty.LifeBonus,
		TimeRow:    -6,
		ResetsLeft: resets,
		pending:    -1,
		lastJump:   -10,
		fallTick:   -1,
		rng:        rand.New(rand.NewPCG(seed, seed*31+7)),
		enemyTimer: 180,
	}
	if b.Has(TTough) {
		w.Lives++
	}
	if b.Has(TBugProof) {
		w.BugShield = 1
	}
	if b.Has(TSprinter) {
		w.TimeRow = -11
	}
	if b.Has(TOversleep) {
		w.TimeRow = -2.5
	}
	if b.Family.ID == FRich {
		w.Achievements = 1
	}
	w.rows = append(w.rows, Row{Slots: [2]Slot{{Exists: true}, {Exists: true}}})
	w.VisX = laneX[0]
	w.ensureRows(40)
	return w
}

func (w *World) Events() []Event {
	ev := w.events
	w.events = nil
	return ev
}

func (w *World) emit(e Event) { w.events = append(w.events, e) }

// Age is the maximum age reached in years.
func (w *World) Age() float64 { return float64(w.MaxRow) / rowsPerYear }

func (w *World) Stage() Stage {
	if w.Birth.Family.ID == FMover {
		// Mover families see a random stage background each 2 years.
		return stages[(w.MaxRow/(rowsPerYear*2)*7+3)%len(stages)]
	}
	return stages[stageIndexForAge(w.Age())]
}

// RealStage ignores the mover family's cosmetic shuffle.
func (w *World) RealStage() Stage { return stages[stageIndexForAge(w.Age())] }

func (w *World) achievementValue() float64 {
	if w.Birth.Has(TScholar) {
		return 600
	}
	return 500
}

func (w *World) PointMultiplier() float64 {
	m := w.Birth.Difficulty.PointMul
	if w.Birth.Family.ID == FFree {
		m *= 1.2
	}
	if w.Birth.Has(TThrifty) {
		m *= 1.05
	}
	return m
}

// Points is the "life points" score.
func (w *World) Points() int {
	base := w.Age()*100 + float64(w.Achievements)*w.achievementValue() + float64(w.RarePoints)
	return int(base * w.PointMultiplier())
}

func (w *World) Row(n int) *Row {
	if n < 0 {
		return &Row{}
	}
	w.ensureRows(n + 1)
	return &w.rows[n]
}

func (w *World) ensureRows(n int) {
	for len(w.rows) < n {
		w.rows = append(w.rows, w.genRow(len(w.rows)))
	}
}

func (w *World) genRow(n int) Row {
	age := float64(n) / rowsPerYear
	d := w.Birth.Difficulty
	var r Row

	if w.noRouteLeft == 0 && n > 40 && w.rng.Float64() < math.Min(0.012+age*0.0002, 0.03)*d.BugMul {
		w.noRouteLeft = 5 + w.rng.IntN(6)
	}
	pBoth := math.Max(0.2, 0.62-age*0.006)
	if w.noRouteLeft > 0 {
		w.noRouteLeft--
		r.NoRoute = true
		pBoth = 0
	}
	if n <= safeRows {
		pBoth = 1
	}
	if w.rng.Float64() < pBoth {
		r.Slots[0].Exists = true
		r.Slots[1].Exists = true
	} else {
		r.Slots[w.rng.IntN(2)].Exists = true
	}

	pBug := 0.0
	if n > 15 {
		pBug = math.Min(0.03+age*0.0015, 0.15) * d.BugMul
	}
	lucky := 1.0
	if w.Birth.Has(TLucky) {
		lucky = 2
	}
	both := r.Slots[0].Exists && r.Slots[1].Exists
	for i := range r.Slots {
		s := &r.Slots[i]
		if !s.Exists {
			continue
		}
		if w.rng.Float64() < pBug {
			switch k := w.rng.IntN(3); {
			case k == 0 && both:
				// A floor that you fall through is only placed where the
				// other side is safe, so it is always avoidable.
				s.Bug = BugFall
			case k == 1 || k == 0:
				s.Bug = BugReverse
			default:
				s.Bug = BugWarp
			}
			continue
		}
		switch x := w.rng.Float64(); {
		case n > 5 && x < 0.008*lucky:
			s.Item = ItemRare
		case n > 3 && x < 0.008*lucky+0.05:
			s.Item = ItemAchievement
			st := stages[stageIndexForAge(age)]
			s.ItemName = st.Achievements[w.rng.IntN(len(st.Achievements))]
		}
	}
	return r
}

// timeSpeed returns rows per tick.
func (w *World) timeSpeed() float64 {
	age := w.Age()
	rps := 0.8 + 2.6*(1-math.Exp(-age/50))
	rps *= w.Birth.Difficulty.TimeMul
	if w.Birth.Has(TEarly) {
		rps *= 0.9
	}
	if w.Birth.Family.ID == FFree {
		rps *= 1.15
	}
	return rps / 60
}

// Jump requests a jump to the given lane (0 = left, 1 = right).
func (w *World) Jump(dir int) {
	if w.Over {
		return
	}
	if w.Tick-w.lastJump < 3 {
		w.pending = dir
		return
	}
	w.doJump(dir)
}

func (w *World) doJump(dir int) {
	w.lastJump = w.Tick
	w.jumpTick = w.Tick
	if w.Reversed > 0 {
		dir = 1 - dir
	}
	if w.Birth.Has(TLost) && w.rng.Float64() < 0.06 {
		dir = 1 - dir
		w.msg("あれ？こっちだっけ", color.RGBA{0xff, 0xff, 0xff, 0xff})
	}
	target := w.PlayerRow + 1
	if w.Birth.Has(TDoubleJump) && w.rng.Float64() < 0.15 {
		if s := w.Row(target + 1).Slots[dir]; s.Exists && s.Bug == BugNone {
			target++
			w.msg("2段ジャンプ！", rarityColors[RaritySR])
		}
	}
	w.emit(EvJump)
	slot := &w.Row(target).Slots[dir]
	if !slot.Exists {
		w.burst(laneX[dir], float64(target)*rowH, color.RGBA{0xff, 0xff, 0xff, 0xff}, 6)
		if w.Invincible == 0 {
			w.fallTick = w.Tick
			w.fallLane = dir
			w.msg("足をふみはずした！", colRed)
		}
		w.miss("足をふみはずした")
		return
	}
	w.PlayerRow = target
	w.PlayerLane = dir
	w.fallTick = -1
	w.MaxRow = max(w.MaxRow, w.PlayerRow)
	w.land(slot)
}

func (w *World) land(slot *Slot) {
	if slot.Bug != BugNone {
		if w.BugShield > 0 {
			w.BugShield--
			slot.Bug = BugNone
			w.msg("バグを回避した（耐性）", rarityColors[RarityR])
		} else {
			w.triggerBug(slot)
			return
		}
	}
	w.collect(slot)
	if w.Birth.Has(TMagnet) {
		w.collect(&w.Row(w.PlayerRow).Slots[1-w.PlayerLane])
	}
}

func (w *World) triggerBug(slot *Slot) {
	bug := slot.Bug
	w.emit(EvBug)
	w.Shake = 6
	switch bug {
	case BugFall:
		w.msg("ERR: 床が実装されていません", color.RGBA{0xff, 0x50, 0x50, 0xff})
		w.moveTo(max(0, w.PlayerRow-3))
	case BugReverse:
		w.msg("ERR: 左右の設定が反転しました", color.RGBA{0xff, 0x50, 0x50, 0xff})
		w.Reversed = 300
	case BugWarp:
		w.msg("ERR: 座標ずれ（ラッキー）", color.RGBA{0x50, 0xff, 0x90, 0xff})
		w.moveTo(w.PlayerRow + 5)
		w.MaxRow = max(w.MaxRow, w.PlayerRow)
	}
}

// moveTo places the player on an existing, bug-free slot of the given row
// (searching downward if the row has none).
func (w *World) moveTo(row int) {
	for r := row; r >= 0; r-- {
		for _, lane := range []int{w.PlayerLane, 1 - w.PlayerLane} {
			s := &w.Row(r).Slots[lane]
			if s.Exists && s.Bug == BugNone {
				w.PlayerRow, w.PlayerLane = r, lane
				return
			}
		}
	}
	w.PlayerRow = 0
}

func (w *World) collect(s *Slot) {
	if !s.Exists || s.Collected || s.Item == ItemNone {
		return
	}
	s.Collected = true
	y := float64(w.PlayerRow) * rowH
	switch s.Item {
	case ItemAchievement:
		w.Achievements++
		w.msg("実績解除："+s.ItemName, color.RGBA{0xff, 0xe0, 0x40, 0xff})
		w.burst(laneX[w.PlayerLane], y, color.RGBA{0xff, 0xe0, 0x40, 0xff}, 10)
		w.emit(EvItem)
	case ItemRare:
		pts := 1000 + w.rng.IntN(4001)
		if w.Birth.Has(TGambler) {
			pts = w.rng.IntN(10001)
		}
		w.RarePoints += pts
		ev := rareEvents[w.rng.IntN(len(rareEvents))]
		w.msg(fmt.Sprintf("%s +%d", ev.Text, pts), rarityColors[RaritySSR])
		w.burst(laneX[w.PlayerLane], y, rarityColors[RaritySSR], 24)
		w.emit(EvRare)
	}
}

func (w *World) miss(cause string) {
	if w.Invincible > 0 {
		return
	}
	if w.Birth.Family.ID == FProtect && !w.protectUsed && w.Age() < 20 {
		w.protectUsed = true
		w.Invincible = 90
		w.msg("親が守ってくれた（過保護）", rarityColors[RarityR])
		return
	}
	w.Lives--
	w.Invincible = 90
	w.Shake = 10
	w.emit(EvMiss)
	if w.Lives <= 0 {
		w.Lives = 0
		w.die(cause)
	}
}

func (w *World) die(cause string) {
	w.Over = true
	w.Cause = cause
	w.emit(EvOver)
}

// Update advances the world by one tick.
func (w *World) Update() {
	w.Tick++
	w.updateEffects()
	w.VisRow += (float64(w.PlayerRow) - w.VisRow) * 0.4
	targetX := laneX[w.PlayerLane]
	if w.Falling() {
		targetX = laneX[w.fallLane]
	} else if w.fallTick >= 0 {
		w.fallTick = -1
	}
	w.VisX += (targetX - w.VisX) * 0.4
	if w.Over {
		return
	}
	if w.Tick%30 == 0 {
		w.takeSnapshot()
	}
	if w.pending >= 0 && w.Tick-w.lastJump >= 3 {
		d := w.pending
		w.pending = -1
		w.doJump(d)
		if w.Over {
			return
		}
	}
	if w.Invincible > 0 {
		w.Invincible--
	}
	if w.Reversed > 0 {
		w.Reversed--
		if w.Reversed == 0 {
			w.msg("設定がもとにもどった", color.RGBA{0xff, 0xff, 0xff, 0xff})
		}
	}
	w.ensureRows(w.PlayerRow + 30)

	// Time rises from below.
	w.TimeRow += w.timeSpeed()
	w.TimeRow = math.Max(w.TimeRow, float64(w.PlayerRow)-10)
	if w.TimeRow >= float64(w.PlayerRow)+0.2 {
		if w.Birth.Has(TPhoenix) && !w.phoenixUsed {
			w.phoenixUsed = true
			w.TimeRow = float64(w.PlayerRow) - 10
			w.Invincible = 90
			w.msg("不死鳥のごとく復活！", rarityColors[RaritySSR])
			w.burst(w.VisX, float64(w.PlayerRow)*rowH, rarityColors[RaritySSR], 30)
		} else {
			w.die("時間に追いつかれた")
			return
		}
	}

	w.updateEnemies()
}

func (w *World) updateEnemies() {
	age := w.Age()
	d := w.Birth.Difficulty
	w.enemyTimer--
	if w.enemyTimer <= 0 && w.PlayerRow > 8 {
		st := w.RealStage()
		dir := 1.0
		x := -20.0
		if w.rng.IntN(2) == 0 {
			dir, x = -1, screenW+20
		}
		speed := math.Min(0.7+age*0.012, 2.0)
		if w.Birth.Has(TDodge) {
			speed *= 0.85
		}
		w.Enemies = append(w.Enemies, &Enemy{
			Row:  w.PlayerRow + 4 + w.rng.IntN(6),
			X:    x,
			VX:   speed * dir,
			Name: st.Enemies[w.rng.IntN(len(st.Enemies))],
		})
		w.enemyTimer = int(math.Max(45, 160-age*1.3) / d.EnemyMul)
	}
	kept := w.Enemies[:0]
	for _, e := range w.Enemies {
		e.X += e.VX
		if e.X < -40 || e.X > screenW+40 || float64(e.Row) < w.TimeRow-1 {
			continue
		}
		if e.Row == w.PlayerRow && math.Abs(e.X-laneX[w.PlayerLane]) < 16 && w.Invincible == 0 {
			if w.Birth.Family.ID == FBig && w.rng.Float64() < 0.2 {
				w.msg("家族がかばってくれた", rarityColors[RarityR])
				w.Invincible = 60
				continue
			}
			w.burst(e.X, float64(e.Row)*rowH, color.RGBA{0xff, 0x40, 0x40, 0xff}, 14)
			w.miss(e.Name + "にやられた")
			if w.Over {
				return
			}
			continue
		}
		kept = append(kept, e)
	}
	w.Enemies = kept
}

func (w *World) takeSnapshot() {
	s := snapshot{
		playerRow: w.PlayerRow, playerLane: w.PlayerLane, maxRow: w.MaxRow,
		lives: w.Lives, achievements: w.Achievements, rare: w.RarePoints,
		timeRow: w.TimeRow, reversed: w.Reversed,
	}
	for _, e := range w.Enemies {
		s.enemies = append(s.enemies, *e)
	}
	w.snapshots = append(w.snapshots, s)
	if len(w.snapshots) > 8 {
		w.snapshots = w.snapshots[1:]
	}
}

// CanReset reports whether "reset & reboot" is available right now.
func (w *World) CanReset() bool { return w.ResetsLeft > 0 && len(w.snapshots) > 0 }

// Reset rewinds the world about three seconds.
func (w *World) Reset() bool {
	if !w.CanReset() {
		return false
	}
	idx := max(0, len(w.snapshots)-6)
	s := w.snapshots[idx]
	w.snapshots = w.snapshots[:idx]
	w.PlayerRow, w.PlayerLane, w.MaxRow = s.playerRow, s.playerLane, s.maxRow
	w.Lives = max(1, s.lives)
	w.Achievements, w.RarePoints = s.achievements, s.rare
	w.TimeRow = s.timeRow - 3
	w.Reversed = s.reversed
	w.Enemies = nil
	for _, e := range s.enemies {
		w.Enemies = append(w.Enemies, &e)
	}
	w.Invincible = 120
	w.Over = false
	w.Cause = ""
	w.ResetsLeft--
	w.pending = -1
	w.fallTick = -1
	w.msg("リセット再起動しました", color.RGBA{0x80, 0xff, 0xff, 0xff})
	w.emit(EvReset)
	return true
}

func (w *World) msg(s string, c color.RGBA) {
	w.Messages = append(w.Messages, &Message{Text: s, Life: 120, Color: c})
	if len(w.Messages) > 3 {
		w.Messages = w.Messages[len(w.Messages)-3:]
	}
}

func (w *World) burst(x, y float64, c color.RGBA, n int) {
	for range n {
		a := w.rng.Float64() * math.Pi * 2
		s := 0.8 + w.rng.Float64()*2.5
		life := 18 + w.rng.Float64()*20
		w.Particles = append(w.Particles, &Particle{
			X: x, Y: y, VX: math.Cos(a) * s, VY: math.Sin(a) * s,
			Life: life, Max: life, Color: c,
		})
	}
}

func (w *World) updateEffects() {
	if w.Shake > 0 {
		w.Shake *= 0.85
		if w.Shake < 0.5 {
			w.Shake = 0
		}
	}
	ps := w.Particles[:0]
	for _, p := range w.Particles {
		p.X += p.VX
		p.Y -= p.VY // world Y grows upward
		p.VY -= 0.1
		p.Life--
		if p.Life > 0 {
			ps = append(ps, p)
		}
	}
	w.Particles = ps
	ms := w.Messages[:0]
	for _, m := range w.Messages {
		m.Life--
		if m.Life > 0 {
			ms = append(ms, m)
		}
	}
	w.Messages = ms
}

// ShakeOffset returns a random screen offset.
func (w *World) ShakeOffset() (float64, float64) {
	if w.Shake == 0 {
		return 0, 0
	}
	return math.Round((w.rng.Float64()*2 - 1) * w.Shake), math.Round((w.rng.Float64()*2 - 1) * w.Shake)
}

// VisibleAhead is how many rows ahead are visible within a no-route zone.
func (w *World) VisibleAhead() int {
	if w.Birth.Has(TForesight) {
		return 6
	}
	return 2
}

// Falling reports whether the fall animation is playing. After a fatal
// fall it keeps playing so the player drops off the screen.
func (w *World) Falling() bool {
	if w.fallTick < 0 {
		return false
	}
	return w.Over || w.Tick-w.fallTick < fallTicks
}

// FallOffset is how many pixels the player has dropped below its row.
func (w *World) FallOffset() float64 {
	if !w.Falling() {
		return 0
	}
	t := float64(w.Tick - w.fallTick)
	if !w.Over {
		// Drop, then pop back up onto the original step.
		half := float64(fallTicks) / 2
		if t > half {
			t = float64(fallTicks) - t
		}
	}
	return 0.12 * t * t
}

// JumpHop returns the vertical hop offset in pixels for the jump animation.
func (w *World) JumpHop() float64 {
	t := w.Tick - w.jumpTick
	if t >= 8 || w.jumpTick == 0 {
		return 0
	}
	return math.Sin(math.Pi*float64(t)/8) * 7
}
