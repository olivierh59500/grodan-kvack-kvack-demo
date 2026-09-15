package grodan

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestYMPlayerReadProducesFloat32StereoWithoutAllocating(t *testing.T) {
	player, err := NewYMPlayer(musicData, sampleRate, true)
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

	fontMap := NewFontMap(8, 8)
	fontMap.AddChar('A', 0, 0, 0)
	fontMap.AddBlank(' ', 0)

	scroll := NewScrollText("a a", fontImage, fontMap, 1, false)
	if got, want := len(scroll.glyphs), 3; got != want {
		t.Fatalf("glyph count = %d, want %d", got, want)
	}
	if got, want := scroll.contentLength, float64(24); got != want {
		t.Fatalf("content length = %v, want %v", got, want)
	}
	if scroll.glyphs[1].image != nil {
		t.Fatal("space unexpectedly has a drawable glyph")
	}
	if scroll.glyphs[0].image != scroll.glyphs[2].image {
		t.Fatal("repeated character did not reuse its cached sub-image")
	}
	if got, want := scroll.glyphs[2].offset, float64(16); got != want {
		t.Fatalf("last glyph offset = %v, want %v", got, want)
	}
}
