// Package grodan contains the platform-independent demo.
package grodan

import (
	"bytes"
	"fmt"
	"github.com/olivierh59500/democonstructionkit/presets"
	originalassets "grodan-kvack-kvack-demo"
	"image"
	"image/color"

	kit "github.com/olivierh59500/democonstructionkit"
	"github.com/olivierh59500/democonstructionkit/composite"
	"github.com/olivierh59500/democonstructionkit/motion"
	"github.com/olivierh59500/democonstructionkit/scrolling"
	"github.com/olivierh59500/democonstructionkit/sound"
	"github.com/olivierh59500/democonstructionkit/sprites"
	"grodan-kvack-kvack-demo/dck/internal/textdata"

	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	audio "github.com/olivierh59500/democonstructionkit/sound/output"
)

const (
	screenWidth  = 640
	screenHeight = 400
	sampleRate   = 48000
	spriteCount  = 12
	spriteWidth  = 16
	spriteHeight = 10
	spriteStride = 17
)

// Embedded assets
var (
	bgGreenData = originalassets.DCKAssetBgGreenData()

	bgPinkData = originalassets.DCKAssetBgPinkData()

	upRasterData = originalassets.DCKAssetUpRasterData()

	bsRasterData = originalassets.DCKAssetBsRasterData()

	spriteData = originalassets.DCKAssetSpriteData()

	bsFontData = originalassets.DCKAssetBsFontData()

	upFontData = originalassets.DCKAssetUpFontData()

	lFontData = originalassets.DCKAssetLFontData()

	musicData = originalassets.DCKAssetMusicData()
)

// Game represents the game state
type Game struct {
	// Images
	bgGreen           *ebiten.Image
	bgPink            *ebiten.Image
	greenBackground   *composite.Background
	pinkBackground    *composite.Background
	upScrollRaster    *ebiten.Image
	bigScrollRaster   *ebiten.Image
	smallRasterTop    *ebiten.Image
	smallRasterBottom *ebiten.Image
	sprite            *ebiten.Image
	bsFont            *ebiten.Image
	upFont            *ebiten.Image
	lFont             *ebiten.Image
	spriteGroup       *sprites.Group

	// Canvases
	bs2Canvas   *ebiten.Image
	upCanvas    *ebiten.Image
	smallCanvas *ebiten.Image

	// Animation state
	moveY    float64
	howmuchY float64
	moveX    float64
	howmuchX float64
	bgcount  float64

	Y   float64
	hY  float64
	X   float64
	gox float64

	// Independent ribbons share the same DCK renderer and transport API.
	scrolls [4]*scrolling.Scrolling

	// Audio
	audioContext     *audio.Context
	audioPlayer      *audio.Player
	musicStream      *sound.Stream
	audioInitialized bool
}

// NewGame creates a new game instance
func NewGame() *Game {
	g := &Game{
		moveY:    0,
		howmuchY: 1,
		moveX:    0,
		howmuchX: 1,
		bgcount:  0,
		Y:        0,
		hY:       1,
		X:        0,
		gox:      0,
	}

	// Load images
	g.loadImages()
	background := composite.BackgroundConfig{PeriodX: screenWidth, PeriodY: screenHeight, CopiesX: 3, CopiesY: 2}
	var err error
	g.greenBackground, err = composite.NewBackground(background)
	if err != nil {
		panic(err)
	}
	g.pinkBackground, err = composite.NewBackground(background)
	if err != nil {
		panic(err)
	}
	frames := make([]*ebiten.Image, spriteCount)
	for i := range frames {
		srcX := i * spriteStride
		rect := image.Rect(srcX, 0, srcX+spriteWidth, spriteHeight)
		frames[i] = g.sprite.SubImage(rect).(*ebiten.Image)
	}
	formation, err := motion.NewHarmonicFormation(presets.GrodanSpriteFormationConfig())
	if err != nil {
		panic(err)
	}
	g.spriteGroup, err = sprites.NewGroup(sprites.GroupConfig{
		Frames: frames, Count: spriteCount, FrameStride: 1, ScaleX: 2, ScaleY: 2,
		Harmonic: formation, HarmonicClockStep: [2]float64{.02, .03},
		HarmonicEnvelope: &motion.BounceBankConfig{Start: []float64{0}, Velocity: []float64{.1}, Min: -50, Max: 50},
	})
	if err != nil {
		panic(err)
	}

	// Create canvases
	g.bs2Canvas = newRenderTarget(640, 200)
	g.upCanvas = newRenderTarget(32, 400)
	g.smallCanvas = newRenderTarget(320, 32)

	// Initialize scroll texts
	g.initScrollTexts()

	return g
}

func newRenderTarget(width, height int) *ebiten.Image {
	return ebiten.NewImageWithOptions(
		image.Rect(0, 0, width, height),
		&ebiten.NewImageOptions{Unmanaged: true},
	)
}

// loadImages loads all image assets
func (g *Game) loadImages() {
	g.bgGreen = mustLoadImage("Grodan_green.png", bgGreenData)
	g.bgPink = mustLoadImage("Grodan_pink.png", bgPinkData)
	upRaster := mustLoadImage("upscrollraster.png", upRasterData)
	g.upScrollRaster = upRaster.SubImage(image.Rect(0, 0, 16, 200)).(*ebiten.Image)
	g.smallRasterTop = upRaster.SubImage(image.Rect(0, 16, 160, 20)).(*ebiten.Image)
	g.smallRasterBottom = upRaster.SubImage(image.Rect(0, 64, 160, 68)).(*ebiten.Image)
	bigRaster := mustLoadImage("bigscrollraster.png", bsRasterData)
	g.bigScrollRaster = bigRaster.SubImage(image.Rect(0, 0, 160, 96)).(*ebiten.Image)
	g.sprite = mustLoadImage("sprite.png", spriteData)
	g.bsFont = mustLoadImage("bsfont.png", bsFontData)
	g.upFont = mustLoadImage("upfonts.png", upFontData)
	g.lFont = mustLoadImage("lfont.png", lFontData)
}

func mustLoadImage(name string, data []byte) *ebiten.Image {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		panic(fmt.Errorf("decode embedded image %s: %w", name, err))
	}
	return ebiten.NewImageFromImage(img)
}

// initScrollTexts initializes the scrolling texts
func (g *Game) initScrollTexts() {
	// Initialize font maps
	bsFontMap, err := presets.FontAtlas("grodan-kvack-kvack-demo", g.bsFont)
	if err != nil {
		panic(err)
	}
	upFontMap, err := presets.FontAtlas("grodan-up", g.upFont)
	if err != nil {
		panic(err)
	}
	lFontMap, err := presets.FontAtlas("grodan-small", g.lFont)
	if err != nil {
		panic(err)
	}

	texts := textdata.Messages()
	recipes := presets.GrodanRibbons(texts,
		bsFontMap, upFontMap, lFontMap)
	for i := range g.scrolls {
		g.scrolls[i], err = scrolling.New(scrolling.Config{Ribbon: &recipes[i]})
		if err != nil {
			panic(err)
		}
	}
}

// initAudio initializes the audio system
func (g *Game) initAudio() {
	g.audioContext = audio.NewContext(sampleRate)

	var err error
	g.musicStream, err = sound.Open("music.ym", musicData, sound.Options{SampleRate: sampleRate, Loop: true, PCMFormat: sound.Float32, Gain: 0.5, Quantize16: true})
	if err != nil {
		log.Printf("Failed to open music: %v", err)
		return
	}

	g.audioPlayer, err = g.audioContext.NewPlayerF32(g.musicStream)
	if err != nil {
		log.Printf("Failed to create audio player: %v", err)
		if closeErr := g.musicStream.Close(); closeErr != nil {
			log.Printf("Failed to close music stream: %v", closeErr)
		}
		g.musicStream = nil
		return
	}

	g.audioPlayer.Play()
}

// Update updates the game state
func (g *Game) Update() error {
	// On Android, NewGame runs before the Activity has installed its context.
	// Opening the audio device here avoids blocking the application startup.
	if !g.audioInitialized {
		g.audioInitialized = true
		g.initAudio()
	}

	// Update background 1 animation
	g.bgcount += 0.1

	if g.moveY < -400 {
		g.howmuchY = 1
	}
	if g.moveY > 0 {
		g.howmuchY = -1
	}
	g.moveY += g.howmuchY

	if g.bgcount > 10 {
		if g.moveX < -640*2 {
			g.howmuchX = 16
		}
		if g.moveX > 0 {
			g.howmuchX = -16
		}
		g.moveX += g.howmuchX
	}

	if g.bgcount > 20 {
		g.bgcount = 0
	}

	// Update background 2 animation
	if g.Y < -400 {
		g.hY = 2
		g.gox = 16
	}
	if g.Y > 0 {
		g.hY = -2
		g.gox = -16
	}

	g.X += g.gox
	if g.X < -710 {
		g.X = -710
	}
	if g.X > 0 {
		g.X = 0
	}
	g.Y += g.hY

	if err := g.spriteGroup.Update(kit.Frame{}); err != nil {
		return err
	}

	for _, index := range [...]int{0, 2, 3, 1} {
		if g.scrolls[index] != nil {
			if err := g.scrolls[index].Update(kit.Frame{}); err != nil {
				return err
			}
		}
	}

	return nil
}

// Draw draws the game
func (g *Game) Draw(screen *ebiten.Image) {
	// Clear screen
	screen.Fill(color.Black)

	g.greenBackground.DrawAt(screen, g.bgGreen, g.moveX, g.moveY)
	g.pinkBackground.DrawAt(screen, g.bgPink, g.X, g.Y)

	// Draw sprites
	g.spriteGroup.Draw(screen)

	// Draw big scroll
	g.drawBigScroll(screen)

	// Draw up scroll
	g.drawUpScroll(screen)

	// Draw small scrolls
	g.drawSmallScrolls(screen)
}

// drawBigScroll draws the big scrolling text
func (g *Game) drawBigScroll(screen *ebiten.Image) {
	if g.scrolls[0] == nil {
		return
	}

	// Clear canvas
	g.bs2Canvas.Clear()

	// Draw the glyphs at their final size, without an intermediate canvas.
	g.scrolls[0].Draw(g.bs2Canvas)

	// Apply raster effect
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(4, 2)
	op.Blend = ebiten.BlendSourceAtop
	composite.Instance{Image: g.bigScrollRaster, Options: *op}.Draw(g.bs2Canvas)

	// Draw to screen
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 200)
	screen.DrawImage(g.bs2Canvas, op)
}

// drawUpScroll draws the vertical scrolling text
func (g *Game) drawUpScroll(screen *ebiten.Image) {
	if g.scrolls[1] == nil {
		return
	}

	// Clear canvas
	g.upCanvas.Clear()

	// Draw vertical scroll text
	g.scrolls[1].Draw(g.upCanvas)

	// Apply raster effect
	maskOp := &ebiten.DrawImageOptions{}
	maskOp.GeoM.Scale(2, 2)
	maskOp.Blend = ebiten.BlendSourceAtop
	composite.Instance{Image: g.upScrollRaster, Options: *maskOp}.Draw(g.upCanvas)

	// Draw to screen at multiple positions
	positions := [...]float64{0, 64, 128, 480, 544, 608}
	for _, x := range positions {
		drawOp := &ebiten.DrawImageOptions{}
		drawOp.GeoM.Translate(x, 0)
		screen.DrawImage(g.upCanvas, drawOp)
	}
}

// drawSmallScrolls draws the small scrolling texts
func (g *Game) drawSmallScrolls(screen *ebiten.Image) {
	if g.scrolls[2] == nil || g.scrolls[3] == nil {
		return
	}

	// Both scrolls share one render target. This lets Ebitengine batch their
	// glyphs and masks before a single final draw to the screen.
	g.smallCanvas.Clear()
	g.scrolls[2].Draw(g.smallCanvas)
	g.scrolls[3].Draw(g.smallCanvas)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	op.Blend = ebiten.BlendSourceAtop
	composite.Instance{Image: g.smallRasterTop, Options: *op}.Draw(g.smallCanvas)

	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(0, 24)
	op.Blend = ebiten.BlendSourceAtop
	composite.Instance{Image: g.smallRasterBottom, Options: *op}.Draw(g.smallCanvas)

	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(0, 16)
	screen.DrawImage(g.smallCanvas, op)
}

// Layout returns the screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// Cleanup releases resources
func (g *Game) Cleanup() {
	for i, scroll := range g.scrolls {
		if scroll != nil {
			_ = scroll.Close()
			g.scrolls[i] = nil
		}
	}
	if g.audioPlayer != nil {
		if err := g.audioPlayer.Close(); err != nil {
			log.Printf("Failed to close audio player: %v", err)
		}
		g.audioPlayer = nil
	}
	if g.musicStream != nil {
		if err := g.musicStream.Close(); err != nil {
			log.Printf("Failed to close music stream: %v", err)
		}
		g.musicStream = nil
	}
}
