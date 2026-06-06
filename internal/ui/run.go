package ui

import (
	"context"
	"fmt"
	"path/filepath"

	"plexprep/internal/media"
)

// RunHeadless converts every non-noop file in root under profile, printing
// plain progress. Useful for scripting / no-TTY environments.
func RunHeadless(root string, profile media.Profile) error {
	paths, err := media.FindVideos(root)
	if err != nil {
		return err
	}
	ctx := context.Background()
	var savedTot int64
	var ok, fail int

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
		out := media.OutputPath(p)
		fmt.Printf("▸ %s  (%v)\n", filepath.Base(p), it.Plan.Reasons)

		var last float64
		for pr := range media.Encode(ctx, it.Info, it.Plan, out) {
			if pr.Err != nil {
				fmt.Printf("  ✘ %v\n", pr.Err)
				fail++
				break
			}
			if pr.Done {
				if ni, e := media.Probe(out); e == nil {
					saved := it.Info.SizeBytes - ni.SizeBytes
					savedTot += saved
					fmt.Printf("  ✓ %s → %s  (saved %s)\n",
						media.HumanBytes(it.Info.SizeBytes),
						media.HumanBytes(ni.SizeBytes),
						media.HumanBytes(saved))
				}
				ok++
				break
			}
			if pr.Fraction-last >= 0.10 {
				last = pr.Fraction
				fmt.Printf("  … %.0f%% @ %s\n", pr.Fraction*100, pr.Speed)
			}
		}
	}
	fmt.Printf("\nDone. %d converted, %d failed, reclaimed %s\n", ok, fail, media.HumanBytes(savedTot))
	return nil
}
