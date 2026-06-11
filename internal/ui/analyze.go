package ui

import (
	"fmt"

	"plexprep/internal/media"
	"plexprep/internal/style"
)

// AnalyzeReport prints a styled folder analysis headlessly.
func AnalyzeReport(root string) error {
	r, err := media.Analyze(root)
	if err != nil {
		return err
	}

	banner(`--analyze "` + root + `"`)

	files := style.Bright.B(fmt.Sprintf("%d", r.Files))
	if r.Files4K > 0 {
		files += style.Mid.S(fmt.Sprintf("  (%d 4K)", r.Files4K))
	}
	if r.ProbeErrors > 0 {
		files += style.Red.S(fmt.Sprintf("  [%d unreadable]", r.ProbeErrors))
	}

	lines := []string{
		style.Mid.S("root      : ") + style.Green.S(root),
		style.Mid.S("files     : ") + files,
		style.Mid.S("codecs    : ") + style.Green.S(r.CodecSummary()),
		style.Mid.S("method    : ") + methodLabel(r.Recommended),
		style.Mid.S("why       : ") + style.Dim.S(style.Trunc(r.Why, 60)),
		style.Mid.S("work      : ") + workLine(r.ReencodeCount, r.AudioOnly, r.NoOp),
		style.Mid.S("size      : ") + style.Green.S(media.HumanBytes(r.OrigBytes)) +
			style.Dim.S(" -> ") + style.Green.S(media.HumanBytes(r.ProjBytes)),
		style.Mid.S("reclaim   : ") + savings(r.SavedBytes(), r.SavedPct(), 10),
		style.Mid.S("est. time : ") + style.Bright.S("~"+media.HumanDuration(r.EstSecs)) +
			style.Dim.S("  (varies w/ CPU; copies instant)"),
	}
	fmt.Println(style.Frame("ANALYSIS", lines))
	fmt.Println()
	if len(r.ProbeErrorDetails) > 0 {
		fmt.Println(style.Red.S("  unreadable files:"))
		for _, pe := range r.ProbeErrorDetails {
			fmt.Println(style.Dim.S("   - ") + style.Mid.S(pe.Path) + style.Dim.S("  ("+style.Trunc(pe.Msg, 80)+")"))
		}
		fmt.Println()
	}
	fmt.Println(style.Dim.S("  // next: ") +
		style.Green.S("plexprep \""+root+"\"") + style.Dim.S(" (interactive)  ·  ") +
		style.Green.S("--dry") + style.Dim.S(" to list files  ·  ") +
		style.Green.S("--run") + style.Dim.S(" to convert"))
	fmt.Println()
	return nil
}
