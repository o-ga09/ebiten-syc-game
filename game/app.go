package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	keyName = "jinsei.name"
	keyRuns = "jinsei.runs"
)

type Scene interface {
	Update(a *App)
	Draw(dst *ebiten.Image)
}

type App struct {
	scene Scene
	in    Input
	sound *Sound
	rng   *rand.Rand

	Name       string
	Runs       int
	askingName bool
	onName     func()
}

func NewApp() *App {
	seed := uint64(time.Now().UnixNano())
	a := &App{
		sound: NewSound(),
		rng:   rand.New(rand.NewPCG(seed, seed>>7|1)),
		Name:  loadStore(keyName),
	}
	a.Runs, _ = strconv.Atoi(loadStore(keyRuns))
	a.scene = NewTitleScene(a)
	return a
}

func (a *App) Update() error {
	a.in.Poll()
	if a.askingName {
		if n, ok := pollName(); ok {
			a.askingName = false
			if n != "" {
				a.Name = n
				saveStore(keyName, n)
			}
			if a.onName != nil && a.Name != "" {
				f := a.onName
				a.onName = nil
				f()
			}
		}
		return nil
	}
	a.scene.Update(a)
	return nil
}

func (a *App) Draw(screen *ebiten.Image) { a.scene.Draw(screen) }

func (a *App) Layout(int, int) (int, int) { return screenW, screenH }

// AskName opens the name dialog and calls then when a name is set.
func (a *App) AskName(then func()) {
	a.askingName = true
	a.onName = then
	askName(a.Name)
}

// resetsFor returns how many resets one life gets after n lives played.
func resetsFor(runs int) int {
	switch {
	case runs >= 10:
		return 2
	case runs >= 3:
		return 1
	default:
		return 0
	}
}

const resetUnlockRuns = 3

func (a *App) StartLife() {
	a.Runs++
	saveStore(keyRuns, strconv.Itoa(a.Runs))
	a.sound.Start()
	a.scene = NewGachaScene(a, RollBirth(a.rng))
}

func playEvents(a *App, evs []Event) {
	for _, e := range evs {
		a.sound.Play(e)
	}
}

// ---------------------------------------------------------------- title

type TitleScene struct {
	tick                int
	start, rank, rename *Button
}

func NewTitleScene(a *App) *TitleScene {
	return &TitleScene{
		start:  NewButton(screenW/2, 300, 170, 34, "うまれる", color.RGBA{0xe8, 0x40, 0x60, 0xff}),
		rank:   NewButton(screenW/2, 344, 170, 28, "ランキング", color.RGBA{0x40, 0x70, 0xd0, 0xff}),
		rename: NewButton(screenW/2, 384, 170, 24, "なまえをかえる", color.RGBA{0x50, 0x50, 0x68, 0xff}),
	}
}

func (s *TitleScene) Update(a *App) {
	s.tick++
	switch {
	case a.in.TappedButton(s.rank):
		a.sound.Cursor()
		a.scene = NewRankingScene(a, s)
	case a.in.TappedButton(s.rename):
		a.sound.Cursor()
		a.AskName(nil)
	case a.in.TappedButton(s.start) || a.in.OK:
		if a.Name == "" {
			a.AskName(a.StartLife)
			return
		}
		a.StartLife()
	}
}

func (s *TitleScene) Draw(dst *ebiten.Image) {
	dst.Fill(color.RGBA{0x18, 0x14, 0x30, 0xff})
	for i := range 40 {
		x := math.Mod(float64(i*53+s.tick/2), screenW)
		y := math.Mod(float64(i*97)+float64(s.tick)*0.3*float64(i%3+1), screenH)
		fillRect(dst, x, y, 1, 1, color.RGBA{0xff, 0xff, 0xff, 0x80})
	}
	drawTextScaled(dst, "じんせい", screenW/2, 70, colWhite, AlignCenter, 3)
	cols := []color.RGBA{{0xff, 0x4d, 0x6d, 0xff}, {0xff, 0xd1, 0x66, 0xff}, {0x06, 0xd6, 0xa0, 0xff}, {0x4c, 0xc9, 0xf0, 0xff}}
	for i, ch := range "GAME" {
		y := 120 + math.Sin(float64(s.tick)/12+float64(i))*4
		drawTextScaled(dst, string(ch), screenW/2-72+float64(i)*48+12, y, cols[i], AlignCenter, 5)
	}
	drawText(dst, "（仮）", screenW/2+100, 186, colDim, AlignCenter)
	drawText(dst, "ガチャで決まる人生アクション", screenW/2, 210, colYellow, AlignCenter)
	drawText(dst, "左右タップで のぼれ！", screenW/2, 240, colWhite, AlignCenter)
	s.start.Draw(dst)
	s.rank.Draw(dst)
	s.rename.Draw(dst)
}

// ---------------------------------------------------------------- gacha

type gachaCard struct {
	Category, Name, Desc string
	Rarity               Rarity
}

type GachaScene struct {
	birth    Birth
	cards    []gachaCard
	revealed int
	tick     int
	unlock   string
}

func NewGachaScene(a *App, b Birth) *GachaScene {
	s := &GachaScene{birth: b}
	for _, id := range b.Talents {
		t, _ := talentByID(id)
		s.cards = append(s.cards, gachaCard{"才能", t.Name, t.Desc, t.Rarity})
	}
	s.cards = append(s.cards,
		gachaCard{"難易度", b.Difficulty.Name, fmt.Sprintf("ポイント×%.1f", b.Difficulty.PointMul), b.Difficulty.Rarity},
		gachaCard{"家庭環境", b.Family.Name, b.Family.Desc, b.Family.Rarity},
	)
	switch {
	case a.Runs == resetUnlockRuns:
		s.unlock = "リセット機能が解放されました（1回）"
	case a.Runs == 10:
		s.unlock = "リセットが2回使えるようになりました"
	}
	return s
}

func (s *GachaScene) Update(a *App) {
	s.tick++
	if s.revealed < len(s.cards) && s.tick%22 == 0 {
		s.reveal(a)
	}
	if a.in.Tapped() {
		if s.revealed < len(s.cards) {
			for s.revealed < len(s.cards) {
				s.reveal(a)
			}
			return
		}
		a.scene = NewPlayScene(a, s.birth)
	}
}

func (s *GachaScene) reveal(a *App) {
	c := s.cards[s.revealed]
	s.revealed++
	if c.Rarity >= RaritySR {
		a.sound.Play(EvRare)
	} else {
		a.sound.Cursor()
	}
}

func (s *GachaScene) Draw(dst *ebiten.Image) {
	dst.Fill(color.RGBA{0x20, 0x18, 0x38, 0xff})
	drawTextScaled(dst, "生まれガチャ", screenW/2, 18, colYellow, AlignCenter, 2)
	for i, c := range s.cards {
		y := 56 + float64(i)*62
		if i >= s.revealed {
			drawPanel(dst, 16, y, screenW-32, 54)
			drawText(dst, "？？？", screenW/2, y+21, colDim, AlignCenter)
			continue
		}
		rc := rarityColors[c.Rarity]
		if c.Rarity == RaritySSR {
			h := float64(s.tick%60) / 60
			rc = color.RGBA{uint8(160 + 95*math.Sin(h*6.28)), uint8(160 + 95*math.Sin(h*6.28+2)), uint8(160 + 95*math.Sin(h*6.28+4)), 0xff}
		}
		drawPanel(dst, 16, y, screenW-32, 54)
		fillRect(dst, 16, y, 4, 54, rc)
		drawText(dst, c.Rarity.String(), 28, y+6, rc, AlignLeft)
		drawText(dst, c.Category, screenW-24, y+6, colDim, AlignRight)
		drawTextScaled(dst, c.Name, 28, y+20, colWhite, AlignLeft, 1)
		drawText(dst, c.Desc, 28, y+36, colDim, AlignLeft)
	}
	y := 56 + float64(len(s.cards))*62
	drawText(dst, "※難易度の変更・返品はできません", screenW/2, y, colDim, AlignCenter)
	if s.unlock != "" {
		drawText(dst, s.unlock, screenW/2, y+18, color.RGBA{0x80, 0xff, 0xff, 0xff}, AlignCenter)
	}
	if s.revealed >= len(s.cards) && s.tick/20%2 == 0 {
		drawTextScaled(dst, "タップで誕生！", screenW/2, screenH-40, colWhite, AlignCenter, 2)
	}
}

// ---------------------------------------------------------------- play

type PlayScene struct {
	a       *App
	world   *World
	sprites *Sprites
	canvas  *ebiten.Image
	graves  *Async[[]GraveDTO]
	ticks   int

	paused      bool
	showGuide   bool
	deathTimer  int
	pauseBtn    *Button
	resetBtn    *Button
	resumeBtn   *Button
	guideBtn    *Button
	giveUpBtn   *Button
	doResetBtn  *Button
	noResetBtn  *Button
	promptReset bool
}

func NewPlayScene(a *App, b Birth) *PlayScene {
	s := &PlayScene{
		a:          a,
		world:      NewWorld(b, a.rng.Uint64(), resetsFor(a.Runs)),
		sprites:    NewSprites(b.Look),
		canvas:     ebiten.NewImage(screenW, screenH),
		graves:     FetchGraves(0, 400),
		pauseBtn:   NewButton(screenW-16, 6, 24, 22, "II", color.RGBA{0x50, 0x50, 0x68, 0xff}),
		resetBtn:   NewButton(screenW-44, 6, 24, 22, "R", color.RGBA{0x30, 0x90, 0x90, 0xff}),
		resumeBtn:  NewButton(screenW/2, 170, 170, 30, "つづける", color.RGBA{0x40, 0x90, 0x50, 0xff}),
		guideBtn:   NewButton(screenW/2, 210, 170, 30, "攻略本を見る", color.RGBA{0x40, 0x70, 0xd0, 0xff}),
		giveUpBtn:  NewButton(screenW/2, 250, 170, 30, "あきらめる", color.RGBA{0xa0, 0x40, 0x40, 0xff}),
		doResetBtn: NewButton(screenW/2, 230, 190, 32, "リセット再起動", color.RGBA{0x30, 0x90, 0x90, 0xff}),
		noResetBtn: NewButton(screenW/2, 272, 190, 28, "受け入れる", color.RGBA{0x60, 0x50, 0x60, 0xff}),
	}
	a.sound.Loud = b.Has(TLoud)
	return s
}

func (s *PlayScene) Update(a *App) {
	w := s.world
	if s.graves.Done() && s.graves.Err == nil {
		for _, g := range s.graves.Value {
			// Thin out graves so they don't overlap (newest first from the API).
			if slices.ContainsFunc(w.Graves, func(o Grave) bool { return math.Abs(o.Age-g.Age) < 1.2 }) {
				continue
			}
			w.Graves = append(w.Graves, Grave{Age: g.Age, Name: g.Name, Cause: g.Cause, Lane: a.rng.IntN(2)})
		}
		s.graves = nil
	}

	if s.paused {
		switch {
		case a.in.TappedButton(s.resumeBtn) || a.in.Back:
			s.paused, s.showGuide = false, false
		case a.in.TappedButton(s.guideBtn):
			a.sound.Cursor()
			s.showGuide = !s.showGuide
		case a.in.TappedButton(s.giveUpBtn):
			s.paused = false
			w.ResetsLeft = 0
			w.die("人生をあきらめた")
			playEvents(a, w.Events())
		}
		return
	}

	if w.Over {
		w.Update()
		s.deathTimer++
		if s.promptReset {
			switch {
			case a.in.TappedButton(s.doResetBtn) || a.in.Reset:
				s.promptReset = false
				s.deathTimer = 0
				w.Reset()
			case a.in.TappedButton(s.noResetBtn) || a.in.Back:
				s.finish(a)
			}
		} else if s.deathTimer == 40 && w.CanReset() {
			s.promptReset = true
		} else if s.deathTimer > 80 {
			s.finish(a)
		}
		playEvents(a, w.Events())
		return
	}

	s.ticks++
	if a.in.Back || a.in.TappedButton(s.pauseBtn) {
		s.paused = true
		return
	}
	if a.in.Reset || a.in.TappedButton(s.resetBtn) {
		s.tryReset(a)
	}
	for _, p := range a.in.Taps {
		if s.pauseBtn.Hit(p) || s.resetBtn.Hit(p) {
			continue
		}
		if p.X < screenW/2 {
			w.Jump(0)
		} else {
			w.Jump(1)
		}
	}
	if a.in.Left {
		w.Jump(0)
	}
	if a.in.Right {
		w.Jump(1)
	}
	w.Update()
	playEvents(a, w.Events())
}

func (s *PlayScene) tryReset(a *App) {
	w := s.world
	switch {
	case a.Runs < resetUnlockRuns:
		w.msg(fmt.Sprintf("この機能は解放されていません（あと%d回）", resetUnlockRuns-a.Runs), colDim)
	case !w.CanReset():
		w.msg("リセット権が残っていません", colDim)
	default:
		w.Reset()
	}
}

func (s *PlayScene) finish(a *App) {
	a.scene = NewResultScene(a, s.world, float64(s.ticks)/60)
}

func (s *PlayScene) Draw(dst *ebiten.Image) {
	w := s.world
	drawWorld(s.canvas, w, s.sprites)
	op := &ebiten.DrawImageOptions{}
	ox, oy := w.ShakeOffset()
	op.GeoM.Translate(ox, oy)
	dst.Fill(colBlack)
	dst.DrawImage(s.canvas, op)
	s.drawHUD(dst)

	if s.paused {
		fillRect(dst, 0, 0, screenW, screenH, color.RGBA{0, 0, 0, 0xa0})
		drawTextScaled(dst, "ポーズ", screenW/2, 110, colWhite, AlignCenter, 2)
		s.resumeBtn.Draw(dst)
		s.guideBtn.Draw(dst)
		s.giveUpBtn.Draw(dst)
		if s.showGuide {
			drawPanel(dst, 14, 296, screenW-28, 76)
			drawText(dst, "『あなた専用 攻略本』", screenW/2, 304, colYellow, AlignCenter)
			drawText(dst, "ただいま品切れです。", screenW/2, 322, colWhite, AlignCenter)
			drawText(dst, "入荷予定：未定", screenW/2, 338, colWhite, AlignCenter)
			drawText(dst, "（他の人の攻略本は合いません）", screenW/2, 354, colDim, AlignCenter)
		}
	}
	if w.Over {
		if s.promptReset {
			fillRect(dst, 0, 0, screenW, screenH, color.RGBA{0, 0, 0, 0x90})
			drawTextScaled(dst, "GAME OVER?", screenW/2, 150, colRed, AlignCenter, 2)
			drawText(dst, w.Cause, screenW/2, 186, colWhite, AlignCenter)
			drawText(dst, fmt.Sprintf("リセット権 のこり%d", w.ResetsLeft), screenW/2, 206, color.RGBA{0x80, 0xff, 0xff, 0xff}, AlignCenter)
			s.doResetBtn.Draw(dst)
			s.noResetBtn.Draw(dst)
		} else if s.deathTimer > 10 {
			drawTextScaled(dst, "GAME OVER", screenW/2, 180, colRed, AlignCenter, 3)
		}
	}
}

func (s *PlayScene) drawHUD(dst *ebiten.Image) {
	w := s.world
	fillRect(dst, 0, 0, screenW, 34, color.RGBA{0x10, 0x10, 0x20, 0xc0})
	drawText(dst, fmt.Sprintf("%dpt", w.Points()), 6, 4, colYellow, AlignLeft)
	hearts := ""
	for range w.Lives {
		hearts += "♥"
	}
	drawText(dst, hearts, screenW-62, 4, colRed, AlignRight)
	drawText(dst, fmt.Sprintf("%.1f歳 %s", w.Age(), w.RealStage().Name), 6, 19, colWhite, AlignLeft)
	if w.Reversed > 0 && w.Tick/10%2 == 0 {
		drawText(dst, "左右反転中", screenW-62, 19, colRed, AlignRight)
	}
	s.pauseBtn.Draw(dst)
	if s.a.Runs >= resetUnlockRuns {
		s.resetBtn.Label = strconv.Itoa(w.ResetsLeft)
		s.resetBtn.Color = color.RGBA{0x30, 0x90, 0x90, 0xff}
	} else {
		s.resetBtn.Label = "R"
		s.resetBtn.Color = color.RGBA{0x40, 0x40, 0x48, 0xff}
	}
	s.resetBtn.Draw(dst)

	for i, m := range w.Messages {
		y := 44 + float64(i)*18
		tw := textWidth(m.Text)
		fillRect(dst, screenW/2-tw/2-6, y-2, tw+12, 16, color.RGBA{0, 0, 0, 0x90})
		drawText(dst, m.Text, screenW/2, y, m.Color, AlignCenter)
	}
}

// ---------------------------------------------------------------- result

type ResultScene struct {
	world  *World
	submit *Async[RunResult]
	tick   int
	again  *Button
	rank   *Button
	title  *Button
}

func NewResultScene(a *App, w *World, secs float64) *ResultScene {
	ids := make([]string, len(w.Birth.Talents))
	for i, t := range w.Birth.Talents {
		ids[i] = string(t)
	}
	req := RunRequest{
		Name:         a.Name,
		Points:       w.Points(),
		Age:          math.Round(w.Age()*10) / 10,
		Achievements: w.Achievements,
		Difficulty:   string(w.Birth.Difficulty.ID),
		Family:       string(w.Birth.Family.ID),
		Talents:      ids,
		Look:         w.Birth.LookKey(),
		Cause:        w.Cause,
		PlaySeconds:  secs,
	}
	return &ResultScene{
		world:  w,
		submit: SubmitRun(req),
		again:  NewButton(screenW/2, 384, 190, 32, "もう一度うまれる", color.RGBA{0xe8, 0x40, 0x60, 0xff}),
		rank:   NewButton(screenW/2-50, 426, 92, 26, "ランキング", color.RGBA{0x40, 0x70, 0xd0, 0xff}),
		title:  NewButton(screenW/2+50, 426, 92, 26, "タイトル", color.RGBA{0x50, 0x50, 0x68, 0xff}),
	}
}

func (s *ResultScene) Update(a *App) {
	s.tick++
	if s.tick < 30 {
		return
	}
	switch {
	case a.in.TappedButton(s.again) || a.in.OK:
		a.StartLife()
	case a.in.TappedButton(s.rank):
		a.sound.Cursor()
		a.scene = NewRankingScene(a, s)
	case a.in.TappedButton(s.title) || a.in.Back:
		a.sound.Cursor()
		a.scene = NewTitleScene(a)
	}
}

func (s *ResultScene) Draw(dst *ebiten.Image) {
	w := s.world
	dst.Fill(color.RGBA{0x14, 0x10, 0x24, 0xff})
	drawTextScaled(dst, "GAME OVER", screenW/2, 14, colRed, AlignCenter, 2)
	drawTextScaled(dst, fmt.Sprintf("享年 %.1f歳", w.Age()), screenW/2, 46, colWhite, AlignCenter, 2)
	drawText(dst, "「"+epitaph(w.Age())+"」", screenW/2, 76, colYellow, AlignCenter)
	drawText(dst, "死因："+w.Cause, screenW/2, 92, colDim, AlignCenter)

	drawPanel(dst, 14, 112, screenW-28, 108)
	y := 120.0
	line := func(l, r string) {
		drawText(dst, l, 24, y, colWhite, AlignLeft)
		drawText(dst, r, screenW-24, y, colWhite, AlignRight)
		y += 15
	}
	line("年齢", fmt.Sprintf("%.1f×100", w.Age()))
	line("実績", fmt.Sprintf("%d個×%.0f", w.Achievements, w.achievementValue()))
	line("レアイベント", fmt.Sprintf("%d", w.RarePoints))
	line("倍率", fmt.Sprintf("×%.2f", w.PointMultiplier()))
	y += 4
	drawText(dst, "人生ポイント", 24, y+4, colYellow, AlignLeft)
	drawTextScaled(dst, strconv.Itoa(w.Points()), screenW-24, y, colYellow, AlignRight, 2)

	y = 232
	switch {
	case !s.submit.Done():
		drawText(dst, "記録を送信中…", screenW/2, y, colDim, AlignCenter)
	case s.submit.Err != nil:
		drawText(dst, "オフラインのため記録されません", screenW/2, y, colDim, AlignCenter)
	default:
		r := s.submit.Value
		drawText(dst, fmt.Sprintf("全国 %d位 / %d人生", r.Rank, r.Total), screenW/2, y, colWhite, AlignCenter)
		y += 18
		if r.Unique {
			c := rarityColors[RaritySSR]
			if s.tick/8%2 == 0 {
				c = colYellow
			}
			drawText(dst, "称号『唯一無二』獲得！", screenW/2, y, c, AlignCenter)
			drawText(dst, "（この組み合わせはあなたが初めて）", screenW/2, y+14, colDim, AlignCenter)
			y += 32
		}
		drawText(dst, "あなたと同じものを引いた人", screenW/2, y, colDim, AlignCenter)
		y += 16
		for _, id := range w.Birth.Talents {
			t, _ := talentByID(id)
			drawText(dst, t.Name, 30, y, rarityColors[t.Rarity], AlignLeft)
			drawText(dst, fmt.Sprintf("%.1f%%", r.Talents[string(id)]), screenW-30, y, colWhite, AlignRight)
			y += 14
		}
		drawText(dst, w.Birth.Difficulty.Name, 30, y, rarityColors[w.Birth.Difficulty.Rarity], AlignLeft)
		drawText(dst, fmt.Sprintf("%.1f%%", r.Difficulty), screenW-30, y, colWhite, AlignRight)
		y += 14
		drawText(dst, w.Birth.Family.Name, 30, y, rarityColors[w.Birth.Family.Rarity], AlignLeft)
		drawText(dst, fmt.Sprintf("%.1f%%", r.Family), screenW-30, y, colWhite, AlignRight)
	}

	if s.tick >= 30 {
		s.again.Draw(dst)
		s.rank.Draw(dst)
		s.title.Draw(dst)
	}
}

// ---------------------------------------------------------------- ranking

type RankingScene struct {
	back      Scene
	period    string
	req       *Async[[]RankEntry]
	allBtn    *Button
	todayBtn  *Button
	returnBtn *Button
}

func NewRankingScene(a *App, back Scene) *RankingScene {
	s := &RankingScene{
		back:      back,
		allBtn:    NewButton(screenW/2-56, 40, 100, 24, "全期間", color.RGBA{0x40, 0x70, 0xd0, 0xff}),
		todayBtn:  NewButton(screenW/2+56, 40, 100, 24, "今日", color.RGBA{0x40, 0x70, 0xd0, 0xff}),
		returnBtn: NewButton(screenW/2, screenH-40, 140, 28, "もどる", color.RGBA{0x50, 0x50, 0x68, 0xff}),
	}
	s.load("all")
	return s
}

func (s *RankingScene) load(p string) {
	s.period = p
	s.req = FetchRanking(p)
}

func (s *RankingScene) Update(a *App) {
	switch {
	case a.in.TappedButton(s.allBtn):
		a.sound.Cursor()
		s.load("all")
	case a.in.TappedButton(s.todayBtn):
		a.sound.Cursor()
		s.load("today")
	case a.in.TappedButton(s.returnBtn) || a.in.Back:
		a.sound.Cursor()
		a.scene = s.back
	}
}

func (s *RankingScene) Draw(dst *ebiten.Image) {
	dst.Fill(color.RGBA{0x14, 0x18, 0x30, 0xff})
	drawTextScaled(dst, "人生ランキング", screenW/2, 8, colYellow, AlignCenter, 2)
	gray := color.RGBA{0x40, 0x40, 0x50, 0xff}
	blue := color.RGBA{0x40, 0x70, 0xd0, 0xff}
	s.allBtn.Color, s.todayBtn.Color = gray, gray
	if s.period == "all" {
		s.allBtn.Color = blue
	} else {
		s.todayBtn.Color = blue
	}
	s.allBtn.Draw(dst)
	s.todayBtn.Draw(dst)
	y := 76.0
	switch {
	case !s.req.Done():
		drawText(dst, "読み込み中…", screenW/2, y, colDim, AlignCenter)
	case s.req.Err != nil:
		drawText(dst, "ランキングを取得できませんでした", screenW/2, y, colDim, AlignCenter)
	case len(s.req.Value) == 0:
		drawText(dst, "まだ誰も生まれていません", screenW/2, y, colDim, AlignCenter)
	default:
		for i, e := range s.req.Value {
			if i >= 20 {
				break
			}
			c := colWhite
			switch i {
			case 0:
				c = colYellow
			case 1:
				c = color.RGBA{0xd0, 0xd0, 0xe0, 0xff}
			case 2:
				c = color.RGBA{0xe0, 0x90, 0x50, 0xff}
			}
			name := []rune(e.Name)
			if len(name) > 8 {
				name = append(name[:7], '…')
			}
			drawText(dst, fmt.Sprintf("%2d", i+1), 10, y, c, AlignLeft)
			drawText(dst, string(name), 32, y, c, AlignLeft)
			drawText(dst, fmt.Sprintf("%.0f歳", e.Age), 186, y, colDim, AlignRight)
			drawText(dst, strconv.Itoa(e.Points), screenW-10, y, c, AlignRight)
			y += 17
		}
	}
	s.returnBtn.Draw(dst)
}
