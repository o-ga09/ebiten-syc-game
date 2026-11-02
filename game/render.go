package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const platformW = 38

func rowScreenY(w *World, r float64) float64 {
	return playerRowY - (r-w.VisRow)*rowH
}

func lighten(c color.RGBA, f float64) color.RGBA {
	l := func(v uint8) uint8 { return uint8(math.Min(255, float64(v)+(255-float64(v))*f)) }
	return color.RGBA{l(c.R), l(c.G), l(c.B), c.A}
}

func darken(c color.RGBA, f float64) color.RGBA {
	d := func(v uint8) uint8 { return uint8(float64(v) * (1 - f)) }
	return color.RGBA{d(c.R), d(c.G), d(c.B), c.A}
}

func drawBackground(dst *ebiten.Image, w *World) {
	st := w.Stage()
	dst.Fill(st.Sky)
	// Parallax pixel clouds / stars.
	dark := st.Sky.R < 0x80
	for i := range 9 {
		speed := 0.25 + float64(i%3)*0.1
		y := math.Mod(float64(i)*83+w.VisRow*rowH*speed, screenH+60) - 30
		x := math.Mod(float64(i)*131+float64(w.Tick)*0.05*float64(i%2+1), screenW+60) - 30
		if dark {
			fillRect(dst, x, y, 2, 2, color.RGBA{0xff, 0xff, 0xe0, 0xc0})
			fillRect(dst, x+40, y+20, 1, 1, color.RGBA{0xff, 0xff, 0xff, 0xa0})
		} else {
			c := lighten(st.Sky, 0.55)
			fillRect(dst, x, y, 28, 6, c)
			fillRect(dst, x+6, y-4, 14, 4, c)
		}
	}
	// Faint lane guides.
	for _, lx := range laneX {
		fillRect(dst, lx-1, 0, 2, screenH, color.RGBA{0, 0, 0, 0x10})
	}
}

func drawPlatform(dst *ebiten.Image, w *World, s Slot, x, y float64) {
	st := w.Stage()
	base := st.Ground
	x0 := x - platformW/2
	if s.Bug != BugNone {
		// Glitchy: slices jitter and colors flicker a bit.
		if w.Birth.Has(TDebugger) {
			base = color.RGBA{0xe0, 0x30, 0x30, 0xff}
		}
		for i := 0; i < platformW; i += 6 {
			jit := 0.0
			if (w.Tick/4+i)%7 == 0 {
				jit = float64((w.Tick+i)%3 - 1)
			}
			c := base
			if (w.Tick/6+i)%11 == 0 {
				c = color.RGBA{0xff, 0x00, 0xff, 0xff}
			}
			fillRect(dst, x0+float64(i), y+jit, 6, 7, c)
		}
		fillRect(dst, x0, y, platformW, 2, lighten(base, 0.4))
		return
	}
	fillRect(dst, x0, y+1, platformW, 7, darken(base, 0.35))
	fillRect(dst, x0, y, platformW, 6, base)
	fillRect(dst, x0, y, platformW, 2, lighten(base, 0.45))
	for i := 4; i < platformW-2; i += 9 {
		fillRect(dst, x0+float64(i), y+3, 2, 2, darken(base, 0.2))
	}
}

func drawWorld(dst *ebiten.Image, w *World, sp *Sprites) {
	drawBackground(dst, w)

	lo := int(math.Floor(w.VisRow)) - 8
	hi := int(math.Ceil(w.VisRow)) + 18

	// Graves of other players.
	for _, g := range w.Graves {
		r := g.Age * rowsPerYear
		if r < float64(lo) || r > float64(hi) {
			continue
		}
		y := rowScreenY(w, r) + 6
		x := 26.0
		if g.Lane == 1 {
			x = screenW - 26
		}
		drawSprite(dst, sp.Grave, x, y, 1, 1, ebiten.ColorScale{})
		name := []rune(g.Name)
		if len(name) > 5 {
			name = append(name[:4], '…')
		}
		drawText(dst, string(name), x, y-24-lineH, colWhite, AlignCenter)
		drawText(dst, fmt.Sprintf("%.0f歳", g.Age), x, y-22, colDim, AlignCenter)
	}

	// Platforms.
	for r := max(lo, 0); r <= hi; r++ {
		row := w.Row(r)
		y := rowScreenY(w, float64(r))
		hidden := row.NoRoute && r > w.PlayerRow+w.VisibleAhead()
		for lane, s := range row.Slots {
			x := laneX[lane]
			if hidden {
				strokeRect(dst, x-platformW/2, y, platformW, 6, color.RGBA{0xff, 0xff, 0xff, 0x50})
				drawText(dst, "?", x, y-3, color.RGBA{0xff, 0xff, 0xff, 0x90}, AlignCenter)
				continue
			}
			if !s.Exists {
				continue
			}
			drawPlatform(dst, w, s, x, y)
			if s.Item != ItemNone && !s.Collected {
				bob := math.Sin(float64(w.Tick)/10+float64(r)) * 2
				switch s.Item {
				case ItemAchievement:
					drawText(dst, "★", x, y-lineH-3+bob, colYellow, AlignCenter)
				case ItemRare:
					c := rarityColors[RaritySSR]
					if w.Tick/8%2 == 0 {
						c = colWhite
					}
					drawText(dst, "◆", x, y-lineH-3+bob, c, AlignCenter)
				}
			}
		}
		if row.NoRoute && r == w.PlayerRow+w.VisibleAhead()+1 {
			drawText(dst, "ルートがありません", screenW/2, y-lineH-14, colRed, AlignCenter)
		}
	}

	// Enemies.
	for _, e := range w.Enemies {
		y := rowScreenY(w, float64(e.Row)) - 1
		bob := math.Sin(float64(w.Tick)/6+e.X/20) * 1.5
		sx := 1.0
		if e.VX < 0 {
			sx = -1
		}
		drawSprite(dst, sp.Enemy, e.X, y+bob, sx, 1, ebiten.ColorScale{})
		drawText(dst, e.Name, e.X, y-10-lineH+bob, colWhite, AlignCenter)
	}

	// Cat follower.
	if w.Birth.Has(TCatLover) {
		cx := w.VisX - 14
		if w.PlayerLane == 0 {
			cx = w.VisX + 14
		}
		drawSprite(dst, sp.Cat, cx, playerRowY+math.Abs(math.Sin(float64(w.Tick)/5))*-2, 1, 1, ebiten.ColorScale{})
	}

	// Player.
	if !(w.Invincible > 0 && w.Tick/3%2 == 0) {
		sy := 1.0
		if w.Birth.Has(TTall) {
			sy = 1.35
		}
		var cs ebiten.ColorScale
		if w.Over {
			cs.Scale(0.5, 0.5, 0.6, 1)
		}
		py := playerRowY - w.JumpHop() + w.FallOffset()
		drawSprite(dst, sp.Player, w.VisX, py, 1, sy, cs)
		if sp.Hat != nil {
			drawSprite(dst, sp.Hat, w.VisX, py-float64(len(playerArt))*sy+2, 1, 1, cs)
		}
	}

	// Particles.
	for _, p := range w.Particles {
		a := uint8(255 * p.Life / p.Max)
		c := p.Color
		c.A = a
		c.R = uint8(float64(c.R) * float64(a) / 255)
		c.G = uint8(float64(c.G) * float64(a) / 255)
		c.B = uint8(float64(c.B) * float64(a) / 255)
		y := rowScreenY(w, p.Y/rowH)
		fillRect(dst, p.X-1, y-1, 2, 2, c)
	}

	drawTimeWave(dst, w)
}

func drawTimeWave(dst *ebiten.Image, w *World) {
	top := rowScreenY(w, w.TimeRow) + 8
	if top > screenH {
		return
	}
	body := color.RGBA{0x30, 0x10, 0x50, 0xe8}
	crest := color.RGBA{0xa0, 0x60, 0xff, 0xff}
	for x := 0; x < screenW; x += 3 {
		h := math.Sin(float64(x)/14+float64(w.Tick)/9)*3 + math.Sin(float64(x)/5-float64(w.Tick)/5)*1.5
		y := top + h
		fillRect(dst, float64(x), y, 3, screenH-y, body)
		fillRect(dst, float64(x), y, 3, 2, crest)
	}
	label := "～～ 時 間 ～～"
	drawText(dst, label, screenW/2, top+8, color.RGBA{0xe0, 0xc0, 0xff, 0xff}, AlignCenter)
}
