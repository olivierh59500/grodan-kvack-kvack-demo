package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	demo "grodan-kvack-kvack-demo"
)

func run() error {
	ebiten.SetWindowSize(640, 400)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("Grodan and Kvack Kvack Demo")
	ebiten.SetScreenClearedEveryFrame(false)

	game := demo.NewGame()
	defer game.Cleanup()
	return ebiten.RunGame(newDrawOnUpdateGame(game))
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
