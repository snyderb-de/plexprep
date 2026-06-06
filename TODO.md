# TODO

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
