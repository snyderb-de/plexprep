package ui

import (
	"fmt"
	"path/filepath"

	"plexprep/internal/media"
)

// DryRun prints the conversion plan for a folder without launching the TUI.
func DryRun(root string, profile media.Profile) error {
	paths, err := media.FindVideos(root)
	if err != nil {
		return err
	}
	fmt.Printf("plexprep dry-run · %s · profile: %s\n", root, profile)
	fmt.Printf("%d video file(s)\n\n", len(paths))
	fmt.Printf("%-40s %-10s %10s %10s %7s  %s\n", "FILE", "CODEC", "SIZE", "EST", "SAVE", "ACTION")

	var origTot, projTot int64
	for _, p := range paths {
		it, err := media.BuildItem(p, profile)
		if err != nil {
			fmt.Printf("%-40s  probe error: %v\n", truncate(filepath.Base(p), 40), err)
			continue
		}
		origTot += it.Info.SizeBytes
		projTot += it.Plan.ProjectedBytes
		save := "n/a"
		if !it.Plan.NoOp() {
			save = fmt.Sprintf("%.0f%%", it.Plan.SavedPct())
		}
		tag := ""
		if it.Is4K {
			tag = "[4K] "
		}
		fmt.Printf("%-40s %-10s %10s %10s %7s  %s%v\n",
			truncate(filepath.Base(p), 40),
			it.Info.Video.CodecName,
			media.HumanBytes(it.Info.SizeBytes),
			media.HumanBytes(it.Plan.ProjectedBytes),
			save, tag, it.Plan.Reasons)
	}
	fmt.Printf("\nTOTAL  %s → %s   save %s (%.0f%%)\n",
		media.HumanBytes(origTot), media.HumanBytes(projTot),
		media.HumanBytes(origTot-projTot),
		pctOf(origTot-projTot, origTot))
	return nil
}

func pctOf(a, b int64) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b) * 100
}
