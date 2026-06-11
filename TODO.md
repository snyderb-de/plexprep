# TODO

## Desktop: convert progress bar — extra animation/flourish (TBD)

The per-file/overall progress bars now track real ffmpeg progress (fixed: a
`-progress` line was double-counted via `out_time_us`+`out_time_ms`, snapping
the bar to 100% instantly). Still want some extra animation/flourish on the
bars — design later.

Affects `.meterfill` in `desktop/frontend/src/style.css` / `internal/ui/report.go`.

## Audio-only fix bloats the library

Observed on `Z:\TV Shows\True Blood` (80× h264): the Audio-only profile appended
an AAC stereo track to every file and grew the library 194.35 GB → 202.14 GB
(+7.79 GB).

- [x] Don't add AAC when the existing audio already direct-plays widely
      (AAC, AC3, E-AC3, MP3 stereo-or-less). Only add when audio is
      genuinely incompatible (DTS/TrueHD/PCM/multichannel-only).
- [x] AAC fallback is a 192k stereo *downmix* (`-ac:a:N 2`), not a full
      multichannel re-encode.
- [x] When a plan's projected size grows, flag it in the file's `Why`
      reasons (`⚠ grows by ~X`).
- [ ] "Goal"-based profile selection (picked at scan time, not just a fixed
      profile): zero-transcoding (current default), smallest file size
      without quality loss (allows transcoding), 4K, device-targeted
      (e.g. Nvidia Shield compat). Bigger feature — needs its own design
      pass on `media.Profile` / the scan UI before implementation.

## Probe failures are silent

- [x] Skip dotfiles / `._*` AppleDouble sidecars in `FindVideos` (resolved).
- [x] Surface real per-file probe errors: ffprobe stderr is now captured
      (`internal/media/probe.go`) and listed under "unreadable files" in
      `--analyze` output (`internal/ui/analyze.go`), via the new
      `Report.ProbeErrorDetails`.
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

NOTE: user now has Linux and macOS machines available — these items can be
tested directly whenever, no longer blocked on access.
