package grodan

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/olivierh59500/democonstructionkit/sound"
)

// TestMusicPCMCompatibility preserves the audible level and PCM of the original adapter.
func TestMusicPCMCompatibility(t *testing.T) {
	player, err := sound.Open("music.ym", musicData, sound.Options{SampleRate: sampleRate, Loop: true, PCMFormat: sound.Float32, Gain: 0.5, Quantize16: true})
	if err != nil {
		t.Fatal(err)
	}
	defer player.Close()
	data := make([]byte, sampleRate*8)
	for start := 0; start < len(data); {
		end := min(start+4096, len(data))
		n, err := player.Read(data[start:end])
		if err != nil {
			t.Fatal(err)
		}
		if n != end-start {
			t.Fatalf("read %d/%d", n, end-start)
		}
		start = end
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != "8e11e654cb4420f4f13ba04bc6b33cb26d69069c086fbabe14591f20e99ab088" {
		t.Fatalf("soundtrack PCM changed: %s", got)
	}
}
