// Package grodan contains the platform-independent demo.
package grodan

import originalassets "grodan-kvack-kvack-demo"

import (
	"bytes"

	"fmt"
	"github.com/olivierh59500/democonstructionkit/composite"
	"github.com/olivierh59500/democonstructionkit/scrolling"
	"image"
	"image/color"
	_ "image/png"
	"io"
	"log"
	"math"
	"sync"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	audio "github.com/olivierh59500/democonstructionkit/sound/output"
	"github.com/olivierh59500/ym-player/pkg/stsound"
)

const (
	screenWidth     = 640
	screenHeight    = 400
	sampleRate      = 48000
	spriteCount     = 12
	spriteWidth     = 16
	spriteHeight    = 10
	spriteStride    = 17
	spriteBaseX     = 304
	spriteBaseY     = 100
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

	musicData = originalassets.

		// YMPlayer wraps the YM player for Ebiten
		DCKAssetMusicData()
)

type YMPlayer struct {
	player *stsound.StSound
	buffer []int16
	mutex  sync.Mutex
	loop   bool
}

// NewYMPlayer creates a new YM player
func NewYMPlayer(data []byte, sampleRate int, loop bool) (*YMPlayer, error) {
	if sampleRate <= 0 {
		return nil, fmt.Errorf("sample rate must be positive: %d", sampleRate)
	}

	player := stsound.CreateWithRate(sampleRate)

	if err := player.LoadMemory(data); err != nil {
		player.Destroy()
		return nil, fmt.Errorf("failed to load YM data: %w", err)
	}

	player.SetLoopMode(loop)

	return &YMPlayer{
		player: player,
		buffer: make([]int16, 4096),
		loop:   loop,
	}, nil
}

// Read implements io.Reader
func (y *YMPlayer) Read(p []byte) (n int, err error) {
	y.mutex.Lock()
	defer y.mutex.Unlock()

	if y.player == nil {
		return 0, io.ErrClosedPipe
	}
	if len(p) == 0 {
		return 0, nil
	}
	if len(p) < 8 {
		return 0, io.ErrShortBuffer
	}

	samplesNeeded := len(p) / 8
	processed := 0
	for processed < samplesNeeded {
		chunkSize := samplesNeeded - processed
		if chunkSize > len(y.buffer) {
			chunkSize = len(y.buffer)
		}

		if !y.player.Compute(y.buffer[:chunkSize], chunkSize) {
			if !y.loop {
				clear(p[processed*8 : samplesNeeded*8])
				err = io.EOF
				break
			}
		}

		for i := 0; i < chunkSize; i++ {
			sample := float32(y.buffer[i]/2) / (1 << 15)
			bits := math.Float32bits(sample)
			offset := (processed + i) * 8
			p[offset] = byte(bits)
			p[offset+1] = byte(bits >> 8)
			p[offset+2] = byte(bits >> 16)
			p[offset+3] = byte(bits >> 24)
			p[offset+4] = byte(bits)
			p[offset+5] = byte(bits >> 8)
			p[offset+6] = byte(bits >> 16)
			p[offset+7] = byte(bits >> 24)
		}

		processed += chunkSize
	}

	return samplesNeeded * 8, err
}

// Close releases resources
func (y *YMPlayer) Close() error {
	y.mutex.Lock()
	defer y.mutex.Unlock()

	if y.player != nil {
		y.player.Destroy()
		y.player = nil
	}
	return nil
}

// CharMapping represents character position in font image
type CharMapping struct {
	x, y, width, height int
	drawable            bool
}

// FontMap manages character mappings for a bitmap font
type FontMap struct {
	chars       map[rune]CharMapping
	glyphImages map[rune]*ebiten.Image
	charWidth   int
	charHeight  int
}

// NewFontMap creates a font map with automatic character detection
func NewFontMap(charWidth, charHeight int) *FontMap {
	return &FontMap{
		chars:       make(map[rune]CharMapping),
		glyphImages: make(map[rune]*ebiten.Image),
		charWidth:   charWidth,
		charHeight:  charHeight,
	}
}

// AddChar adds a character mapping
func (fm *FontMap) AddChar(char rune, col, row int, width int) {
	if width == 0 {
		width = fm.charWidth
	}
	fm.chars[char] = CharMapping{
		x:        col * fm.charWidth,
		y:        row * fm.charHeight,
		width:    width,
		height:   fm.charHeight,
		drawable: true,
	}
}

// AddBlank adds spacing for a character that has no glyph in the font image.
func (fm *FontMap) AddBlank(char rune, width int) {
	if width == 0 {
		width = fm.charWidth
	}
	fm.chars[char] = CharMapping{width: width, height: fm.charHeight}
}

// InitBigScrollFont initializes the big scroll font (24x33)
func initBigScrollFont() *FontMap {
	fm := NewFontMap(24, 33)

	// Row 0: [NA]![NA][NA][NA]'"()
	fm.AddChar('!', 1, 0, 0)
	fm.AddChar('\'', 5, 0, 0)
	fm.AddChar('"', 6, 0, 0)
	fm.AddChar('(', 7, 0, 0)
	fm.AddChar(')', 8, 0, 0)

	// Row 1: [NA][NA][NA][NA].,0123
	fm.AddChar('.', 4, 1, 0)
	fm.AddChar(',', 5, 1, 0)
	fm.AddChar('0', 6, 1, 0)
	fm.AddChar('1', 7, 1, 0)
	fm.AddChar('2', 8, 1, 0)
	fm.AddChar('3', 9, 1, 0)

	// Row 2: 456789:[NA][NA][NA]
	fm.AddChar('4', 0, 2, 0)
	fm.AddChar('5', 1, 2, 0)
	fm.AddChar('6', 2, 2, 0)
	fm.AddChar('7', 3, 2, 0)
	fm.AddChar('8', 4, 2, 0)
	fm.AddChar('9', 5, 2, 0)
	fm.AddChar(':', 6, 2, 0)

	// Row 3: [NA]?[NA]ABCDEFG
	fm.AddChar('?', 1, 3, 0)
	fm.AddChar('A', 3, 3, 0)
	fm.AddChar('B', 4, 3, 0)
	fm.AddChar('C', 5, 3, 0)
	fm.AddChar('D', 6, 3, 0)
	fm.AddChar('E', 7, 3, 0)
	fm.AddChar('F', 8, 3, 0)
	fm.AddChar('G', 9, 3, 0)

	// Row 4: HIJKLMNOPQ
	fm.AddChar('H', 0, 4, 0)
	fm.AddChar('I', 1, 4, 0)
	fm.AddChar('J', 2, 4, 0)
	fm.AddChar('K', 3, 4, 0)
	fm.AddChar('L', 4, 4, 0)
	fm.AddChar('M', 5, 4, 0)
	fm.AddChar('N', 6, 4, 0)
	fm.AddChar('O', 7, 4, 0)
	fm.AddChar('P', 8, 4, 0)
	fm.AddChar('Q', 9, 4, 0)

	// Row 5: RSTUVWXYZ[NA]
	fm.AddChar('R', 0, 5, 0)
	fm.AddChar('S', 1, 5, 0)
	fm.AddChar('T', 2, 5, 0)
	fm.AddChar('U', 3, 5, 0)
	fm.AddChar('V', 4, 5, 0)
	fm.AddChar('W', 5, 5, 0)
	fm.AddChar('X', 6, 5, 0)
	fm.AddChar('Y', 7, 5, 0)
	fm.AddChar('Z', 8, 5, 0)

	// Space is handled separately (no graphic)
	fm.AddBlank(' ', 24)
	fm.AddBlank('-', 24)

	return fm
}

// InitUpScrollFont initializes the vertical scroll font (33x29)
func initUpScrollFont() *FontMap {
	fm := NewFontMap(33, 29)

	// Row 0: [NA]![NA][NA][NA][NA][NA][NA]()
	fm.AddChar('!', 1, 0, 0)
	fm.AddChar('(', 8, 0, 0)
	fm.AddChar(')', 9, 0, 0)

	// Row 1: [NA][NA][NA][NA].[NA][NA][NA][NA][NA]
	fm.AddChar('.', 4, 1, 0)

	// Row 2: [NA][NA][NA][NA][NA]#:[NA][NA][NA]
	fm.AddChar('#', 5, 2, 0)
	fm.AddChar(':', 6, 2, 0)

	// Row 3: [NA]?[NA]ABCDEFG
	fm.AddChar('?', 1, 3, 0)
	fm.AddChar('A', 3, 3, 0)
	fm.AddChar('B', 4, 3, 0)
	fm.AddChar('C', 5, 3, 0)
	fm.AddChar('D', 6, 3, 0)
	fm.AddChar('E', 7, 3, 0)
	fm.AddChar('F', 8, 3, 0)
	fm.AddChar('G', 9, 3, 0)

	// Row 4: HIJKLMNOPQ
	fm.AddChar('H', 0, 4, 0)
	fm.AddChar('I', 1, 4, 0)
	fm.AddChar('J', 2, 4, 0)
	fm.AddChar('K', 3, 4, 0)
	fm.AddChar('L', 4, 4, 0)
	fm.AddChar('M', 5, 4, 0)
	fm.AddChar('N', 6, 4, 0)
	fm.AddChar('O', 7, 4, 0)
	fm.AddChar('P', 8, 4, 0)
	fm.AddChar('Q', 9, 4, 0)

	// Row 5: RSTUVWXYZ[NA]
	fm.AddChar('R', 0, 5, 0)
	fm.AddChar('S', 1, 5, 0)
	fm.AddChar('T', 2, 5, 0)
	fm.AddChar('U', 3, 5, 0)
	fm.AddChar('V', 4, 5, 0)
	fm.AddChar('W', 5, 5, 0)
	fm.AddChar('X', 6, 5, 0)
	fm.AddChar('Y', 7, 5, 0)
	fm.AddChar('Z', 8, 5, 0)

	// Numbers 0-9 (not in this font, but referenced in text)
	fm.AddBlank('0', 33)
	fm.AddBlank('1', 33)
	fm.AddBlank('2', 33)
	fm.AddBlank('3', 33)
	fm.AddBlank('4', 33)
	fm.AddBlank('5', 33)
	fm.AddBlank('6', 33)
	fm.AddBlank('7', 33)
	fm.AddBlank('8', 33)
	fm.AddBlank('9', 33)

	// Space and missing characters
	fm.AddBlank(' ', 33)
	fm.AddBlank('-', 33)
	fm.AddBlank(',', 33)
	fm.AddBlank('\'', 33)

	return fm
}

// InitSmallFont initializes the small font (8x8)
func initSmallFont() *FontMap {
	fm := NewFontMap(8, 8)

	// Row 0: [NA]![NA][NA][NA][NA][NA]'()
	fm.AddChar('!', 1, 0, 0)
	fm.AddChar('\'', 7, 0, 0)
	fm.AddChar('(', 8, 0, 0)
	fm.AddChar(')', 9, 0, 0)

	// Row 1: [NA][NA][NA][NA]./0123
	fm.AddChar('.', 4, 1, 0)
	fm.AddChar('/', 5, 1, 0)
	fm.AddChar('0', 6, 1, 0)
	fm.AddChar('1', 7, 1, 0)
	fm.AddChar('2', 8, 1, 0)
	fm.AddChar('3', 9, 1, 0)

	// Row 2: 456789:[NA][NA][NA]
	fm.AddChar('4', 0, 2, 0)
	fm.AddChar('5', 1, 2, 0)
	fm.AddChar('6', 2, 2, 0)
	fm.AddChar('7', 3, 2, 0)
	fm.AddChar('8', 4, 2, 0)
	fm.AddChar('9', 5, 2, 0)
	fm.AddChar(':', 6, 2, 0)

	// Row 3: [NA]?[NA]ABCDEFG
	fm.AddChar('?', 1, 3, 0)
	fm.AddChar('A', 3, 3, 0)
	fm.AddChar('B', 4, 3, 0)
	fm.AddChar('C', 5, 3, 0)
	fm.AddChar('D', 6, 3, 0)
	fm.AddChar('E', 7, 3, 0)
	fm.AddChar('F', 8, 3, 0)
	fm.AddChar('G', 9, 3, 0)

	// Row 4: HIJKLMNOPQ
	fm.AddChar('H', 0, 4, 0)
	fm.AddChar('I', 1, 4, 0)
	fm.AddChar('J', 2, 4, 0)
	fm.AddChar('K', 3, 4, 0)
	fm.AddChar('L', 4, 4, 0)
	fm.AddChar('M', 5, 4, 0)
	fm.AddChar('N', 6, 4, 0)
	fm.AddChar('O', 7, 4, 0)
	fm.AddChar('P', 8, 4, 0)
	fm.AddChar('Q', 9, 4, 0)

	// Row 5: RSTUVWXYZ[NA]
	fm.AddChar('R', 0, 5, 0)
	fm.AddChar('S', 1, 5, 0)
	fm.AddChar('T', 2, 5, 0)
	fm.AddChar('U', 3, 5, 0)
	fm.AddChar('V', 4, 5, 0)
	fm.AddChar('W', 5, 5, 0)
	fm.AddChar('X', 6, 5, 0)
	fm.AddChar('Y', 7, 5, 0)
	fm.AddChar('Z', 8, 5, 0)

	// Space and missing characters
	fm.AddBlank(' ', 8)
	fm.AddBlank('-', 8)
	fm.AddBlank(',', 8)
	fm.AddBlank('"', 8)

	return fm
}

// ScrollText manages scrolling text
type scrollGlyph struct {
	image   *ebiten.Image
	offset  float64
	advance float64
}

type ScrollText struct {
	renderer      *scrolling.Scrolling
	glyphs        []scrollGlyph
	scrollX       float64
	speed         float64
	contentLength float64
	charHeight    float64
	vertical      bool // For vertical scrolling
}

// NewScrollText creates a new scrolling text
func NewScrollText(text string, fontImg *ebiten.Image, fontMap *FontMap, speed float64, vertical bool) *ScrollText {
	s := &ScrollText{
		glyphs:     make([]scrollGlyph, 0, len(text)),
		speed:      speed,
		charHeight: float64(fontMap.charHeight),
		vertical:   vertical,
	}
	for _, char := range text {
		char = unicode.ToUpper(char)
		mapping, ok := fontMap.chars[char]
		if vertical {
			s.glyphs = append(s.glyphs, scrollGlyph{
				image:   fontMap.glyphImage(fontImg, char, mapping, ok),
				offset:  s.contentLength,
				advance: s.charHeight,
			})
			s.contentLength += s.charHeight
			continue
		}
		if !ok {
			continue
		}
		advance := float64(mapping.width)
		s.glyphs = append(s.glyphs, scrollGlyph{
			image:   fontMap.glyphImage(fontImg, char, mapping, true),
			offset:  s.contentLength,
			advance: advance,
		})
		s.contentLength += advance
	}

	glyphs := make([]scrolling.Glyph, len(s.glyphs))
	for i, g := range s.glyphs {
		glyphs[i] = scrolling.Glyph{Image: g.image, Advance: g.advance}
	}
	var err error
	s.renderer, err = scrolling.New(scrolling.Config{Glyphs: glyphs, Vertical: vertical})
	if err != nil {
		panic(err)
	}
	return s
}

func (fm *FontMap) glyphImage(fontImg *ebiten.Image, char rune, mapping CharMapping, ok bool) *ebiten.Image {
	if !ok || !mapping.drawable {
		return nil
	}
	if glyph, exists := fm.glyphImages[char]; exists {
		return glyph
	}
	rect := image.Rect(mapping.x, mapping.y, mapping.x+mapping.width, mapping.y+mapping.height)
	glyph := fontImg.SubImage(rect).(*ebiten.Image)
	fm.glyphImages[char] = glyph
	return glyph
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
	upScrollRaster    *ebiten.Image
	bigScrollRaster   *ebiten.Image
	smallRasterTop    *ebiten.Image
	smallRasterBottom *ebiten.Image
	sprite            *ebiten.Image
	bsFont            *ebiten.Image
	upFont            *ebiten.Image
	lFont             *ebiten.Image
	sprites           [spriteCount]*ebiten.Image

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

	ychange        float64
	addy           float64
	siny           float64
	swing          float64
	swingy         float64
	swingSin       float64
	swingCos       float64
	swingYSin      float64
	swingYCos      float64
	spritePhaseSin [spriteCount]float64
	spritePhaseCos [spriteCount]float64

	// Scroll texts
	scrollText1 *ScrollText
	scrollText2 *ScrollText
	scrollText3 *ScrollText
	scrollText4 *ScrollText

	// Audio
	audioContext     *audio.Context
	audioPlayer      *audio.Player
	ymPlayer         *YMPlayer
	audioInitialized bool
}

// NewGame creates a new game instance
func NewGame() *Game {
	g := &Game{
		moveY:     0,
		howmuchY:  1,
		moveX:     0,
		howmuchX:  1,
		bgcount:   0,
		Y:         0,
		hY:        1,
		X:         0,
		gox:       0,
		ychange:   0,
		addy:      0.1,
		siny:      0,
		swing:     0,
		swingy:    0,
		swingCos:  1,
		swingYCos: 1,
	}

	// Load images
	g.loadImages()
	for i := range spriteCount {
		srcX := i * spriteStride
		rect := image.Rect(srcX, 0, srcX+spriteWidth, spriteHeight)
		g.sprites[i] = g.sprite.SubImage(rect).(*ebiten.Image)
		g.spritePhaseSin[i], g.spritePhaseCos[i] = math.Sincos(float64(i) * 0.2)
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
	bsFontMap := initBigScrollFont()
	upFontMap := initUpScrollFont()
	lFontMap := initSmallFont()

	// Main scroll text
	mainText := "                                 HI AND WELCOME TO THE GRODAN AND KVACK KVACK DEMO (THAT NAME WILL PROBABLY MAKE US FAMOUS IN THE GUINNESS BOOK OF RECORDS - THE MOST STUPID NAME IN DEMO HISTORY.  THE PREVIOUS POSSESSORS OF THAT RECORD WAS OMEGA WITH -OMEGAKUL-.   I'M AFRAID WE WILL SOON BE BEATEN BY SYNC'S 'MJOFFE-DEMO', WITH TWO DOTS ABOVE THE 'O'.  DID YOU KNOW THAT THIS IS A COMMENT IN THE MIDDLE OF A SENTENCE? NO?  WE ALSO FORGOT, BUT LET'S CONTINUE WITH WHAT WE WERE WRITING BEFORE WE STARTED WRITING THIS RECORD-CRAP.), CODED BY NICK AND JAS OF THE CAREBEARS. GRAPHIXXXX BY TANIS, THE GREAT (?) OF THE MEGAMIGHTY CAREBEARS.        WE HAVE TO COVER TWO SUBJECTS IN THIS SCROLLTEXT - THE FANTASTIC WORLD OF HARDWARESCROLLERS  AND  GREETINGS....   LET'S START WITH THE STUFF YOU PROBABLY WANT US TO TALK THE MOST ABOUT - HARDWARESCROLLERS....        TIME: LATE MARCH 1989    PLACE: NICK'S COMPUTER ROOM     IT WORKS!!!!!!!  AFTER HAVING TRIED THE ZANY SCROLLTECHNIQUE ON BOTH NICK'S AND JAS' COMPUTERS, WE CONCLUDED THAT IT ACTUALLY WORKED.    ONE DAY LATER, OMEGA CALLS US AND GOES SOMETHING LIKE THIS: - HAAAA HAAAA  WE KNOW HOW TO SCROLL THE WHOLE SCREEN BOTH HORIZONTALLY AND VERTICALLY IN LESS THAN TEN SCANLINES!!!!!!         WE WERE AMAZED THAT THEY HAD ACTUALLY COME UP WITH THE SAME IDEA ON THE SAME DAY AS US, BUT AT LEAST NOBODY ELSE KNEW HOW TO DO IT.     WE MANAGED TO RELEASE THE FIRST HARDWARESCROLLER THE WORLD HAS SEEN, IN THE CUDDLY DEMOS, AND NOW WE ARE GOING TO USE IT COMERCIALLY (CODING GAMES, DICKHEAD)....     NOW A HINT HOW IT'S DONE:    IT HAS NOTHING TO DO WITH ANY OF THE SOUND-REGISTERS.....         HERE IS ANOTHER ADDRESS TO THE CAREBEARS:     T H E   C A R E B E A R S ,    D R A K E N B E R G S G   2 3    8 T R ,      1 1 7   4  1   S T O  C K H O L M ,     S W E  D E N .                NOW FOR SOME GREETINGS:   MEGADUNDERSUPERDUPERGREETINGS TO  ALL THE OTHER MEMBERS OF THE UNION, ESPECIALLY THE EXCEPTIONS (TANIS WISH TO GIVE A SPECIAL HI TO ES) AND THE REPLICANTS (GOODBYE, RATBOY! YOUR INTROS WERE GREAT).   NORMAL MEGAGREETINGS (IN MERIT-ORDER)(WOW) TO   SYNC (WE'VE CHANGED OUR MINDS, YOU'RE THE SECOND BEST SWEDISH CREW. WE JUST HADN'T SEEN MANY SCREENS BY YOU GUYS (IT'S UNDERSTANDABLE - YOU HAVE ONLY RELEASED THREE NOT VERY GOOD ONES)),  OMEGA (TOO BAD, YOU'RE NOT THE SECOND BEST ANYMORE.  PERHAPS IT HAS SOMETHING TO DO WITH  THE TERA-DISTER, THE 'TCB-E'-JATTEDUMMA'-SIGN OR THE FACT THAT SYNC IS BETTER), THE LOST BOYS (SEE YA' SOON AND WE'RE ANXIOUSLY AWAITING YOUR MEGAMEGADEMO)             SOMETHING BETWEEN MEGAGREETINGS AND NORMAL GREETINGS TO:   FLEXIBLE FRONT (GOODBYE), VECTOR (SO YOU CRACKED OUR DEMO, HUH? NICE SCREEN, BY THE WAY), GHOST (SO YOU TRIED TO CRACK OUR DEMO, HUH? GREAT SCREEN, BY THE WAY), 2 LIFE CREW (YOU ARE IMPROVING), MAGNUM FORCE (YOU SEEM TO BE THE BEST OPTIMIZERS IN FRANCE!), NORDIK CODERS (NICE SCREEN).   NORMAL GREETINGS TO:  FASHION (GOOD LUCK WITH YOUR DEMO), OVERLANDERS (THANKS FOR NOT INCLUDING CUDDLY IN YOUR DEMOBREAKER), NO CREW (ESPECIALLY ROCCO. YOU ARE IMPROVING), AUTOMATION (GREAT COMPACT DISKS), MEDWAY BOYS (NICE CD'S),  ST CONNEXION (HOPE YOUR DEMO WILL BE AS GOOD AS YOUR GRAPHICS), FOXX (COOL SCREEN), FOFT (KEEP ON COMPACTING), ZAE (WE HAD A GREAT TIME IN MARSEILLE), KREATORS (ESPECIALLY CHUD), M.A.R.K.U.S (PLEASE SPREAD THIS DEMO AS MUCH AS YOU SPREAD CUDDLY DEMOS), HACKATARIMAN (THANKS FOR ALL THE STUFF), THE ALLIANCE (ESPECIALLY OVERLANDERS (THANKS FOR TCB-FRIENDLY SCROLLTEXTS AND MANY NICE SCREENS), AND BLACK MONOLITH TEAM (YOUR DEMOSCREEN WAS THE BEST IN THE OLD ALLIANCE DEMO), BIRDY (SEND US YOUR CRACKS), LINKAN 'THE LINK' 'JUDGE LINK' LINKSSON (PING-PONG), NYARLOTHATEPS ADEPTS (STRANGE NAME, STRANGE GUYS), GROWTWIG ( NO COMMENT),  TONY KOLLBERG (TJENA, LYCKA TILL MED ASSEMBLERN)     END OF GREETINGS. IF YOU WERE NOT GREETED, TOO BAD. NORMAL FUCKING GREETINGS TO:  CONSTELLATIONS (NOONE WILL EVER COMPLAIN ABOUT TCB AND GET AWAY WITH IT, BESIDES YOUR DEMO WAS WORTHLESS). MEGA FUCKING GREETINGS TO:     MENACING CRACKING ALLIANCE (SO, YOU DON'T LIKE BEING CALLED LAMERS, HOW YA' LIKE BEING CALLED:       MOTHERFUCKIN'   BLEEDIN' (BRITTISH ENGLISH) ULTIMATE CHICKENBRAINS????!!!! I BET IT'S ALMOST AS FUN AS FUCKING GREET TCB).  END OF SCROLLTEXT. LET'S WRAP."

	// Vertical scroll text
	vertText := "                           TANIS, THE FAMOUS GRAFIXX-MAN, IS A NEW MEMBER OF TCB.  HE MADE ALL THE GRAPHICS IN THIS SCREEN PLUS LOTSA LOGOS IN THE MAIN MENU.  WE AGREE THAT THIS 'ONE-BIT-PLANE-MANIA' DOESN'T LOOK VERY GOOD, BUT IT HAD TO BE DONE BY SOMEONE........   BAD LUCK FOR TANIS THAT WE WON'T MAKE MORE DEMOS, THOUGH....       9 9 9 9 9 9 9 9 9 9 9 9 9 9 9 9 9 9 9  ..................                 LET'S WRAP (WE SPELLED IT CORRECTLY!!!).......   "

	// Small scroll texts
	smallText1 := "                                                        ONCE UPON A TIME, WHEN THE JUNK DEMO WAS ALMOST FINISHED - WHEN THE BEST DEMO ON THE ST-MARKET WAS 'LCD' BY TEX, WE VISITED IQ2-CREW (AMIGA-FREAKS). THEY SHOWED US A COUPLE OF DEMOS AND ONE OF THEM WAS THE TECHTECH-DEMO BY SODAN AND MAGICIAN 42. KRILLE AND PUTTE LAUGHED AT US AND SAID THAT IT WAS TOTALLY IMPOSSIBLE TO MAKE ON AN ST. WE STUDIED IT FOR HALF AN HOUR AND SAID: -OF COURSE IT'S POSSIBLE.   WHEN WE WERE BACK HOME (WHEN NO AMIGA-OWNER WAS LISTENING), WE CONCLUDED THAT THERE WAS SIMPLY TOO MUCH MOVEMENT FOR AN ST.        NOW, WE HAVE CONVERTED IT ANYWAY. THE AMIGA VERSION HAD SOME UGLY LINES WHIZZING AROUND, BUT WE HAVE 3 VOICE REAL DIGISOUND AND SOME UGLY SPRITES. BESIDES, WE HAVE SOME TERRIBLE RASTERS.......            WE AGREE THAT THERE ARE BETTER AMIGA-DEMOS NOW, AND PERHAPS WE WILL CONVERT SOME MORE IN THE FUTURE.......     LET'S WRAZZZZZZZ................"

	smallText2 := "                               EVERYBODY THOUGHT IT WAS IMPOSSIBLE.....                                     EVEN WE THOUGHT IT WAS IMPOSSIBLE......                                       IT'S A PITY IT WASN'T.....                                                 THE CAREBEARS PRESENT THE UGLIEST DEMO SO FAR - THE GRODAN AND KVACK KVACK DEMO, A CONVERSION OF THE STUNNING TECHTECH DEMO BY SODAN AND MAGICIAN 42 (ON THE COMPUTER THAT CRASHES WHEN YOU ENTER SUPERVISOR MODE IN SEKA).   IT WAS UGLY ON THE AMIGA TOO, BUT IT SURE KNOCKED YOU OFF THE CHAIR WHEN YOU SAW IT THE FIRST TIME.    "

	g.scrollText1 = NewScrollText(mainText, g.bsFont, bsFontMap, 2, false)
	g.scrollText2 = NewScrollText(vertText, g.upFont, upFontMap, 3, true)
	g.scrollText2.scrollX = -100 // Start below screen
	g.scrollText3 = NewScrollText(smallText1, g.lFont, lFontMap, 1, false)
	g.scrollText4 = NewScrollText(smallText2, g.lFont, lFontMap, 2, false)
}

// initAudio initializes the audio system
func (g *Game) initAudio() {
	g.audioContext = audio.NewContext(sampleRate)

	var err error
	g.ymPlayer, err = NewYMPlayer(musicData, sampleRate, true)
	if err != nil {
		log.Printf("Failed to create YM player: %v", err)
		return
	}

	g.audioPlayer, err = g.audioContext.NewPlayerF32(g.ymPlayer)
	if err != nil {
		log.Printf("Failed to create audio player: %v", err)
		if closeErr := g.ymPlayer.Close(); closeErr != nil {
			log.Printf("Failed to close YM player: %v", closeErr)
		}
		g.ymPlayer = nil
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

	// Update sprite animation
	if g.ychange > 50 {
		g.addy = -0.1
	}
	if g.ychange < -50 {
		g.addy = 0.1
	}
	g.ychange += g.addy

	g.swing += 0.02
	g.swingy += 0.03
	g.swingSin, g.swingCos = math.Sincos(g.swing)
	g.swingYSin, g.swingYCos = math.Sincos(g.swingy)
	g.siny = g.ychange * g.swingYSin

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

	g.drawTiledBackground(screen, g.bgGreen, g.moveX, g.moveY)
	g.drawTiledBackground(screen, g.bgPink, g.X, g.Y)

	// Draw sprites
	g.drawSprites(screen)

	// Draw big scroll
	g.drawBigScroll(screen)

	// Draw up scroll
	g.drawUpScroll(screen)

	// Draw small scrolls
	g.drawSmallScrolls(screen)
}

func (g *Game) drawTiledBackground(screen, background *ebiten.Image, offsetX, offsetY float64) {
	width := float64(background.Bounds().Dx())
	height := float64(background.Bounds().Dy())
	for y := range 2 {
		tileY := offsetY + float64(screenHeight*y)
		if tileY >= screenHeight || tileY+height <= 0 {
			continue
		}
		for x := range 3 {
			tileX := offsetX + float64(screenWidth*x)
			if tileX >= screenWidth || tileX+width <= 0 {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(tileX, tileY)
			composite.Instance{Image: background, Options: *op}.Draw(screen)
		}
	}
}

// drawSprites draws the animated sprites
func (g *Game) drawSprites(screen *ebiten.Image) {
	if g.sprite == nil {
		return
	}

	// Draw multiple sprites with different phases
	for i, sprite := range g.sprites {
		cosPhase := g.spritePhaseCos[i]
		sinPhase := g.spritePhaseSin[i]
		cosSwing := g.swingCos*cosPhase + g.swingSin*sinPhase
		sinSwingY := g.swingYSin*cosPhase - g.swingYCos*sinPhase
		x := spriteBaseX + 290*cosSwing
		y := spriteBaseY + g.ychange*sinSwingY + g.siny

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(x, y)

		composite.Instance{Image: sprite, Options: *op}.Draw(screen)
	}
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
	if g.ymPlayer != nil {
		if err := g.ymPlayer.Close(); err != nil {
			log.Printf("Failed to close YM player: %v", err)
		}
		g.ymPlayer = nil
	}
}
