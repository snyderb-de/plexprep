package main

import (
	"fmt"
	"os"
	"strings"

	"plexprep/internal/media"
	"plexprep/internal/style"
	"plexprep/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

// printUsage renders the help in the phosphor-terminal style.
func printUsage() {
	g, a, b, d, br := style.Green, style.Amber, style.Blue, style.Dim, style.Bright
	fmt.Println()
	fmt.Println("  " + br.B("plexprep") + d.S(" — zero-transcode media forge"))
	fmt.Println("  " + d.S("Batch-convert video into Plex/Jellyfin direct-play files."))
	fmt.Println("  " + d.S("Re-encodes only legacy video; modern video is copied. Audio kept + AAC fallback."))
	fmt.Println()
	cmd := func(c, desc string) string { return "  " + g.S(style.Pad(c, 46)) + d.S(desc) }
	fmt.Println(style.Frame("USAGE", []string{
		cmd("plexprep [folder]", "launch the interactive TUI"),
		cmd("plexprep --analyze <folder>", "recommend method + savings + time"),
		cmd("plexprep --report  <root> [out]", "per-subfolder report → .xlsx + .html"),
		cmd("plexprep --dry     <targets…> [p]", "per-file preview, no encoding"),
		cmd("plexprep --run     <targets…> [p] [--replace]", "convert headlessly, live progress"),
		cmd("plexprep --help | -h", "show this help"),
	}))
	fmt.Println()
	fmt.Println(style.Frame("TARGETS  (--dry/--run)", []string{
		d.S("one or more ") + br.S("folders") + d.S(" (walked) and/or ") + br.S("files"),
		d.S("--from <list>") + d.S("  read newline-separated paths from a file (# comments ok)"),
		d.S("e.g. ") + g.S(`plexprep --run --replace A.mkv B.mkv`),
		d.S("e.g. ") + g.S(`plexprep --run --replace --from shrink.txt`),
	}))
	fmt.Println()
	prof := func(k, name, desc string) string {
		return "  " + a.S(style.Pad(k, 8)) + br.S(style.Pad(name, 22)) + d.S(desc)
	}
	fmt.Println(style.Frame("PROFILE [p]  (--dry/--run; default zero-transcode)", []string{
		prof("(none)", "Zero-Transcode", "x264 CRF18 for legacy, copy modern"),
		prof("4k", "4K UHD", "x265/HEVC CRF20 for legacy, keep HEVC"),
		prof("audio", "Audio-only fix", "copy video, add AAC stereo only"),
	}))
	fmt.Println()
	fmt.Println(style.Frame("OUTPUT", []string{
		d.S("default  : sibling ") + g.S(`"… (plexprep).mkv"`) + d.S(", source untouched"),
		d.S("--replace: output takes source name; source → ") + a.S(".original") + d.S(" backup"),
		d.S("--delete : remove each original right after it converts ") + style.Red.S("(irreversible)"),
		d.S("           frees space mid-batch · TUI toggles: ") + b.S("r") + d.S(" replace · ") + b.S("d") + d.S(" delete"),
		d.S("needs    : ") + br.S("ffmpeg") + d.S(" + ") + br.S("ffprobe") + d.S(" on PATH"),
	}))
	fmt.Println()
}

const usage = `plexprep — zero-transcode media forge ✨

Batch-convert a folder of video into Plex/Jellyfin direct-play files.
Only re-encodes legacy video (MPEG-2/VC-1/WMV/…); modern video is copied.
Original audio is kept lossless and an AAC stereo fallback is appended.

USAGE
  plexprep [folder]                 launch the interactive TUI
                                    (opens on an analysis of [folder] if given)
  plexprep --analyze <folder>       recommend a method + savings + time estimate
  plexprep --report  <root> [out]   analyze each 1st-level subfolder → .xlsx + .html
  plexprep --dry     <targets…> [p]          per-file preview table, no encoding
  plexprep --run     <targets…> [p] [--replace] [--delete]  convert headlessly
  plexprep --help | -h                       show this help

TARGETS  (--dry / --run)
  <targets…> is one or more folders (walked recursively) and/or individual
  files, in any order. Add --from <list> to read newline-separated paths from
  a file (blank lines and # comments are ignored). Examples:
    plexprep --run --replace "A.mkv" "B.mkv"
    plexprep --run --replace --from shrink.txt

PROFILE [p]  (headless --dry/--run only; default: zero-transcode)
  (none)   Zero-Transcode (SD/HD)   x264 CRF18 for legacy, copy modern
  4k       4K UHD                    x265/HEVC CRF20 for legacy, keep HEVC
  audio    Audio-only fix            copy video, just add AAC stereo

OUTPUT
  • Default: written beside the original as "… (plexprep).mkv"; source untouched.
  • --replace (optional): output takes the source's name (as .mkv) and the
    source is renamed to "<name>.original" as a backup.
  • --delete (optional): remove each original right after it converts, freeing
    space mid-batch. Irreversible. The source is only removed once the new file
    is safely in place. TUI toggles: "r" replace, "d" delete.

NOTES
  • Requires ffmpeg and ffprobe on PATH.
`

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help", "-h", "help":
			printUsage()
			return
		}
	}

	// Headless folder report: plexprep --report <root> [output-basename]
	if len(os.Args) > 1 && os.Args[1] == "--report" {
		root, out := "", ""
		for _, a := range os.Args[2:] {
			if strings.HasPrefix(a, "--") {
				continue
			}
			if root == "" {
				root = a
			} else if out == "" {
				out = a
			}
		}
		if root == "" {
			fmt.Fprintf(os.Stderr, "error: --report needs a root folder\n\n%s", usage)
			os.Exit(2)
		}
		if err := ui.ReportFolders(root, out); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}

	// Headless modes: flags and the folder may appear in any order.
	if len(os.Args) > 1 {
		mode := os.Args[1]
		if mode == "--analyze" || mode == "--dry" || mode == "--run" {
			prof := media.ProfileZeroTranscode
			replace := false
			purge := false
			listFile := ""
			var targets []string
			args := os.Args[2:]
			for i := 0; i < len(args); i++ {
				a := args[i]
				switch a {
				case "4k":
					prof = media.Profile4K
				case "audio":
					prof = media.ProfileAudioOnly
				case "--replace":
					replace = true
				case "--delete", "--purge":
					purge = true
				case "--from":
					if i+1 < len(args) {
						i++
						listFile = args[i]
					}
				default:
					if !strings.HasPrefix(a, "--") {
						targets = append(targets, a)
					}
				}
			}

			// Build the working set: explicit files/dirs + an optional --from list.
			if listFile != "" {
				lines, err := media.ReadListFile(listFile)
				if err != nil {
					fmt.Fprintln(os.Stderr, "error: --from:", err)
					os.Exit(1)
				}
				targets = append(targets, lines...)
			}
			if len(targets) == 0 {
				fmt.Fprintf(os.Stderr, "error: %s needs a folder, one or more files, or --from <list>\n\n%s", mode, usage)
				os.Exit(2)
			}
			paths, err := media.ResolveTargets(targets)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}

			// Label for the banner: a lone folder shows its path, otherwise a count.
			label := fmt.Sprintf("%d files", len(paths))
			if len(targets) == 1 {
				label = fmt.Sprintf("%q", targets[0])
			}

			switch mode {
			case "--analyze":
				// --analyze stays folder-oriented (one root).
				err = ui.AnalyzeReport(targets[0])
			case "--dry":
				err = ui.DryRunPaths(label, paths, prof)
			case "--run":
				err = ui.RunHeadlessPaths(label, paths, prof, replace, purge)
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
