# Grodan and Kvack Kvack Demo - Go/Ebiten Port

A faithful port of the classic Atari ST demo "Grodan and Kvack Kvack" by The Carebears (TCB), originally created by Nick and Jas with graphics by Tanis. This port is written in Go using the Ebiten 2D game engine.

## Overview

This demo recreates the visual effects of the original 1989 Atari ST demo, featuring:
- Multiple scrolling text effects
- Animated backgrounds
- Floating sprites in a sinusoidal pattern
- YM chiptune music playback
- Authentic bitmap fonts

## Features

- **Four Scrolling Texts**: 
  - Large horizontal scroll with raster effects
  - Vertical scroll on the sides (moving bottom to top)
  - Two small horizontal scrolls at the top
- **Dual Animated Backgrounds**: Green and pink backgrounds with complex movement patterns
- **Sprite Animation**: 12 animated sprites following a sinusoidal trajectory
- **Authentic Fonts**: Three different bitmap fonts (24x33, 33x29, and 8x8 pixels)
- **YM Music**: Original chiptune music playback using YM format

## Requirements

- Go 1.25 or higher
- Dependencies will be automatically downloaded by Go modules

## Dependencies

- [Ebiten v2](https://github.com/hajimehoshi/ebiten) - A dead simple 2D game library for Go
- [ym-player](https://github.com/olivierh59500/ym-player) - YM music player library

## Installation

1. Clone this repository:
```bash
git clone https://github.com/olivierh59500/grodan-kvack-kvack-demo
cd grodan-kvack-kvack-demo
```

2. Download dependencies:
```bash
go mod download
```

3. Ensure you have the required assets in the `assets/` directory:
   - `Grodan_green.png` - Green background image (640x400)
   - `Grodan_pink.png` - Pink background image (640x400)
   - `upscrollraster.png` - Raster effect for vertical scroll
   - `bigscrollraster.png` - Raster effect for big scroll
   - `sprite.png` - Sprite strip (17x10 pixels per sprite, 12 sprites)
   - `bsfont.png` - Big scroll font (24x33 per character)
   - `upfonts.png` - Vertical scroll font (33x29 per character)
   - `lfont.png` - Small font (8x8 per character)
   - `music.ym` - YM format music file

## Running the Demo

```bash
go run ./cmd/grodan
```

Or build and run:
```bash
go build -o grodan-demo ./cmd/grodan
./grodan-demo
```

## Running on Android

The repository includes an Android shell configured for an ARM64 Google Pixel.
Connect and authorize exactly one device over USB, then run:

```bash
./scripts/run-android.sh
```

The script generates the Ebitengine AAR, builds the debug APK, installs it, and
starts `com.olivierh.grodankvackkvack/.MainActivity`. It detects the Homebrew
Android SDK and Java 17 paths documented in `GUIDE_ANDROID_EBITENGINE_PIXEL.md`.

To validate the Go code without launching the demo:

```bash
go test ./...
go vet ./...
```

## Performance notes

- YM synthesis uses `ym-player` revision `3f73bdca82e5`, 48 kHz output, and a
  reusable zero-allocation float32 stereo stream.
- The desktop launcher only rebuilds a frame after a game update. This avoids
  rendering the same state twice on 120 Hz and faster displays while the demo
  continues to run at 60 updates per second.
- Font glyphs, sprite frames, and raster slices are cached during startup.
- Scrolling text uses binary search to visit only visible glyphs.
- Transient effects use unmanaged render targets that are fully reconstructed
  every frame. Large scrolling backgrounds are tiled directly to reduce GPU
  memory use.

On the Pixel 10a used for profiling, these changes kept presentation at about
60 FPS with no missed VSync in the sampled window. Compared with commit
`1ac0693`, active graphics memory dropped from roughly 183 MB to 100 MB. Exact
CPU and memory figures depend on device state and build mode.

## Technical Details

### Font Mapping
The bitmap fonts use specific character layouts:

**bsfont.png (24x33 pixels per character, 10x6 grid)**
- Row 0: `[NA]![NA][NA][NA]'"()`
- Row 1: `[NA][NA][NA][NA].,0123`
- Row 2: `456789:[NA][NA][NA]`
- Row 3: `[NA]?[NA]ABCDEFG`
- Row 4: `HIJKLMNOPQ`
- Row 5: `RSTUVWXYZ[NA]`

**upfonts.png (33x29 pixels per character, 10x8 grid)**
- Row 0: `[NA]![NA][NA][NA][NA][NA][NA]()`
- Row 1: `[NA][NA][NA][NA].[NA][NA][NA][NA][NA]`
- Row 2: `[NA][NA][NA][NA][NA]#:[NA][NA][NA]`
- Row 3: `[NA]?[NA]ABCDEFG`
- Row 4: `HIJKLMNOPQ`
- Row 5: `RSTUVWXYZ[NA]`

**lfont.png (8x8 pixels per character, 10x7 grid)**
- Row 0: `[NA]![NA][NA][NA][NA][NA]'()`
- Row 1: `[NA][NA][NA][NA]./0123`
- Row 2: `456789:[NA][NA][NA]`
- Row 3: `[NA]?[NA]ABCDEFG`
- Row 4: `HIJKLMNOPQ`
- Row 5: `RSTUVWXYZ[NA]`

### Animation System
- Background movements use sinusoidal functions with different speeds and amplitudes
- Sprite animation creates a "train" effect with 12 sprites following each other
- Each sprite has a phase offset of 0.2 radians
- Text scrolling includes both horizontal and vertical modes
- Vertical scroll moves from bottom to top with proper text ordering
- Raster effects are applied using composite operations

### Sprite Details
The sprites follow a complex trajectory:
- Horizontal movement: `x = 304 + 290 * cos(swing - phase)`
- Vertical movement: `y = 100 + ychange * sin(swingy - phase) + siny`
- Each sprite is 17x10 pixels from the sprite strip
- Sprites are scaled 2x for display

### Audio System
The YM player is integrated with Ebiten's audio system, providing authentic chiptune playback with volume control and looping support.

## Original Credits

- **Original Demo**: The Carebears (TCB)
- **Code**: Olivier Houte aka Bilizir from DMA
- **Graphics**: Tanis
- **Music**: Mad Max
- **Original Platform**: Atari ST (1989)

## Port Credits

This Go/Ebiten port recreates the original demo's effects while adapting to modern hardware capabilities.

## License

This port is created for educational and preservation purposes. The original demo and its assets belong to The Carebears (TCB). The Go port code is provided as-is for learning purposes.

## Notes

- The demo requires all asset files to be present in the `assets/` directory
- Performance may vary depending on your system
- The original demo showcased hardware scrolling techniques that were revolutionary for the Atari ST
- This port aims to preserve the visual experience while using modern rendering techniques

## Historical Context

The original "Grodan and Kvack Kvack" demo was released in 1989 and was notable for being one of the first demos to showcase hardware scrolling on the Atari ST. The demo's name (Swedish for "The Frog and Quack Quack") was self-proclaimed by the creators as "probably the most stupid name in demo history."

The demo was a response to claims that certain effects were impossible on the ST, particularly after seeing the Amiga's TechTech demo. TCB proved that with clever programming, the ST could achieve similar results.
