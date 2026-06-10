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
`<name>.original` as a backup. The encode is written to a temp file and swapped
in only on success, so a cancel or failure never destroys the source.

### Optional delete-after-convert

Press `d` (or pass `--delete`) to **remove each original the moment its
conversion succeeds**, freeing disk space as the batch runs instead of letting
backups pile up to the end. Irreversible — the source is only ever removed once
the new file is safely in place. Combine with `--replace` to convert in place
with no leftover backups.

The interactive flow opens with an **Analysis** screen: recommended method,
total space savings, and an estimated encode time — accept it (Enter) or pick a
method yourself (`c`).

### Web UI (`--serve`)

```
plexprep.exe --serve                 # opens browser, pick a target to scan
plexprep.exe --serve "Y:\Movies"     # scan straight into the report
plexprep.exe --serve --port 8080     # choose the port (default 7777)
```

Starts a local server (binds `127.0.0.1` only) and opens your browser:

1. **Pick** a folder or file (server-side directory browser) with a
   recursive-scan toggle.
2. **Scan** runs async with a live progress bar — the page never blocks on a
   whole-tree probe.
3. **Select** files in the interactive report (same drill-down, filter, and
   checkboxes as the static report).
4. **Convert** — choose profile + `replace`/`delete`, then watch a live status
   dashboard (per-file + overall bars, speed, ETA, running reclaim, log).

Conversion runs server-side using the same engine as `--run`; output rules are
identical (sibling `(plexprep).mkv` by default, or in-place with `replace`).

Headless (scripting / no TTY):

```
plexprep.exe --analyze "Z:\path"            # recommend method + savings + time estimate
plexprep.exe --report  "Z:\library" [out]   # per-subfolder summary → .xlsx + .html
plexprep.exe --dry  <targets…> [4k|audio]   # per-file preview, no encoding
plexprep.exe --run  <targets…> [4k|audio]   # convert, plain-text progress
plexprep.exe --run  <targets…> --replace    # convert in place (source → .original)
plexprep.exe --run  --from list.txt --replace   # convert a hand-picked list
```

`<targets…>` for `--dry`/`--run` is any mix of folders (walked recursively)
and individual files, in any order; `--from <list>` adds newline-separated
paths from a file (blank lines and `#` comments ignored).

### Folder report (`--report`)

Point it at a library root; it analyzes every 1st-level subfolder (recursing
through all their content) plus the loose files in the root (a `(root)` row),
and writes a one-row-per-folder summary as both `plexprep-report.xlsx` and
`plexprep-report.html` — a retro phosphor-terminal dashboard with per-folder
recommended method, space savings, and estimated encode time. Pass an optional
2nd argument to choose the output basename/location.

The HTML report is interactive (pure CSS where possible, no external deps):

- **Drill-down** — click a folder to see its per-file analysis; `:target`-based
  so it works even with JavaScript disabled.
- **Filter** — `all` / `shrinks` / `grows` buttons hide rows by outcome and
  recompute the SUMMARY box live, so you see the real reclaim for just the
  rows you'd convert. Works in the summary and inside each folder panel.
- **Convert-list builder** — check off files (or whole folders, or "select
  shown" to grab the current filter), then **Download .txt** / **Copy paths**.
  The export is exactly the `--from` format, so:

  ```
  plexprep.exe --run --replace --from plexprep-convert.txt
  ```

## Build

```
go build -o plexprep.exe .
```

Requires `ffmpeg` + `ffprobe` on PATH.
