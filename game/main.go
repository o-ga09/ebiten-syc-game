// Command game is the Ebitengine browser game "じんせいGAME（仮）".
package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenW = 270
	screenH = 480
)

func main() {
	ebiten.SetWindowSize(screenW*2, screenH*2)
	ebiten.SetWindowTitle("じんせいGAME（仮）")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(NewApp()); err != nil {
		log.Fatal(err)
	}
}
