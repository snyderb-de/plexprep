package ui

import (
	"context"
	"fmt"
	"path/filepath"

	"plexprep/internal/media"
)

// RunHeadless converts every non-noop file in root under profile, printing
// plain progress. Useful for scripting / no-TTY environments.
//
// When replace is true, each output takes its source's name (as .mkv) and the
// source is renamed to "<name>.original" as a backup. Otherwise the output is
// written alongside as "<name> (plexprep).mkv" and the source is untouched.
func RunHeadless(root string, profile media.Profile, replace bool) error {
	paths, err := media.FindVideos(root)
	if err != nil {
		return err
	}
	ctx := context.Background()
	var savedTot int64
	var ok, fail int
	if replace {
		fmt.Println("replace mode: sources backed up as “.original”")
	}

	for _, p := range paths {
		it, err := media.BuildItem(p, profile)
		if err != nil {
			fmt.Printf("✘ %s: probe error: %v\n", filepath.Base(p), err)
			fail++
			continue
		}
		if it.Plan.NoOp() {
			fmt.Printf("• %s: already optimal, skip\n", filepath.Base(p))
			continue
		}
		tmp := media.TempPath(p)
		fmt.Printf("▸ %s  (%v)\n", filepath.Base(p), it.Plan.Reasons)

		failed := false
		var last float64
		for pr := range media.Encode(ctx, it.Info, it.Plan, tmp) {
			if pr.Err != nil {
				fmt.Printf("  ✘ %v\n", pr.Err)
				fail++
				failed = true
				break
			}
			if pr.Done {
				break
			}
			if pr.Fraction-last >= 0.10 {
				last = pr.Fraction
				fmt.Printf("  … %.0f%% @ %s\n", pr.Fraction*100, pr.Speed)
			}
		}
		if failed {
			continue
		}

		final, err := media.Finalize(p, tmp, replace)
		if err != nil {
			fmt.Printf("  ✘ finalize: %v\n", err)
			fail++
			continue
		}
		if ni, e := media.Probe(final); e == nil {
			saved := it.Info.SizeBytes - ni.SizeBytes
			savedTot += saved
			fmt.Printf("  ✓ %s → %s  (saved %s)\n",
				media.HumanBytes(it.Info.SizeBytes),
				media.HumanBytes(ni.SizeBytes),
				media.HumanBytes(saved))
		}
		ok++
	}
	fmt.Printf("\nDone. %d converted, %d failed, reclaimed %s\n", ok, fail, media.HumanBytes(savedTot))
	return nil
}
