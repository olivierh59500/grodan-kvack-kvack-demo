// Package mobile exposes the demo to ebitenmobile.
package mobile

import (
	enginemobile "github.com/hajimehoshi/ebiten/v2/mobile"
	demo "grodan-kvack-kvack-demo/dck"
)

func init() {
	enginemobile.SetGame(demo.NewGame())
}

// Dummy forces gomobile to include this package in the Android binding.
func Dummy() {}
