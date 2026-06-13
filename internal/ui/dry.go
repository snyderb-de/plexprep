package ui

import (
	"fmt"
	"path/filepath"

	"plexprep/internal/media"
	"plexprep/internal/style"
)

// shortMethod returns a compact codec tag for a plan.
func shortMethod(it *media.Item) string {
	if it.Plan.NoOp() {
		return style.Green.S("keep")
	}
	if !it.Plan.ReencodeVideo {
		return style.Blue.S("+aac")
	}
	return style.Amber.S(it.Plan.TargetCodec[3:]) // strip "lib"
}

// DryRun prints a styled per-file plan for a folder (no encoding).
func DryRun(root string, profile media.Profile) error {
	paths, err := media.FindVideos(root)
	if err != nil {
		return err
	}
	return DryRunPaths(fmt.Sprintf(`"%s"`, root), paths, profile)
}

// DryRunPaths prints a styled per-file plan for an explicit list of files.
func DryRunPaths(label string, paths []string, profile media.Profile) error {
	banner(fmt.Sprintf(`--dry %s`, label))

	if len(paths) == 0 {
		fmt.Println(style.Frame("DRY RUN", []string{style.Amber.S("no video files to preview")}))
		fmt.Println()
		return nil
	}

	// Column widths.
	const wName, wCodec, wSize, wEst, wBar = 38, 11, 10, 10, 22

	header := style.Mid.S(
		style.Pad("FILE", wName) + " " +
			style.Pad("CODEC", wCodec) + " " +
			padR("SIZE", wSize) + " " +
			padR("EST", wEst) + "  " +
			style.Pad("SAVING", wBar) + " ACTION")

	var rows []string
	rows = append(rows, header)
	rows = append(rows, style.Rule(wName+wCodec+wSize+wEst+wBar+30))

	var origTot, projTot int64
	for _, p := range paths {
		it, err := media.BuildItem(p, profile, 0)
		if err != nil {
			rows = append(rows, style.Red.S(style.Trunc(filepath.Base(p), wName)+"  probe error"))
			continue
		}
		origTot += it.Info.SizeBytes
		projTot += it.Plan.ProjectedBytes

		name := style.Trunc(filepath.Base(p), wName)
		if it.Is4K {
			name = style.Trunc(filepath.Base(p), wName-3) + " " + style.Amber.S("4K")
		}
		var save string
		if it.Plan.NoOp() {
			save = style.Pad(style.Dim.S("— already optimal"), wBar)
		} else {
			save = style.Pad(savings(it.Plan.SavedBytes(), it.Plan.SavedPct(), 8), wBar)
		}
		rows = append(rows, fmt.Sprintf("%s %s %s %s  %s %s",
			style.Pad(style.Bright.S(name), wName),
			style.Pad(style.Green.S(it.Info.Video.CodecName), wCodec),
			style.PadL(style.Green.S(media.HumanBytes(it.Info.SizeBytes)), wSize),
			style.PadL(style.Green.S(media.HumanBytes(it.Plan.ProjectedBytes)), wEst),
			save,
			shortMethod(it)))
	}

	fmt.Println(style.Frame(fmt.Sprintf("DRY RUN · %s · %d files", profile, len(paths)), rows))

	saved := origTot - projTot
	fmt.Println()
	fmt.Printf("  %s %s %s %s   %s\n",
		style.Mid.S("TOTAL"),
		style.Green.S(media.HumanBytes(origTot)),
		style.Dim.S("->"),
		style.Green.S(media.HumanBytes(projTot)),
		savings(saved, pctOf(saved, origTot), 12))
	fmt.Println()
	return nil
}

func padR(s string, w int) string { return style.PadL(s, w) }

func pctOf(a, b int64) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b) * 100
}
