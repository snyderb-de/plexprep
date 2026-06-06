package ui

import (
	"fmt"

	"plexprep/internal/media"
)

// AnalyzeReport prints a folder analysis headlessly.
func AnalyzeReport(root string) error {
	r, err := media.Analyze(root)
	if err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("  📊 ANALYSIS ·", root)
	fmt.Println("  ───────────────────────────────────────────────")
	fmt.Printf("  files            %d", r.Files)
	if r.Files4K > 0 {
		fmt.Printf("  (%d are 4K)", r.Files4K)
	}
	if r.ProbeErrors > 0 {
		fmt.Printf("  [%d unreadable]", r.ProbeErrors)
	}
	fmt.Println()
	fmt.Printf("  source codecs    %s\n", r.CodecSummary())
	fmt.Println()
	fmt.Printf("  ✅ recommended   %s\n", r.Recommended)
	fmt.Printf("     why           %s\n", r.Why)
	fmt.Println()
	fmt.Printf("  work             %d re-encode · %d audio-only · %d already-good\n",
		r.ReencodeCount, r.AudioOnly, r.NoOp)
	fmt.Printf("  space            %s → %s   (save %s, %.0f%%)\n",
		media.HumanBytes(r.OrigBytes), media.HumanBytes(r.ProjBytes),
		media.HumanBytes(r.SavedBytes()), r.SavedPct())
	fmt.Printf("  est. encode time ~%s   (varies with CPU; copies are instant)\n",
		media.HumanDuration(r.EstSecs))
	fmt.Println()
	fmt.Println("  next:  plexprep \"" + root + "\"   (interactive)   ·   --run to batch now")
	fmt.Println()
	return nil
}
