# DCK version

This directory contains the construction-kit version of grodan-kvack-kvack-demo. The original Go sources are preserved at their original paths (revision `5338712539ba84170f329ac26403323be754952d`), with small asset accessors so both versions use the same embedded resources.

Run the original with `go run ./cmd/grodan` and this version with `go run ./dck/cmd/grodan` from the repository root.

The choreography and assets remain in this repository. Reusable rendering and
effects come from the published `github.com/olivierh59500/democonstructionkit`
module pinned in `go.mod`. Music is opened with `sound.Open`; DCK selects the decoder from the asset and
provides the configured stereo PCM format. The demo keeps its playback level and loop settings.

The twelve sprite frames are drawn by `sprites.Group` with the reusable
`motion.HarmonicFormation` preset. Two phase clocks and one bouncing amplitude
envelope preserve the authored train while letting another screen change its
image bank, sprite count, spacing, phase rates or wave amplitudes.

The big, vertical and two small text lanes now use the same
`scrolling.Config.Ribbon` transport. `presets.GrodanRibbons` keeps their
independent fonts, strict return thresholds, speeds, scales and placement
editable. The original four messages remain in `dck/internal/textdata`.

The big, vertical and paired small ribbons now use three
`composite.SurfaceLayer` instances. Each layer owns one bounded canvas and
combines its text with editable raster passes and output copies; the two small
scrolls keep their shared canvas and original draw order.

The two repeated backdrops use `effects.GatedBackgroundPair`. Its preset
retains the first background's gated horizontal bounce and the second
background's coupled X/Y boundary changes. A pure test checks the authored
positions over 50,000 ticks; both images and motion limits are editable.
