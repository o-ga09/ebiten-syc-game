package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Pixel art is described with strings; each byte maps to a palette color
// and '.' is transparent.

var playerArt = []string{
	"...HHHH...",
	"..HHHHHH..",
	"..HSSSSH..",
	"..SESSES..",
	"..SSSSSS..",
	"...SMMS...",
	"..CCCCCC..",
	".SCCCCCCS.",
	".SCCCCCCS.",
	"..CCCCCC..",
	"..PP..PP..",
	"..PP..PP..",
	"..KK..KK..",
}

var hatArts = map[int][]string{
	1: { // cap
		"..RRRRR...",
		".RRRRRRRR.",
	},
	2: { // ribbon
		"......RR.R",
		".......RR.",
	},
	3: { // crown
		"..Y.Y.Y...",
		"..YYYYY...",
	},
	4: { // ahoge
		".....H....",
		"....H.....",
	},
}

var enemyArt = []string{
	"..XXXXXX..",
	".XXXXXXXX.",
	"XXWWXXWWXX",
	"XXWBXXBWXX",
	"XXXXXXXXXX",
	"XXXBBBBXXX",
	".XXXXXXXX.",
	"..X.XX.X..",
}

var graveArt = []string{
	"...GGGG...",
	"..GGGGGG..",
	"..GLLLLG..",
	"..GGLLGG..",
	"..GLLLLG..",
	"..GGGGGG..",
	"..GGGGGG..",
	"DDDDDDDDDD",
}

var catArt = []string{
	"O...O.",
	"OOOOO.",
	"OEOEO.",
	"OOOOOO",
	".O..O.",
}

func makeSprite(art []string, pal map[byte]color.Color) *ebiten.Image {
	h := len(art)
	w := 0
	for _, r := range art {
		w = max(w, len(r))
	}
	img := ebiten.NewImage(w, h)
	pix := make([]byte, w*h*4)
	for y, r := range art {
		for x := 0; x < len(r); x++ {
			c, ok := pal[r[x]]
			if !ok {
				continue
			}
			cr, cg, cb, ca := c.RGBA()
			i := (y*w + x) * 4
			pix[i], pix[i+1], pix[i+2], pix[i+3] = byte(cr>>8), byte(cg>>8), byte(cb>>8), byte(ca>>8)
		}
	}
	img.WritePixels(pix)
	return img
}

// Sprites holds per-run sprite images.
type Sprites struct {
	Player *ebiten.Image
	Hat    *ebiten.Image
	Enemy  *ebiten.Image
	Grave  *ebiten.Image
	Cat    *ebiten.Image
}

func NewSprites(l Look) *Sprites {
	skin := color.RGBA{0xff, 0xd0, 0xa8, 0xff}
	hair := hairColors[l.Hair%len(hairColors)]
	cloth := clothesColors[l.Clothes%len(clothesColors)]
	pal := map[byte]color.Color{
		'H': hair, 'S': skin, 'E': colBlack, 'M': color.RGBA{0xe0, 0x70, 0x70, 0xff},
		'C': cloth, 'P': color.RGBA{0x38, 0x40, 0x68, 0xff}, 'K': color.RGBA{0x50, 0x30, 0x20, 0xff},
		'R': color.RGBA{0xff, 0x40, 0x60, 0xff}, 'Y': colYellow,
	}
	s := &Sprites{Player: makeSprite(playerArt, pal)}
	if art, ok := hatArts[l.Hat]; ok {
		s.Hat = makeSprite(art, pal)
	}
	s.Enemy = makeSprite(enemyArt, map[byte]color.Color{
		'X': color.RGBA{0xd0, 0x30, 0x40, 0xff}, 'W': colWhite, 'B': colBlack,
	})
	s.Grave = makeSprite(graveArt, map[byte]color.Color{
		'G': color.RGBA{0x90, 0x90, 0xa0, 0xff}, 'L': color.RGBA{0x60, 0x60, 0x70, 0xff},
		'D': color.RGBA{0x50, 0x80, 0x40, 0xff},
	})
	s.Cat = makeSprite(catArt, map[byte]color.Color{
		'O': color.RGBA{0xf0, 0xa0, 0x40, 0xff}, 'E': colBlack,
	})
	return s
}

func drawSprite(dst, img *ebiten.Image, x, y, sx, sy float64, cs ebiten.ColorScale) {
	op := &ebiten.DrawImageOptions{}
	w, h := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	op.GeoM.Translate(-w/2, -h)
	op.GeoM.Scale(sx, sy)
	op.GeoM.Translate(float64(int(x)), float64(int(y)))
	op.ColorScale = cs
	dst.DrawImage(img, op)
}
