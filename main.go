package main

import (
	"fmt"
	"os"
	"strings"

	"plexprep/internal/media"
	"plexprep/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

const usage = `plexprep — zero-transcode media forge ✨

Batch-convert a folder of video into Plex/Jellyfin direct-play files.
Only re-encodes legacy video (MPEG-2/VC-1/WMV/…); modern video is copied.
Original audio is kept lossless and an AAC stereo fallback is appended.

USAGE
  plexprep [folder]                 launch the interactive TUI
                                    (opens on an analysis of [folder] if given)
  plexprep --analyze <folder>       recommend a method + savings + time estimate
  plexprep --dry     <folder> [p]            per-file preview table, no encoding
  plexprep --run     <folder> [p] [--replace] convert headlessly, plain progress
  plexprep --help | -h                       show this help

PROFILE [p]  (headless --dry/--run only; default: zero-transcode)
  (none)   Zero-Transcode (SD/HD)   x264 CRF18 for legacy, copy modern
  4k       4K UHD                    x265/HEVC CRF20 for legacy, keep HEVC
  audio    Audio-only fix            copy video, just add AAC stereo

OUTPUT
  • Default: written beside the original as "… (plexprep).mkv"; source untouched.
  • --replace (optional): output takes the source's name (as .mkv) and the
    source is renamed to "<name>.original" as a backup (never deleted).
    In the TUI, toggle this on the review screen with the "r" key.

NOTES
  • Requires ffmpeg and ffprobe on PATH.
`

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help", "-h", "help":
			fmt.Print(usage)
			return
		}
	}

	// Headless modes: flags and the folder may appear in any order.
	if len(os.Args) > 1 {
		mode := os.Args[1]
		if mode == "--analyze" || mode == "--dry" || mode == "--run" {
			prof := media.ProfileZeroTranscode
			replace := false
			folder := ""
			for _, a := range os.Args[2:] {
				switch a {
				case "4k":
					prof = media.Profile4K
				case "audio":
					prof = media.ProfileAudioOnly
				case "--replace":
					replace = true
				default:
					if folder == "" && !strings.HasPrefix(a, "--") {
						folder = a
					}
				}
			}
			if folder == "" {
				fmt.Fprintf(os.Stderr, "error: %s needs a folder\n\n%s", mode, usage)
				os.Exit(2)
			}
			var err error
			switch mode {
			case "--analyze":
				err = ui.AnalyzeReport(folder)
			case "--dry":
				err = ui.DryRun(folder, prof)
			case "--run":
				err = ui.RunHeadless(folder, prof, replace)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			return
		}
	}

	start := ""
	if len(os.Args) > 1 {
		start = os.Args[1]
	}

	p := tea.NewProgram(ui.New(start), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
