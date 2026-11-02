package main

import (
	"image"
	"image/color"
	"slices"

	"github.com/hajimehoshi/bitmapfont/v4"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	face = text.NewGoXFace(bitmapfont.Face)

	colWhite  = color.RGBA{0xff, 0xff, 0xff, 0xff}
	colBlack  = color.RGBA{0x10, 0x10, 0x18, 0xff}
	colDim    = color.RGBA{0x90, 0x90, 0xa0, 0xff}
	colYellow = color.RGBA{0xff, 0xe0, 0x40, 0xff}
	colRed    = color.RGBA{0xff, 0x50, 0x50, 0xff}
	colPanel  = color.RGBA{0x10, 0x10, 0x20, 0xd8}
)

type Align int

const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)

// lineH is the height of one text line with bitmapfont.
const lineH = 12

// drawText draws text with a 1px dark outline so it reads on any background.
func drawText(dst *ebiten.Image, s string, x, y float64, clr color.Color, align Align) {
	drawTextScaled(dst, s, x, y, clr, align, 1)
}

func drawTextScaled(dst *ebiten.Image, s string, x, y float64, clr color.Color, align Align, scale float64) {
	w := textWidth(s) * scale
	switch align {
	case AlignCenter:
		x -= w / 2
	case AlignRight:
		x -= w
	}
	x, y = float64(int(x)), float64(int(y))
	draw := func(dx, dy float64, c color.Color) {
		op := &text.DrawOptions{}
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(x+dx, y+dy)
		op.ColorScale.ScaleWithColor(c)
		text.Draw(dst, s, face, op)
	}
	o := scale
	for _, d := range [][2]float64{{-o, 0}, {o, 0}, {0, -o}, {0, o}, {o, o}} {
		draw(d[0], d[1], colBlack)
	}
	draw(0, 0, clr)
}

func textWidth(s string) float64 {
	w, _ := text.Measure(s, face, lineH)
	return w
}

func fillRect(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	vector.FillRect(dst, float32(x), float32(y), float32(w), float32(h), c, false)
}

func strokeRect(dst *ebiten.Image, x, y, w, h float64, c color.Color) {
	vector.StrokeRect(dst, float32(x)+0.5, float32(y)+0.5, float32(w)-1, float32(h)-1, 1, c, false)
}

// Button is a tappable rectangle.
type Button struct {
	Rect  image.Rectangle
	Label string
	Color color.RGBA
}

func NewButton(cx, y, w, h int, label string, c color.RGBA) *Button {
	return &Button{Rect: image.Rect(cx-w/2, y, cx+w/2, y+h), Label: label, Color: c}
}

func (b *Button) Hit(p image.Point) bool { return p.In(b.Rect) }

func (b *Button) Draw(dst *ebiten.Image) {
	r := b.Rect
	x, y, w, h := float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy())
	fillRect(dst, x, y+2, w, h, colBlack)
	fillRect(dst, x, y, w, h, b.Color)
	fillRect(dst, x, y, w, 2, color.RGBA{0xff, 0xff, 0xff, 0x60})
	strokeRect(dst, x, y, w, h, colBlack)
	drawText(dst, b.Label, x+w/2, y+(h-lineH)/2, colWhite, AlignCenter)
}

func drawPanel(dst *ebiten.Image, x, y, w, h float64) {
	fillRect(dst, x, y, w, h, colPanel)
	strokeRect(dst, x, y, w, h, color.RGBA{0xff, 0xff, 0xff, 0x80})
}

// Input collects the per-frame input in screen coordinates.
type Input struct {
	Taps  []image.Point
	Left  bool
	Right bool
	OK    bool
	Back  bool
	Reset bool

	touchIDs []ebiten.TouchID
}

func (in *Input) Poll() {
	in.Taps = in.Taps[:0]
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		in.Taps = append(in.Taps, image.Pt(x, y))
	}
	in.touchIDs = inpututil.AppendJustPressedTouchIDs(in.touchIDs[:0])
	for _, id := range in.touchIDs {
		x, y := ebiten.TouchPosition(id)
		in.Taps = append(in.Taps, image.Pt(x, y))
	}
	in.Left = slices.ContainsFunc([]ebiten.Key{ebiten.KeyArrowLeft, ebiten.KeyA}, inpututil.IsKeyJustPressed)
	in.Right = slices.ContainsFunc([]ebiten.Key{ebiten.KeyArrowRight, ebiten.KeyD}, inpututil.IsKeyJustPressed)
	in.OK = slices.ContainsFunc([]ebiten.Key{ebiten.KeySpace, ebiten.KeyEnter}, inpututil.IsKeyJustPressed)
	in.Back = inpututil.IsKeyJustPressed(ebiten.KeyEscape)
	in.Reset = inpututil.IsKeyJustPressed(ebiten.KeyR)
}

// Tapped reports whether anything was tapped or the OK key pressed.
func (in *Input) Tapped() bool { return len(in.Taps) > 0 || in.OK }

// TappedButton reports whether b was tapped this frame.
func (in *Input) TappedButton(b *Button) bool {
	for _, p := range in.Taps {
		if b.Hit(p) {
			return true
		}
	}
	return false
}
