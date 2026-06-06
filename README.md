# plexprep ✨

Zero-transcode media forge — a Bubble Tea TUI that batch-converts a folder of
video into Plex/Jellyfin **direct-play** files, with a size-savings preview
*before* you commit.

## Philosophy

Don't re-encode everything — that wastes time and loses quality. plexprep only
touches what actually forces a server transcode:

- **Legacy video** (MPEG-2, VC-1, WMV, old MPEG-4/DivX) → re-encoded to modern codec.
- **Modern video** (H.264, and HEVC in 4K mode) → **copied untouched** (no quality loss).
- **Audio** → original kept lossless; an **AAC stereo** track is appended so
  browsers/phones never transcode the audio.
- Interlaced sources are auto-deinterlaced (yadif).

## Profiles

| Profile | Video | When |
|---|---|---|
| **Zero-Transcode (SD/HD)** | x264 CRF18 for legacy, else copy | default, most libraries |
| **4K UHD** | x265/HEVC CRF20 for legacy, keep HEVC | 2160p content |
| **Audio-only fix** | always copy video, just add AAC | already-modern video that only transcodes for audio |

## Usage

Interactive TUI:

```
plexprep.exe "Z:\TV Shows\Baseball (1994)"
```

Pick a profile → review the per-file savings table (space toggles, `a`/`n`
select all/none) → Enter to convert. By default outputs land beside originals
as `… (plexprep).mkv` and originals are untouched.

### Optional in-place replace

Press `r` on the review screen (or pass `--replace` headlessly) to make each
output **take the source's name** (as `.mkv`). The source is renamed to
`<name>.original` as a backup — it is never deleted, so you can eyeball the
result and purge the `.original` files yourself once happy. The encode is
written to a temp file and swapped in only on success, so a cancel or failure
never destroys the source.

The interactive flow opens with an **Analysis** screen: recommended method,
total space savings, and an estimated encode time — accept it (Enter) or pick a
method yourself (`c`).

Headless (scripting / no TTY):

```
plexprep.exe --analyze "Z:\path"           # recommend method + savings + time estimate
plexprep.exe --dry "Z:\path" [4k|audio]    # per-file preview, no encoding
plexprep.exe --run "Z:\path" [4k|audio]    # convert, plain-text progress
plexprep.exe --run "Z:\path" --replace     # convert in place (source → .original)
```

## Build

```
go build -o plexprep.exe .
```

Requires `ffmpeg` + `ffprobe` on PATH.
