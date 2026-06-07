package ui

import (
	"fmt"

	"plexprep/internal/media"
	"plexprep/internal/style"
)

// banner prints the shell-prompt line shared by all console outputs.
func banner(cmd string) {
	fmt.Println()
	fmt.Printf("%s%s%s %s\n",
		style.Amber.B("bag@plexprep"),
		style.Mid.S(":"),
		style.Mid.S("~$"),
		style.Bright.S("plexprep "+cmd))
	fmt.Println()
}

// savings renders the bar + figure for a saving (or growth) at the given bar width.
func savings(savedBytes int64, pct float64, barW int) string {
	if savedBytes < 0 {
		return style.Red.S("▲ +"+media.HumanBytes(-savedBytes)) + style.Red.S(" larger")
	}
	return fmt.Sprintf("%s %s %s",
		style.Bar(pct, barW),
		style.Tier(pct).B(fmt.Sprintf("%.0f%%", pct)),
		style.Dim.S(media.HumanBytes(savedBytes)))
}

// methodLabel colors a recommended-method string.
func methodLabel(p media.Profile) string {
	return style.Amber.S(p.String())
}

// workLine renders the re-encode / add-AAC / keep counts, colored.
func workLine(reenc, audio, keep int) string {
	return fmt.Sprintf("%s re-encode · %s add-AAC · %s keep",
		style.Amber.B(fmt.Sprintf("%d", reenc)),
		style.Blue.B(fmt.Sprintf("%d", audio)),
		style.Green.B(fmt.Sprintf("%d", keep)))
}
