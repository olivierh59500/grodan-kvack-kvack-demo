// Package grodan contains the platform-independent demo.
package grodan

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"io"
	"log"
	"math"
	"sync"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/olivierh59500/ym-player/pkg/stsound"
)

const (
	screenWidth  = 640
	screenHeight = 400
	sampleRate   = 44100
)

// Embedded assets
var (
	//go:embed assets/Grodan_green.png
	bgGreenData []byte
	//go:embed assets/Grodan_pink.png
	bgPinkData []byte
	//go:embed assets/upscrollraster.png
	upRasterData []byte
	//go:embed assets/bigscrollraster.png
	bsRasterData []byte
	//go:embed assets/sprite.png
	spriteData []byte
	//go:embed assets/bsfont.png
	bsFontData []byte
	//go:embed assets/upfonts.png
	upFontData []byte
	//go:embed assets/lfont.png
	lFontData []byte
	//go:embed assets/music.ym
	musicData []byte
)

// YMPlayer wraps the YM player for Ebiten
type YMPlayer struct {
	player     *stsound.StSound
	sampleRate int
	buffer     []int16
	mutex      sync.Mutex
	position   int64
	totalBytes int64
	loop       bool
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

	info := player.GetInfo()
	totalSamples := int64(info.MusicTimeInMs) * int64(sampleRate) / 1000

	return &YMPlayer{
		player:     player,
		sampleRate: sampleRate,
		buffer:     make([]int16, 4096),
		totalBytes: totalSamples * 4, // 16-bit stereo: four bytes per frame.
		loop:       loop,
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
	if len(p) < 4 {
		return 0, io.ErrShortBuffer
	}

	samplesNeeded := len(p) / 4
	processed := 0
	for processed < samplesNeeded {
		chunkSize := samplesNeeded - processed
		if chunkSize > len(y.buffer) {
			chunkSize = len(y.buffer)
		}

		if !y.player.Compute(y.buffer[:chunkSize], chunkSize) {
			if !y.loop {
				err = io.EOF
				break
			}
		}

		for i := 0; i < chunkSize; i++ {
			sample := y.buffer[i]
			offset := (processed + i) * 4
			p[offset] = byte(sample)
			p[offset+1] = byte(sample >> 8)
			p[offset+2] = byte(sample)
			p[offset+3] = byte(sample >> 8)
		}

		processed += chunkSize
	}

	n = processed * 4
	y.position += int64(n)

	return n, err
}

// Seek implements io.Seeker
func (y *YMPlayer) Seek(offset int64, whence int) (int64, error) {
	y.mutex.Lock()
	defer y.mutex.Unlock()
	if y.player == nil {
		return 0, io.ErrClosedPipe
	}

	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = y.position + offset
	case io.SeekEnd:
		newPos = y.totalBytes + offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	if newPos < 0 {
		newPos = 0
	}
	if newPos > y.totalBytes {
		newPos = y.totalBytes
	}

	// Ebitengine seeks PCM streams in bytes. StSound seeks in milliseconds.
	newPos -= newPos % 4
	milliseconds := newPos / 4 * 1000 / int64(y.sampleRate)
	y.player.Seek(uint32(milliseconds))
	y.position = newPos
	return newPos, nil
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
	chars      map[rune]CharMapping
	charWidth  int
	charHeight int
}

// NewFontMap creates a font map with automatic character detection
func NewFontMap(charWidth, charHeight int) *FontMap {
	return &FontMap{
		chars:      make(map[rune]CharMapping),
		charWidth:  charWidth,
		charHeight: charHeight,
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
type ScrollText struct {
	text          string
	fontImg       *ebiten.Image
	fontMap       *FontMap
	scrollX       float64
	speed         float64
	contentLength float64
	vertical      bool // For vertical scrolling
}

// NewScrollText creates a new scrolling text
func NewScrollText(text string, fontImg *ebiten.Image, fontMap *FontMap, speed float64, vertical bool) *ScrollText {
	s := &ScrollText{
		text:     text,
		fontImg:  fontImg,
		fontMap:  fontMap,
		speed:    speed,
		vertical: vertical,
	}
	for _, char := range text {
		if vertical {
			s.contentLength += float64(fontMap.charHeight)
			continue
		}
		if mapping, ok := fontMap.chars[unicode.ToUpper(char)]; ok {
			s.contentLength += float64(mapping.width)
		}
	}
	return s
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
func (s *ScrollText) Draw(dst *ebiten.Image, y float64, scale float64) {
	if s.vertical {
		// Vertical scrolling - text moves from bottom to top
		yPos := screenHeight - s.scrollX // Start from bottom of screen

		// Draw text in correct order (not reversed)
		for _, char := range s.text {
			if yPos >= screenHeight {
				break
			}
			if yPos > -float64(s.fontMap.charHeight)*scale && yPos < screenHeight {
				s.drawChar(dst, char, 0, yPos, scale)
			}
			yPos += float64(s.fontMap.charHeight) * scale
		}
	} else {
		// Horizontal scrolling
		x := s.scrollX
		for _, char := range s.text {
			if x >= screenWidth {
				break
			}
			if mapping, ok := s.fontMap.chars[unicode.ToUpper(char)]; ok {
				if x > -float64(mapping.width)*scale && x < float64(screenWidth) {
					s.drawChar(dst, char, x, y, scale)
				}
				x += float64(mapping.width) * scale
			}
		}
	}
}

// drawChar draws a single character
func (s *ScrollText) drawChar(dst *ebiten.Image, char rune, x, y, scale float64) {
	// Convert to uppercase if needed
	char = unicode.ToUpper(char)

	mapping, ok := s.fontMap.chars[char]
	if !ok || !mapping.drawable {
		return // Character not in font map
	}

	srcRect := image.Rect(mapping.x, mapping.y, mapping.x+mapping.width, mapping.y+mapping.height)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)

	dst.DrawImage(s.fontImg.SubImage(srcRect).(*ebiten.Image), op)
}

// Game represents the game state
type Game struct {
	// Images
	bgGreen  *ebiten.Image
	bgPink   *ebiten.Image
	upRaster *ebiten.Image
	bsRaster *ebiten.Image
	sprite   *ebiten.Image
	bsFont   *ebiten.Image
	upFont   *ebiten.Image
	lFont    *ebiten.Image

	// Font maps
	bsFontMap *FontMap
	upFontMap *FontMap
	lFontMap  *FontMap

	// Canvases
	bgCanvas  *ebiten.Image
	bg2Canvas *ebiten.Image
	bsCanvas  *ebiten.Image
	bs2Canvas *ebiten.Image
	upCanvas  *ebiten.Image
	lCanvas   *ebiten.Image
	l2Canvas  *ebiten.Image

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

	ychange float64
	addy    float64
	sinx    float64
	siny    float64
	swing   float64
	swingy  float64
	spx     float64
	spy     float64

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
		moveY:    0,
		howmuchY: 1,
		moveX:    0,
		howmuchX: 1,
		bgcount:  0,
		Y:        0,
		hY:       1,
		X:        0,
		gox:      0,
		ychange:  0,
		addy:     0.1,
		sinx:     0,
		siny:     0,
		swing:    0,
		swingy:   0,
		spx:      304,
		spy:      100,
	}

	// Load images
	g.loadImages()

	// Create canvases
	g.bgCanvas = ebiten.NewImage(640*3, 400*2)
	g.bg2Canvas = ebiten.NewImage(640*3, 400*2)
	g.bsCanvas = ebiten.NewImage(640, 40)
	g.bs2Canvas = ebiten.NewImage(640, 200)
	g.upCanvas = ebiten.NewImage(32, 400)
	g.lCanvas = ebiten.NewImage(320, 8)
	g.l2Canvas = ebiten.NewImage(320, 8)

	// Initialize background canvases
	g.initBackgrounds()

	// Initialize scroll texts
	g.initScrollTexts()

	return g
}

// loadImages loads all image assets
func (g *Game) loadImages() {
	g.bgGreen = mustLoadImage("Grodan_green.png", bgGreenData)
	g.bgPink = mustLoadImage("Grodan_pink.png", bgPinkData)
	g.upRaster = mustLoadImage("upscrollraster.png", upRasterData)
	g.bsRaster = mustLoadImage("bigscrollraster.png", bsRasterData)
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

// initBackgrounds initializes the background canvases
func (g *Game) initBackgrounds() {
	// Initialize green background
	if g.bgGreen != nil {
		for y := 0; y < 2; y++ {
			for x := 0; x < 3; x++ {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(640*x), float64(400*y))
				g.bgCanvas.DrawImage(g.bgGreen, op)
			}
		}
	}

	// Initialize pink background
	if g.bgPink != nil {
		for y := 0; y < 2; y++ {
			for x := 0; x < 3; x++ {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(640*x), float64(400*y))
				g.bg2Canvas.DrawImage(g.bgPink, op)
			}
		}
	}
}

// initScrollTexts initializes the scrolling texts
func (g *Game) initScrollTexts() {
	// Initialize font maps
	g.bsFontMap = initBigScrollFont()
	g.upFontMap = initUpScrollFont()
	g.lFontMap = initSmallFont()

	// Main scroll text
	mainText := "                                 HI AND WELCOME TO THE GRODAN AND KVACK KVACK DEMO (THAT NAME WILL PROBABLY MAKE US FAMOUS IN THE GUINNESS BOOK OF RECORDS - THE MOST STUPID NAME IN DEMO HISTORY.  THE PREVIOUS POSSESSORS OF THAT RECORD WAS OMEGA WITH -OMEGAKUL-.   I'M AFRAID WE WILL SOON BE BEATEN BY SYNC'S 'MJOFFE-DEMO', WITH TWO DOTS ABOVE THE 'O'.  DID YOU KNOW THAT THIS IS A COMMENT IN THE MIDDLE OF A SENTENCE? NO?  WE ALSO FORGOT, BUT LET'S CONTINUE WITH WHAT WE WERE WRITING BEFORE WE STARTED WRITING THIS RECORD-CRAP.), CODED BY NICK AND JAS OF THE CAREBEARS. GRAPHIXXXX BY TANIS, THE GREAT (?) OF THE MEGAMIGHTY CAREBEARS.        WE HAVE TO COVER TWO SUBJECTS IN THIS SCROLLTEXT - THE FANTASTIC WORLD OF HARDWARESCROLLERS  AND  GREETINGS....   LET'S START WITH THE STUFF YOU PROBABLY WANT US TO TALK THE MOST ABOUT - HARDWARESCROLLERS....        TIME: LATE MARCH 1989    PLACE: NICK'S COMPUTER ROOM     IT WORKS!!!!!!!  AFTER HAVING TRIED THE ZANY SCROLLTECHNIQUE ON BOTH NICK'S AND JAS' COMPUTERS, WE CONCLUDED THAT IT ACTUALLY WORKED.    ONE DAY LATER, OMEGA CALLS US AND GOES SOMETHING LIKE THIS: - HAAAA HAAAA  WE KNOW HOW TO SCROLL THE WHOLE SCREEN BOTH HORIZONTALLY AND VERTICALLY IN LESS THAN TEN SCANLINES!!!!!!         WE WERE AMAZED THAT THEY HAD ACTUALLY COME UP WITH THE SAME IDEA ON THE SAME DAY AS US, BUT AT LEAST NOBODY ELSE KNEW HOW TO DO IT.     WE MANAGED TO RELEASE THE FIRST HARDWARESCROLLER THE WORLD HAS SEEN, IN THE CUDDLY DEMOS, AND NOW WE ARE GOING TO USE IT COMERCIALLY (CODING GAMES, DICKHEAD)....     NOW A HINT HOW IT'S DONE:    IT HAS NOTHING TO DO WITH ANY OF THE SOUND-REGISTERS.....         HERE IS ANOTHER ADDRESS TO THE CAREBEARS:     T H E   C A R E B E A R S ,    D R A K E N B E R G S G   2 3    8 T R ,      1 1 7   4  1   S T O  C K H O L M ,     S W E  D E N .                NOW FOR SOME GREETINGS:   MEGADUNDERSUPERDUPERGREETINGS TO  ALL THE OTHER MEMBERS OF THE UNION, ESPECIALLY THE EXCEPTIONS (TANIS WISH TO GIVE A SPECIAL HI TO ES) AND THE REPLICANTS (GOODBYE, RATBOY! YOUR INTROS WERE GREAT).   NORMAL MEGAGREETINGS (IN MERIT-ORDER)(WOW) TO   SYNC (WE'VE CHANGED OUR MINDS, YOU'RE THE SECOND BEST SWEDISH CREW. WE JUST HADN'T SEEN MANY SCREENS BY YOU GUYS (IT'S UNDERSTANDABLE - YOU HAVE ONLY RELEASED THREE NOT VERY GOOD ONES)),  OMEGA (TOO BAD, YOU'RE NOT THE SECOND BEST ANYMORE.  PERHAPS IT HAS SOMETHING TO DO WITH  THE TERA-DISTER, THE 'TCB-E'-JATTEDUMMA'-SIGN OR THE FACT THAT SYNC IS BETTER), THE LOST BOYS (SEE YA' SOON AND WE'RE ANXIOUSLY AWAITING YOUR MEGAMEGADEMO)             SOMETHING BETWEEN MEGAGREETINGS AND NORMAL GREETINGS TO:   FLEXIBLE FRONT (GOODBYE), VECTOR (SO YOU CRACKED OUR DEMO, HUH? NICE SCREEN, BY THE WAY), GHOST (SO YOU TRIED TO CRACK OUR DEMO, HUH? GREAT SCREEN, BY THE WAY), 2 LIFE CREW (YOU ARE IMPROVING), MAGNUM FORCE (YOU SEEM TO BE THE BEST OPTIMIZERS IN FRANCE!), NORDIK CODERS (NICE SCREEN).   NORMAL GREETINGS TO:  FASHION (GOOD LUCK WITH YOUR DEMO), OVERLANDERS (THANKS FOR NOT INCLUDING CUDDLY IN YOUR DEMOBREAKER), NO CREW (ESPECIALLY ROCCO. YOU ARE IMPROVING), AUTOMATION (GREAT COMPACT DISKS), MEDWAY BOYS (NICE CD'S),  ST CONNEXION (HOPE YOUR DEMO WILL BE AS GOOD AS YOUR GRAPHICS), FOXX (COOL SCREEN), FOFT (KEEP ON COMPACTING), ZAE (WE HAD A GREAT TIME IN MARSEILLE), KREATORS (ESPECIALLY CHUD), M.A.R.K.U.S (PLEASE SPREAD THIS DEMO AS MUCH AS YOU SPREAD CUDDLY DEMOS), HACKATARIMAN (THANKS FOR ALL THE STUFF), THE ALLIANCE (ESPECIALLY OVERLANDERS (THANKS FOR TCB-FRIENDLY SCROLLTEXTS AND MANY NICE SCREENS), AND BLACK MONOLITH TEAM (YOUR DEMOSCREEN WAS THE BEST IN THE OLD ALLIANCE DEMO), BIRDY (SEND US YOUR CRACKS), LINKAN 'THE LINK' 'JUDGE LINK' LINKSSON (PING-PONG), NYARLOTHATEPS ADEPTS (STRANGE NAME, STRANGE GUYS), GROWTWIG ( NO COMMENT),  TONY KOLLBERG (TJENA, LYCKA TILL MED ASSEMBLERN)     END OF GREETINGS. IF YOU WERE NOT GREETED, TOO BAD. NORMAL FUCKING GREETINGS TO:  CONSTELLATIONS (NOONE WILL EVER COMPLAIN ABOUT TCB AND GET AWAY WITH IT, BESIDES YOUR DEMO WAS WORTHLESS). MEGA FUCKING GREETINGS TO:     MENACING CRACKING ALLIANCE (SO, YOU DON'T LIKE BEING CALLED LAMERS, HOW YA' LIKE BEING CALLED:       MOTHERFUCKIN'   BLEEDIN' (BRITTISH ENGLISH) ULTIMATE CHICKENBRAINS????!!!! I BET IT'S ALMOST AS FUN AS FUCKING GREET TCB).  END OF SCROLLTEXT. LET'S WRAP."

	// Vertical scroll text
	vertText := "                           TANIS, THE FAMOUS GRAFIXX-MAN, IS A NEW MEMBER OF TCB.  HE MADE ALL THE GRAPHICS IN THIS SCREEN PLUS LOTSA LOGOS IN THE MAIN MENU.  WE AGREE THAT THIS 'ONE-BIT-PLANE-MANIA' DOESN'T LOOK VERY GOOD, BUT IT HAD TO BE DONE BY SOMEONE........   BAD LUCK FOR TANIS THAT WE WON'T MAKE MORE DEMOS, THOUGH....       9 9 9 9 9 9 9 9 9 9 9 9 9 9 9 9 9 9 9  ..................                 LET'S WRAP (WE SPELLED IT CORRECTLY!!!).......   "

	// Small scroll texts
	smallText1 := "                                                        ONCE UPON A TIME, WHEN THE JUNK DEMO WAS ALMOST FINISHED - WHEN THE BEST DEMO ON THE ST-MARKET WAS 'LCD' BY TEX, WE VISITED IQ2-CREW (AMIGA-FREAKS). THEY SHOWED US A COUPLE OF DEMOS AND ONE OF THEM WAS THE TECHTECH-DEMO BY SODAN AND MAGICIAN 42. KRILLE AND PUTTE LAUGHED AT US AND SAID THAT IT WAS TOTALLY IMPOSSIBLE TO MAKE ON AN ST. WE STUDIED IT FOR HALF AN HOUR AND SAID: -OF COURSE IT'S POSSIBLE.   WHEN WE WERE BACK HOME (WHEN NO AMIGA-OWNER WAS LISTENING), WE CONCLUDED THAT THERE WAS SIMPLY TOO MUCH MOVEMENT FOR AN ST.        NOW, WE HAVE CONVERTED IT ANYWAY. THE AMIGA VERSION HAD SOME UGLY LINES WHIZZING AROUND, BUT WE HAVE 3 VOICE REAL DIGISOUND AND SOME UGLY SPRITES. BESIDES, WE HAVE SOME TERRIBLE RASTERS.......            WE AGREE THAT THERE ARE BETTER AMIGA-DEMOS NOW, AND PERHAPS WE WILL CONVERT SOME MORE IN THE FUTURE.......     LET'S WRAZZZZZZZ................"

	smallText2 := "                               EVERYBODY THOUGHT IT WAS IMPOSSIBLE.....                                     EVEN WE THOUGHT IT WAS IMPOSSIBLE......                                       IT'S A PITY IT WASN'T.....                                                 THE CAREBEARS PRESENT THE UGLIEST DEMO SO FAR - THE GRODAN AND KVACK KVACK DEMO, A CONVERSION OF THE STUNNING TECHTECH DEMO BY SODAN AND MAGICIAN 42 (ON THE COMPUTER THAT CRASHES WHEN YOU ENTER SUPERVISOR MODE IN SEKA).   IT WAS UGLY ON THE AMIGA TOO, BUT IT SURE KNOCKED YOU OFF THE CHAIR WHEN YOU SAW IT THE FIRST TIME.    "

	if g.bsFont != nil && g.bsFontMap != nil {
		g.scrollText1 = NewScrollText(mainText, g.bsFont, g.bsFontMap, 2, false)
	}
	if g.upFont != nil && g.upFontMap != nil {
		g.scrollText2 = NewScrollText(vertText, g.upFont, g.upFontMap, 3, true) // Vertical scroll
		g.scrollText2.scrollX = -100                                            // Start below screen
	}
	if g.lFont != nil && g.lFontMap != nil {
		g.scrollText3 = NewScrollText(smallText1, g.lFont, g.lFontMap, 1, false)
		g.scrollText4 = NewScrollText(smallText2, g.lFont, g.lFontMap, 2, false)
	}
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

	g.audioPlayer, err = g.audioContext.NewPlayer(g.ymPlayer)
	if err != nil {
		log.Printf("Failed to create audio player: %v", err)
		g.ymPlayer.Close()
		g.ymPlayer = nil
		return
	}

	g.audioPlayer.SetVolume(0.7)
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
	g.siny = g.ychange * math.Sin(g.swingy)

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

	// Draw background 1
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(g.moveX, g.moveY)
	screen.DrawImage(g.bgCanvas, op)

	// Draw background 2
	op.GeoM.Reset()
	op.GeoM.Translate(g.X, g.Y)
	screen.DrawImage(g.bg2Canvas, op)

	// Draw sprites
	g.drawSprites(screen)

	// Draw big scroll
	g.drawBigScroll(screen)

	// Draw up scroll
	g.drawUpScroll(screen)

	// Draw small scrolls
	g.drawSmallScrolls(screen)
}

// drawSprites draws the animated sprites
func (g *Game) drawSprites(screen *ebiten.Image) {
	if g.sprite == nil {
		return
	}

	// Draw multiple sprites with different phases
	for i := 0; i < 12; i++ {
		phase := float64(i) * 0.2
		x := g.spx + 290*math.Cos(g.swing-phase)
		y := g.spy + g.ychange*math.Sin(g.swingy-phase) + g.siny

		srcX := (i % 12) * 17
		srcRect := image.Rect(srcX, 0, srcX+16, 10)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(x, y)

		screen.DrawImage(g.sprite.SubImage(srcRect).(*ebiten.Image), op)
	}
}

// drawBigScroll draws the big scrolling text
func (g *Game) drawBigScroll(screen *ebiten.Image) {
	if g.scrollText1 == nil || g.bsRaster == nil {
		return
	}

	// Clear canvases
	g.bsCanvas.Clear()
	g.bs2Canvas.Clear()

	// Draw scroll text
	g.scrollText1.Draw(g.bsCanvas, 0, 1)

	// Scale up
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(8, 6)
	g.bs2Canvas.DrawImage(g.bsCanvas, op)

	// Apply raster effect
	op.GeoM.Reset()
	op.GeoM.Scale(4, 2)
	op.Blend = ebiten.BlendSourceAtop
	g.bs2Canvas.DrawImage(g.bsRaster, op)

	// Draw to screen
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 200)
	screen.DrawImage(g.bs2Canvas, op)
}

// drawUpScroll draws the vertical scrolling text
func (g *Game) drawUpScroll(screen *ebiten.Image) {
	if g.scrollText2 == nil || g.upRaster == nil {
		return
	}

	// Clear canvas
	g.upCanvas.Clear()

	// Draw vertical scroll text
	g.scrollText2.Draw(g.upCanvas, 0, 1)

	// Apply raster effect
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	op.Blend = ebiten.BlendSourceAtop
	g.upCanvas.DrawImage(g.upRaster, op)

	// Draw to screen at multiple positions
	positions := []float64{0, 64, 128, 480, 544, 608}
	for _, x := range positions {
		op = &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x, 0)
		screen.DrawImage(g.upCanvas, op)
	}
}

// drawSmallScrolls draws the small scrolling texts
func (g *Game) drawSmallScrolls(screen *ebiten.Image) {
	if g.scrollText3 == nil || g.scrollText4 == nil || g.upRaster == nil {
		return
	}

	// Clear canvases
	g.lCanvas.Clear()
	g.l2Canvas.Clear()

	// Draw scroll text 3
	g.scrollText3.Draw(g.lCanvas, 0, 1)

	// Apply raster effect
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, -16)
	op.GeoM.Scale(2, 2)
	op.Blend = ebiten.BlendSourceAtop
	g.lCanvas.DrawImage(g.upRaster, op)

	// Draw to screen
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(0, 16)
	screen.DrawImage(g.lCanvas, op)

	// Draw scroll text 4
	g.scrollText4.Draw(g.l2Canvas, 0, 1)

	// Apply raster effect
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, -64)
	op.GeoM.Scale(2, 2)
	op.Blend = ebiten.BlendSourceAtop
	g.l2Canvas.DrawImage(g.upRaster, op)

	// Draw to screen
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	op.GeoM.Translate(0, 64)
	screen.DrawImage(g.l2Canvas, op)
}

// Layout returns the screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// Cleanup releases resources
func (g *Game) Cleanup() {
	if g.audioPlayer != nil {
		g.audioPlayer.Close()
	}
	if g.ymPlayer != nil {
		g.ymPlayer.Close()
	}
}
