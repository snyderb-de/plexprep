# TODO

## Desktop: convert needs an "abort now" in addition to "abort after current"

The status dashboard's `st-abort` only stops *after* the current file
finishes (`Convert`'s loop checks `a.abort` between files). For a long
encode, add an immediate "abort now" that kills the in-flight ffmpeg
process too (cancel the `Encode` context / `cmd.Process.Kill()`) and
discards the partial `tmp` output.

Affects `desktop/app.go` Convert (`a.abort` check + `media.Encode`),
`internal/media/encode.go`, and the status dashboard buttons in
`internal/ui/serve_assets.go` (embedJS `st-abort`).

## Desktop: convert progress bars don't track real progress

During convert, the per-file progress bar (and the overall bar) jump straight
to 100% while the file is still encoding, instead of tracking ffmpeg's actual
`frac`/`speed`/`eta`. Fix the progress reporting so both bars reflect real
encode progress. Also want some extra animation/flourish on the bars (TBD —
design later).

Affects `embedJS` (`st-pbar`/`st-obar`) in `internal/ui/serve_assets.go` and
the `pp:convert` "progress" events emitted from `desktop/app.go` Convert /
`internal/media/encode.go` Encode.

## Audio-only fix bloats the library

Observed on `Z:\TV Shows\True Blood` (80× h264): the Audio-only profile appended
an AAC stereo track to every file and grew the library 194.35 GB → 202.14 GB
(+7.79 GB). Adding AAC unconditionally is wrong when it only costs space.

- [ ] Don't add AAC when the existing audio already direct-plays widely
      (e.g. the file already has an AAC track, or AC3/E-AC3 stereo that most
      clients handle). Only add when the audio is genuinely incompatible
      (DTS/TrueHD/PCM/multichannel-only) AND a stereo fallback is missing.
- [ ] Make the AAC fallback bitrate sane and/or only mux a *stereo downmix*,
      not a full re-encode that can exceed the source audio size.
- [ ] When a plan's projected size grows, flag it clearly in analyze/dry and
      consider auto-marking it "keep" unless the user opts in.

## Probe failures are silent

Same run reported `80 files [32 unreadable]`. Root cause found: all 32 were
macOS AppleDouble sidecars (`._Episode.mkv`) — junk written on a non-Mac drive,
carrying the `.mkv` extension, which ffprobe rejects with
"invalid as first byte of an EBML number".

- [x] Skip dotfiles / `._*` AppleDouble sidecars in `FindVideos` (resolved).
- [ ] Still surface real per-file probe errors with `--verbose`: path +
      ffprobe stderr, so genuinely unreadable files can be diagnosed.
- [ ] Confirm remaining failures aren't `ffprobe` timeouts or long/UNC Windows
      paths; add a timeout + clearer handling.

## HTML report — sortable columns

- [x] Click-to-sort columns (asc/desc, arrow indicator). Numeric columns sort
      by raw `data-sort` value (bytes/seconds), not the human string.
      Dependency-free inline JS in `internal/ui/report.go` (sortJS).

## Linux / macOS support
plexprep is currently developed and tested on Windows only. Port + verify on Unix:

- [ ] Verify `ffmpeg`/`ffprobe` discovery works on Linux/macOS (PATH lookup is
      already cross-platform via `exec.Command`, but confirm).
- [ ] Test path handling with `/` roots and spaces (e.g. `/mnt/media/...`,
      `/Volumes/...`). `filepath` is OS-aware, but exercise it.
- [ ] Re-check the encode-time throughput constants in `internal/media/analyze.go`
      — they were calibrated on one Windows desktop CPU. Consider a quick
      self-calibration sample encode instead of hard-coded `tputX264Slow`, etc.
- [x] CI build matrix (GitHub Actions) producing release binaries for
      windows/amd64, linux/amd64, darwin/amd64, darwin/arm64.
      (`.github/workflows/release.yml`, triggered by `v*` tags;
      `ci.yml` runs vet+build on push/PR.)
- [ ] Smoke-test the Bubble Tea TUI under common Unix terminals (iTerm2, GNOME
      Terminal, Alacritty) — colors/altscreen/gradient.
- [ ] Document install via `go install` and via downloaded release binaries.
