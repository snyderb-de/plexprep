package ui

import (
	"context"
	"fmt"
	"path/filepath"

	"plexprep/internal/media"
	"plexprep/internal/style"
)

// RunHeadless converts every non-noop file in root under profile with a styled,
// live progress display.
//
// When replace is true, each output takes its source's name (as .mkv) and the
// source is renamed to "<name>.original" as a backup. Otherwise the output is
// written alongside as "<name> (plexprep).mkv" and the source is untouched.
func RunHeadless(root string, profile media.Profile, replace, purge bool) error {
	paths, err := media.FindVideos(root)
	if err != nil {
		return err
	}

	flags := ""
	if replace {
		flags += " --replace"
	}
	if purge {
		flags += " --delete"
	}
	banner(fmt.Sprintf(`--run "%s"%s`, root, flags))

	if len(paths) == 0 {
		fmt.Println(style.Frame("RUN", []string{style.Amber.S("no video files found under " + root)}))
		fmt.Println()
		return nil
	}
	if purge {
		fmt.Println(style.Red.B("  ⚠ delete mode — each original is REMOVED after it converts (irreversible)") + "\n")
	} else if replace {
		fmt.Println(style.Dim.S("  replace mode — sources backed up as ") + style.Amber.S(".original") + "\n")
	}

	ctx := context.Background()
	var savedTot int64
	var ok, fail, skip int

	for i, p := range paths {
		name := filepath.Base(p)
		idx := style.Mid.S(fmt.Sprintf("[%d/%d]", i+1, len(paths)))

		it, err := media.BuildItem(p, profile)
		if err != nil {
			fmt.Printf("  %s %s %s\n", idx, style.Red.S("✘"), style.Red.S(style.Trunc(name, 50)+" — probe error"))
			fail++
			continue
		}
		if it.Plan.NoOp() {
			fmt.Printf("  %s %s %s %s\n", idx, style.Green.S("●"), style.Bright.S(style.Trunc(name, 50)), style.Dim.S("already optimal"))
			skip++
			continue
		}

		tmp := media.TempPath(p)
		shortName := style.Trunc(name, 40)

		failed := false
		for pr := range media.Encode(ctx, it.Info, it.Plan, tmp) {
			if pr.Err != nil {
				fmt.Printf("\r  %s %s %-60s\n", idx, style.Red.S("✘"), style.Red.S(style.Trunc(name, 50)+" — "+style.Trunc(pr.Err.Error(), 40)))
				fail++
				failed = true
				break
			}
			if pr.Done {
				break
			}
			// Live, in-place progress.
			fmt.Printf("\r  %s %s %s %s %s   ",
				idx, style.Amber.S("▸"), style.Pad(style.Bright.S(shortName), 40),
				style.ProgBar(pr.Fraction*100, 16),
				style.Tier(pr.Fraction*100).B(fmt.Sprintf("%3.0f%% %s", pr.Fraction*100, style.Dim.S("@ "+pr.Speed))))
		}
		if failed {
			continue
		}

		final, err := media.Finalize(p, tmp, replace, purge)
		if err != nil {
			fmt.Printf("\r  %s %s finalize: %v%-20s\n", idx, style.Red.S("✘"), err, "")
			fail++
			continue
		}
		ni, _ := media.Probe(final)
		saved := int64(0)
		if ni != nil {
			saved = it.Info.SizeBytes - ni.SizeBytes
			savedTot += saved
		}
		pct := pctOf(saved, it.Info.SizeBytes)
		fmt.Printf("\r  %s %s %s %s %s%-10s\n",
			idx, style.Green.B("✓"), style.Pad(style.Bright.S(shortName), 40),
			style.Green.S(media.HumanBytes(it.Info.SizeBytes))+style.Dim.S(" -> ")+style.Green.S(media.HumanBytes(ni.SizeBytes)),
			savings(saved, pct, 8), "")
		ok++
	}

	lines := []string{
		style.Mid.S("converted : ") + style.Green.B(fmt.Sprintf("%d", ok)),
		style.Mid.S("skipped   : ") + style.Bright.S(fmt.Sprintf("%d", skip)) + style.Dim.S("  (already optimal)"),
	}
	if fail > 0 {
		lines = append(lines, style.Mid.S("failed    : ")+style.Red.B(fmt.Sprintf("%d", fail)))
	}
	lines = append(lines, style.Mid.S("reclaimed : ")+style.Bright.B(media.HumanBytes(savedTot)))
	fmt.Println()
	fmt.Println(style.Frame("DONE", lines))
	fmt.Println()
	return nil
}
