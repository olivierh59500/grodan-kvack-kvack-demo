# DCK version

This directory contains the construction-kit version of grodan-kvack-kvack-demo. The original Go sources are preserved at their original paths (revision `5338712539ba84170f329ac26403323be754952d`), with small asset accessors so both versions use the same embedded resources.

Run the original with `go run ./cmd/grodan` and this version with `go run ./dck/cmd/grodan` from the repository root.

The choreography and assets remain in this repository. Reusable rendering and
effects come from the published `github.com/olivierh59500/democonstructionkit`
module pinned in `go.mod`. Go downloads the dependencies automatically, including
`github.com/olivierh59500/ym-player v1.0.0` for YM playback. Second Reality retains its original ST3 music synchronization.
