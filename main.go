package main

import (
	"fmt"
	"os"

	"plexprep/internal/media"
	"plexprep/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Headless: plexprep --analyze <folder>
	if len(os.Args) > 2 && os.Args[1] == "--analyze" {
		if err := ui.AnalyzeReport(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}

	// Headless: plexprep --dry|--run <folder> [profile]
	if len(os.Args) > 2 && (os.Args[1] == "--dry" || os.Args[1] == "--run") {
		prof := media.ProfileZeroTranscode
		if len(os.Args) > 3 {
			switch os.Args[3] {
			case "4k":
				prof = media.Profile4K
			case "audio":
				prof = media.ProfileAudioOnly
			}
		}
		var err error
		if os.Args[1] == "--dry" {
			err = ui.DryRun(os.Args[2], prof)
		} else {
			err = ui.RunHeadless(os.Args[2], prof)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
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
