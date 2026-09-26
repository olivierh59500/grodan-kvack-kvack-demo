package grodan

import (
	"encoding/binary"
	"github.com/olivierh59500/democonstructionkit/font"
	"github.com/olivierh59500/democonstructionkit/motion"
	"github.com/olivierh59500/democonstructionkit/scrolling"
	"image"
	"math"
	"testing"

	"github.com/olivierh59500/democonstructionkit/sound"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestMusicStreamReadProducesFloat32StereoWithoutAllocating(t *testing.T) {
	player, err := sound.Open("music.ym", musicData, sound.Options{SampleRate: sampleRate, Loop: true, PCMFormat: sound.Float32, Gain: 0.5, Quantize16: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := player.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	buffer := make([]byte, 4096*8)
	read := func() {
		n, err := player.Read(buffer)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if n != len(buffer) {
			t.Fatalf("Read bytes = %d, want %d", n, len(buffer))
		}
	}

	read()
	nonZero := false
	for i := 0; i < len(buffer); i += 8 {
		left := binary.LittleEndian.Uint32(buffer[i : i+4])
		right := binary.LittleEndian.Uint32(buffer[i+4 : i+8])
		if left != right {
			t.Fatalf("frame %d is not mono duplicated to stereo: %d != %d", i/8, left, right)
		}
		sample := math.Float32frombits(left)
		if sample != 0 {
			nonZero = true
		}
		if math.IsNaN(float64(sample)) || sample < -0.5 || sample > 0.5 {
			t.Fatalf("frame %d sample = %v, want range [-0.5, 0.5]", i/8, sample)
		}
	}
	if !nonZero {
		t.Fatal("audio buffer unexpectedly contains only silence")
	}

	if allocations := testing.AllocsPerRun(20, read); allocations != 0 {
		t.Fatalf("Read allocations = %v, want 0", allocations)
	}
}

func TestScrollTextCachesGlyphsAndPreservesSpacing(t *testing.T) {
	fontImage := ebiten.NewImage(16, 8)
	t.Cleanup(fontImage.Deallocate)

	metrics, err := font.NewGrid(font.Grid{Bounds: fontImage.Bounds(), Cell: image.Pt(8, 8), Columns: 2, Order: "A", Uppercase: true})
	if err != nil {
		t.Fatal(err)
	}
	fontMap, err := scrolling.NewAtlas(fontImage, metrics)
	if err != nil {
		t.Fatal(err)
	}

	glyphs := fontMap.Layout("a a", scrolling.AtlasText{SkipMissing: true})
	if got, want := len(glyphs), 3; got != want {
		t.Fatalf("glyph count = %d, want %d", got, want)
	}
	config := scrolling.RibbonConfig{Text: "a a", Font: fontMap, SkipMissing: true,
		CullAdvance: 40, Clock: motion.RibbonClockConfig{Velocity: -1, Restart: 640,
			Multiplier: 1, Wrap: motion.RibbonWrapBelow}}
	scroll, err := scrolling.New(scrolling.Config{Ribbon: &config})
	if err != nil {
		t.Fatal(err)
	}
	defer scroll.Close()
	if scroll.RibbonController() == nil {
		t.Fatal("scrolling facade did not expose the ribbon controller")
	}
	if got, want := scroll.RibbonController().Length(), float64(24); got != want {
		t.Fatalf("content length = %v, want %v", got, want)
	}
	if glyphs[1].Image != nil {
		t.Fatal("space unexpectedly has a drawable glyph")
	}
	if glyphs[0].Image != glyphs[2].Image {
		t.Fatal("repeated character did not reuse its cached sub-image")
	}
	if got, want := glyphs[2].Offset, float64(16); got != want {
		t.Fatalf("last glyph offset = %v, want %v", got, want)
	}
}
