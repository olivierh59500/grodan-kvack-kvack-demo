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

	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	audio "github.com/olivierh59500/democonstructionkit/sound/output"
)

const (
	screenWidth     = 640
	screenHeight    = 400
	sampleRate      = 48000
	spriteCount     = 12
	spriteWidth     = 16
	spriteHeight    = 10
	spriteStride    = 17
	bigScrollXScale = 8
	bigScrollYScale = 6
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

// ScrollText manages scrolling text

type ScrollText struct {
	renderer      *scrolling.Scrolling
	glyphs        []scrolling.Glyph
	scrollX       float64
	speed         float64
	contentLength float64
	vertical      bool // For vertical scrolling
}

// NewScrollText creates a new scrolling text
func NewScrollText(text string, atlas *scrolling.Atlas, speed float64, vertical bool) *ScrollText {
	glyphs := atlas.Layout(text, scrolling.AtlasText{Vertical: vertical, SkipMissing: !vertical})
	renderer, err := scrolling.New(scrolling.Config{Glyphs: glyphs, Vertical: vertical})
	if err != nil {
		panic(err)
	}
	return &ScrollText{renderer: renderer, glyphs: glyphs, speed: speed, vertical: vertical, contentLength: renderer.Length()}
}

// Update updates the scroll position
func (s *ScrollText) Update() {
	if s.vertical {
		s.scrollX += s.speed // Move up (positive direction)
		// For vertical scroll, reset when text has completely scrolled off top
		if s.scrollX > s.contentLength+screenHeight {
			s.scrollX = -100 // Start from below screen
		}
	} else {
		s.scrollX -= s.speed
		if s.scrollX < -s.contentLength {
			s.scrollX = float64(screenWidth)
		}
	}
}

// Draw draws the scrolling text
func (s *ScrollText) Draw(dst *ebiten.Image, y, scaleX, scaleY float64) {
	state := scrolling.IdentityState()
	state.ScaleX = scaleX
	state.ScaleY = scaleY
	if s.vertical {
		state.Y = float64(dst.Bounds().Dy()) - s.scrollX*scaleY
		state.Map = func(g scrolling.Sample, op *ebiten.DrawImageOptions) bool {
			return g.Y+g.Glyph.Advance*scaleY > 0 && g.Y < float64(dst.Bounds().Dy())
		}
	} else {
		state.X = s.scrollX * scaleX
		state.Y = y * scaleY
		state.Map = func(g scrolling.Sample, op *ebiten.DrawImageOptions) bool {
			return g.X+g.Glyph.Advance*scaleX > 0 && g.X < float64(dst.Bounds().Dx())
		}
	}
	s.renderer.DrawAt(dst, state)
}

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

	// Scroll texts
	scrollText1 *ScrollText
	scrollText2 *ScrollText
	scrollText3 *ScrollText
	scrollText4 *ScrollText

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

	// Main scroll text
	mainText := "                                 HI AND WELCOME TO THE GRODAN AND KVACK KVACK DEMO (THAT NAME WILL PROBABLY MAKE US FAMOUS IN THE GUINNESS BOOK OF RECORDS - THE MOST STUPID NAME IN DEMO HISTORY.  THE PREVIOUS POSSESSORS OF THAT RECORD WAS OMEGA WITH -OMEGAKUL-.   I'M AFRAID WE WILL SOON BE BEATEN BY SYNC'S 'MJOFFE-DEMO', WITH TWO DOTS ABOVE THE 'O'.  DID YOU KNOW THAT THIS IS A COMMENT IN THE MIDDLE OF A SENTENCE? NO?  WE ALSO FORGOT, BUT LET'S CONTINUE WITH WHAT WE WERE WRITING BEFORE WE STARTED WRITING THIS RECORD-CRAP.), CODED BY NICK AND JAS OF THE CAREBEARS. GRAPHIXXXX BY TANIS, THE GREAT (?) OF THE MEGAMIGHTY CAREBEARS.        WE HAVE TO COVER TWO SUBJECTS IN THIS SCROLLTEXT - THE FANTASTIC WORLD OF HARDWARESCROLLERS  AND  GREETINGS....   LET'S START WITH THE STUFF YOU PROBABLY WANT US TO TALK THE MOST ABOUT - HARDWARESCROLLERS....        TIME: LATE MARCH 1989    PLACE: NICK'S COMPUTER ROOM     IT WORKS!!!!!!!  AFTER HAVING TRIED THE ZANY SCROLLTECHNIQUE ON BOTH NICK'S AND JAS' COMPUTERS, WE CONCLUDED THAT IT ACTUALLY WORKED.    ONE DAY LATER, OMEGA CALLS US AND GOES SOMETHING LIKE THIS: - HAAAA HAAAA  WE KNOW HOW TO SCROLL THE WHOLE SCREEN BOTH HORIZONTALLY AND VERTICALLY IN LESS THAN TEN SCANLINES!!!!!!         WE WERE AMAZED THAT THEY HAD ACTUALLY COME UP WITH THE SAME IDEA ON THE SAME DAY AS US, BUT AT LEAST NOBODY ELSE KNEW HOW TO DO IT.     WE MANAGED TO RELEASE THE FIRST HARDWARESCROLLER THE WORLD HAS SEEN, IN THE CUDDLY DEMOS, AND NOW WE ARE GOING TO USE IT COMERCIALLY (CODING GAMES, DICKHEAD)....     NOW A HINT HOW IT'S DONE:    IT HAS NOTHING TO DO WITH ANY OF THE SOUND-REGISTERS.....         HERE IS ANOTHER ADDRESS TO THE CAREBEARS:     T H E   C A R E B E A R S ,    D R A K E N B E R G S G   2 3    8 T R ,      1 1 7   4  1   S T O  C K H O L M ,     S W E  D E N .                NOW FOR SOME GREETINGS:   MEGADUNDERSUPERDUPERGREETINGS TO  ALL THE OTHER MEMBERS OF THE UNION, ESPECIALLY THE EXCEPTIONS (TANIS WISH TO GIVE A SPECIAL HI TO ES) AND THE REPLICANTS (GOODBYE, RATBOY! YOUR INTROS WERE GREAT).   NORMAL MEGAGREETINGS (IN MERIT-ORDER)(WOW) TO   SYNC (WE'VE CHANGED OUR MINDS, YOU'RE THE SECOND BEST SWEDISH CREW. WE JUST HADN'T SEEN MANY SCREENS BY YOU GUYS (IT'S UNDERSTANDABLE - YOU HAVE ONLY RELEASED THREE NOT VERY GOOD ONES)),  OMEGA (TOO BAD, YOU'RE NOT THE SECOND BEST ANYMORE.  PERHAPS IT HAS SOMETHING TO DO WITH  THE TERA-DISTER, THE 'TCB-E'-JATTEDUMMA'-SIGN OR THE FACT THAT SYNC IS BETTER), THE LOST BOYS (SEE YA' SOON AND WE'RE ANXIOUSLY AWAITING YOUR MEGAMEGADEMO)             SOMETHING BETWEEN MEGAGREETINGS AND NORMAL GREETINGS TO:   FLEXIBLE FRONT (GOODBYE), VECTOR (SO YOU CRACKED OUR DEMO, HUH? NICE SCREEN, BY THE WAY), GHOST (SO YOU TRIED TO CRACK OUR DEMO, HUH? GREAT SCREEN, BY THE WAY), 2 LIFE CREW (YOU ARE IMPROVING), MAGNUM FORCE (YOU SEEM TO BE THE BEST OPTIMIZERS IN FRANCE!), NORDIK CODERS (NICE SCREEN).   NORMAL GREETINGS TO:  FASHION (GOOD LUCK WITH YOUR DEMO), OVERLANDERS (THANKS FOR NOT INCLUDING CUDDLY IN YOUR DEMOBREAKER), NO CREW (ESPECIALLY ROCCO. YOU ARE IMPROVING), AUTOMATION (GREAT COMPACT DISKS), MEDWAY BOYS (NICE CD'S),  ST CONNEXION (HOPE YOUR DEMO WILL BE AS GOOD AS YOUR GRAPHICS), FOXX (COOL SCREEN), FOFT (KEEP ON COMPACTING), ZAE (WE HAD A GREAT TIME IN MARSEILLE), KREATORS (ESPECIALLY CHUD), M.A.R.K.U.S (PLEASE SPREAD THIS DEMO AS MUCH AS YOU SPREAD CUDDLY DEMOS), HACKATARIMAN (THANKS FOR ALL THE STUFF), THE ALLIANCE (ESPECIALLY OVERLANDERS (THANKS FOR TCB-FRIENDLY SCROLLTEXTS AND MANY NICE SCREENS), AND BLACK MONOLITH TEAM (YOUR DEMOSCREEN WAS THE BEST IN THE OLD ALLIANCE DEMO), BIRDY (SEND US YOUR CRACKS), LINKAN 'THE LINK' 'JUDGE LINK' LINKSSON (PING-PONG), NYARLOTHATEPS ADEPTS (STRANGE NAME, STRANGE GUYS), GROWTWIG ( NO COMMENT),  TONY KOLLBERG (TJENA, LYCKA TILL MED ASSEMBLERN)     END OF GREETINGS. IF YOU WERE NOT GREETED, TOO BAD. NORMAL FUCKING GREETINGS TO:  CONSTELLATIONS (NOONE WILL EVER COMPLAIN ABOUT TCB AND GET AWAY WITH IT, BESIDES YOUR DEMO WAS WORTHLESS). MEGA FUCKING GREETINGS TO:     MENACING CRACKING ALLIANCE (SO, YOU DON'T LIKE BEING CALLED LAMERS, HOW YA' LIKE BEING CALLED:       MOTHERFUCKIN'   BLEEDIN' (BRITTISH ENGLISH) ULTIMATE CHICKENBRAINS????!!!! I BET IT'S ALMOST AS FUN AS FUCKING GREET TCB).  END OF SCROLLTEXT. LET'S WRAP."

	// Vertical scroll text
	vertText := "                           TANIS, THE FAMOUS GRAFIXX-MAN, IS A NEW MEMBER OF TCB.  HE MADE ALL THE GRAPHICS IN THIS SCREEN PLUS LOTSA LOGOS IN THE MAIN MENU.  WE AGREE THAT THIS 'ONE-BIT-PLANE-MANIA' DOESN'T LOOK VERY GOOD, BUT IT HAD TO BE DONE BY SOMEONE........   BAD LUCK FOR TANIS THAT WE WON'T MAKE MORE DEMOS, THOUGH....       9 9 9 9 9 9 9 9 9 9 9 9 9 9 9 9 9 9 9  ..................                 LET'S WRAP (WE SPELLED IT CORRECTLY!!!).......   "

	// Small scroll texts
	smallText1 := "                                                        ONCE UPON A TIME, WHEN THE JUNK DEMO WAS ALMOST FINISHED - WHEN THE BEST DEMO ON THE ST-MARKET WAS 'LCD' BY TEX, WE VISITED IQ2-CREW (AMIGA-FREAKS). THEY SHOWED US A COUPLE OF DEMOS AND ONE OF THEM WAS THE TECHTECH-DEMO BY SODAN AND MAGICIAN 42. KRILLE AND PUTTE LAUGHED AT US AND SAID THAT IT WAS TOTALLY IMPOSSIBLE TO MAKE ON AN ST. WE STUDIED IT FOR HALF AN HOUR AND SAID: -OF COURSE IT'S POSSIBLE.   WHEN WE WERE BACK HOME (WHEN NO AMIGA-OWNER WAS LISTENING), WE CONCLUDED THAT THERE WAS SIMPLY TOO MUCH MOVEMENT FOR AN ST.        NOW, WE HAVE CONVERTED IT ANYWAY. THE AMIGA VERSION HAD SOME UGLY LINES WHIZZING AROUND, BUT WE HAVE 3 VOICE REAL DIGISOUND AND SOME UGLY SPRITES. BESIDES, WE HAVE SOME TERRIBLE RASTERS.......            WE AGREE THAT THERE ARE BETTER AMIGA-DEMOS NOW, AND PERHAPS WE WILL CONVERT SOME MORE IN THE FUTURE.......     LET'S WRAZZZZZZZ................"

	smallText2 := "                               EVERYBODY THOUGHT IT WAS IMPOSSIBLE.....                                     EVEN WE THOUGHT IT WAS IMPOSSIBLE......                                       IT'S A PITY IT WASN'T.....                                                 THE CAREBEARS PRESENT THE UGLIEST DEMO SO FAR - THE GRODAN AND KVACK KVACK DEMO, A CONVERSION OF THE STUNNING TECHTECH DEMO BY SODAN AND MAGICIAN 42 (ON THE COMPUTER THAT CRASHES WHEN YOU ENTER SUPERVISOR MODE IN SEKA).   IT WAS UGLY ON THE AMIGA TOO, BUT IT SURE KNOCKED YOU OFF THE CHAIR WHEN YOU SAW IT THE FIRST TIME.    "

	g.scrollText1 = NewScrollText(mainText, bsFontMap, 2, false)
	g.scrollText2 = NewScrollText(vertText, upFontMap, 3, true)
	g.scrollText2.scrollX = -100 // Start below screen
	g.scrollText3 = NewScrollText(smallText1, lFontMap, 1, false)
	g.scrollText4 = NewScrollText(smallText2, lFontMap, 2, false)
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

	// Update scroll texts
	if g.scrollText1 != nil {
		g.scrollText1.Update()
	}
	if g.scrollText3 != nil {
		g.scrollText3.Update()
	}
	if g.scrollText4 != nil {
		g.scrollText4.Update()
	}

	if g.scrollText2 != nil {
		g.scrollText2.Update()
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
	if g.scrollText1 == nil {
		return
	}

	// Clear canvas
	g.bs2Canvas.Clear()

	// Draw the glyphs at their final size, without an intermediate canvas.
	g.scrollText1.Draw(g.bs2Canvas, 0, bigScrollXScale, bigScrollYScale)

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
	if g.scrollText2 == nil {
		return
	}

	// Clear canvas
	g.upCanvas.Clear()

	// Draw vertical scroll text
	g.scrollText2.Draw(g.upCanvas, 0, 1, 1)

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
	if g.scrollText3 == nil || g.scrollText4 == nil {
		return
	}

	// Both scrolls share one render target. This lets Ebitengine batch their
	// glyphs and masks before a single final draw to the screen.
	g.smallCanvas.Clear()
	g.scrollText3.Draw(g.smallCanvas, 0, 1, 1)
	g.scrollText4.Draw(g.smallCanvas, 24, 1, 1)

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
