package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"plexprep/internal/media"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.err != nil {
		return "\n" + boxStyle.BorderForeground(cRed).Render(
			errStyle.Render("✘ error")+"\n\n"+m.err.Error()) +
			"\n" + helpStyle.Render("  q / ctrl+c to quit") + "\n"
	}

	header := titleStyle.Render("✨ PLEXPREP ✨") + "  " +
		subtitleStyle.Render(gradientText("zero-transcode media forge"))

	var body string
	switch m.state {
	case statePath:
		body = m.viewPath()
	case stateAnalyze:
		body = m.viewAnalyze()
	case stateProfile:
		body = m.viewProfile()
	case stateScanning:
		body = m.viewScanning()
	case stateReview:
		body = m.viewReview()
	case stateConvert:
		body = m.viewConvert()
	case stateDone:
		body = m.viewDone()
	}
	return "\n" + header + "\n\n" + body + "\n"
}

func (m Model) viewPath() string {
	b := boxStyle.Render(
		lipgloss.NewStyle().Foreground(cFg).Render("Which folder should I scan?") + "\n" +
			helpStyle.Render("(searched recursively for video files)") + "\n\n" +
			m.ti.View())
	return b + "\n" + helpStyle.Render("  enter ▸ continue   ·   ctrl+c ▸ quit")
}

func (m Model) viewProfile() string {
	var rows []string
	for i, p := range profiles {
		cur := i == m.profileIdx
		marker := "  "
		name := normalRow.Render(p.String())
		if cur {
			marker = selectedRow.Render("▸ ")
			name = selectedRow.Render(p.String())
		}
		line := marker + name
		if cur {
			line += "\n     " + helpStyle.Render(profileBlurb[p])
		}
		rows = append(rows, line)
	}
	b := boxStyle.Render(
		lipgloss.NewStyle().Foreground(cFg).Bold(true).Render("Pick a conversion profile") +
			"\n\n" + strings.Join(rows, "\n"))
	return b + "\n" + helpStyle.Render("  ↑/↓ move · enter select · esc back")
}

func (m Model) viewScanning() string {
	// Analyze phase: report not yet ready and no file list yet.
	if m.report == nil && len(m.found) == 0 {
		return boxStyle.Render(
			gradientText("📊 Analyzing ") + m.root + "\n\n" +
				m.spin.View() + " probing files, measuring bitrates, picking the best method…")
	}
	done := len(m.items) + m.probeErr
	bar := m.spin.View()
	pct := 0.0
	if len(m.found) > 0 {
		pct = float64(done) / float64(len(m.found))
	}
	line := fmt.Sprintf("%s probing %d / %d files…", bar, done, len(m.found))
	return boxStyle.Render(
		gradientText("🔍 Scanning ") + m.root + "\n\n" +
			line + "\n\n" + m.overall.ViewAs(pct))
}

func (m Model) viewAnalyze() string {
	r := m.report
	if r == nil {
		return boxStyle.Render("analyzing…")
	}
	lbl := lipgloss.NewStyle().Foreground(cDim).Width(16)
	val := lipgloss.NewStyle().Foreground(cFg)

	four := ""
	if r.Files4K > 0 {
		four = badge4K.Render("4K") + dimRow.Render(fmt.Sprintf(" ×%d", r.Files4K))
	}

	lines := []string{
		gradientText("📊 Analysis") + dimRow.Render("  "+r.Root),
		"",
		lbl.Render("files") + val.Render(fmt.Sprintf("%d", r.Files)) + "  " + four,
		lbl.Render("source codecs") + val.Render(r.CodecSummary()),
		"",
		lbl.Render("recommended") + okStyle.Render(r.Recommended.String()),
		lbl.Render("why") + dimRow.Render(truncate(r.Why, 56)),
		"",
		lbl.Render("work") + val.Render(fmt.Sprintf("%d re-encode · %d audio-only · %d already-good",
			r.ReencodeCount, r.AudioOnly, r.NoOp)),
		lbl.Render("space") + val.Render(media.HumanBytes(r.OrigBytes)+" → "+media.HumanBytes(r.ProjBytes)) +
			"  " + savingsStyle.Render(fmt.Sprintf("save %s (%.0f%%)", media.HumanBytes(r.SavedBytes()), r.SavedPct())),
		lbl.Render("est. time") + copyStyle.Render("~"+media.HumanDuration(r.EstSecs)) +
			dimRow.Render("  (varies with CPU; copies are instant)"),
	}
	return boxStyle.Render(strings.Join(lines, "\n")) + "\n" +
		helpStyle.Render("  enter ▸ use recommended & scan · c ▸ choose method · esc back · q quit")
}

func (m Model) viewReview() string {
	if len(m.items) == 0 {
		return boxStyle.Render(copyStyle.Render("No video files found here.")) +
			"\n" + helpStyle.Render("  q ▸ quit")
	}

	var rows []string
	var selOrig, selProj int64
	rows = append(rows, dimRow.Render(fmt.Sprintf("   %-34s %-9s %8s %8s %7s", "FILE", "CODEC", "SIZE", "→ EST", "SAVE")))

	// window the list to keep it on screen
	start, end := windowBounds(m.cursor, len(m.items), 14)
	for i := start; i < end; i++ {
		it := m.items[i]
		name := truncate(filepath.Base(it.Info.Path), 34)
		codec := it.Info.Video.CodecName
		size := media.HumanBytes(it.Info.SizeBytes)
		est := media.HumanBytes(it.Plan.ProjectedBytes)

		check := checkOff
		if it.Selected && !it.Plan.NoOp() {
			check = checkOn
			selOrig += it.Info.SizeBytes
			selProj += it.Plan.ProjectedBytes
		}

		savTxt := fmt.Sprintf("%4.0f%%", it.Plan.SavedPct())
		var savStyled string
		switch {
		case it.Plan.NoOp():
			savStyled = copyStyle.Render(" n/a")
		case it.Plan.SavedBytes() < 0:
			savStyled = growStyle.Render(savTxt)
		case !it.Plan.ReencodeVideo:
			savStyled = copyStyle.Render("copy")
		default:
			savStyled = savingsStyle.Render(savTxt)
		}

		row := fmt.Sprintf(" %s %-34s %-9s %8s %8s %s", check, name, codec, size, est, savStyled)
		if i == m.cursor {
			row = selectedRow.Render("▸") + row[len([]byte("▸"))-2:]
			row = lipgloss.NewStyle().Background(lipgloss.Color("#26273b")).Render(row)
		}
		if it.Is4K {
			row += " " + badge4K.Render("4K")
		}
		rows = append(rows, row)
	}

	saved := selOrig - selProj
	pct := 0.0
	if selOrig > 0 {
		pct = float64(saved) / float64(selOrig) * 100
	}
	footer := footerStyle.Render(fmt.Sprintf(
		"profile %s   ·   selected %s → %s   %s",
		copyStyle.Render(m.profile.String()),
		media.HumanBytes(selOrig), media.HumanBytes(selProj),
		savingsStyle.Render(fmt.Sprintf("save %s (%.0f%%)", media.HumanBytes(saved), pct)),
	))

	help := helpStyle.Render("  ↑/↓ move · space toggle · a all · n none · enter ▸ CONVERT · esc back · q quit")
	return boxStyle.Render(strings.Join(rows, "\n")) + "\n" + footer + "\n" + help
}

func (m Model) viewConvert() string {
	if len(m.sel) == 0 {
		return boxStyle.Render("nothing to do")
	}
	it := m.sel[m.convIdx]
	name := filepath.Base(it.Info.Path)

	head := fmt.Sprintf("%s converting %s%s",
		m.spin.View(),
		selectedRow.Render(fmt.Sprintf("[%d/%d] ", m.convIdx+1, len(m.sel))),
		normalRow.Render(truncate(name, 46)))

	reason := helpStyle.Render("   " + strings.Join(it.Plan.Reasons, " · "))

	speed := m.curSpeed
	if speed == "" {
		speed = "…"
	}
	fileBar := m.bar.ViewAs(m.curFrac) + "  " +
		copyStyle.Render(fmt.Sprintf("%5.1f%%", m.curFrac*100)) +
		dimRow.Render("  @ "+speed)

	overallFrac := (float64(m.convIdx) + m.curFrac) / float64(len(m.sel))
	overallBar := m.overall.ViewAs(overallFrac) + "  " +
		savingsStyle.Render(fmt.Sprintf("%5.1f%%", overallFrac*100))

	saved := ""
	if m.savedReal != 0 {
		saved = "\n\n" + savingsStyle.Render("   reclaimed so far: "+media.HumanBytes(m.savedReal))
	}

	return boxStyle.Render(
		head + "\n" + reason + "\n\n" +
			dimRow.Render("   this file ") + "\n   " + fileBar + "\n\n" +
			dimRow.Render("   overall   ") + "\n   " + overallBar + saved) +
		"\n" + helpStyle.Render("  ctrl+c ▸ cancel")
}

func (m Model) viewDone() string {
	title := okStyle.Render("✅ Done!")
	if m.doneErr > 0 {
		title = copyStyle.Render("⚠ Finished with some errors")
	}
	lines := []string{
		title,
		"",
		fmt.Sprintf("  converted   %s", okStyle.Render(fmt.Sprintf("%d", m.doneOK))),
	}
	if m.doneErr > 0 {
		lines = append(lines, fmt.Sprintf("  failed      %s", errStyle.Render(fmt.Sprintf("%d", m.doneErr))))
	}
	lines = append(lines,
		fmt.Sprintf("  reclaimed   %s", savingsStyle.Render(media.HumanBytes(m.savedReal))),
		"",
		helpStyle.Render("  Outputs written next to originals as “… (plexprep).mkv”."),
		helpStyle.Render("  Eyeball them, then delete the originals when happy."),
	)
	for _, e := range m.convErrs {
		lines = append(lines, errStyle.Render("  ✘ "+truncate(e, 70)))
	}
	return boxStyle.Render(strings.Join(lines, "\n")) +
		"\n" + helpStyle.Render("  enter ▸ quit")
}

// --- helpers ---

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

func windowBounds(cursor, total, size int) (int, int) {
	if total <= size {
		return 0, total
	}
	start := cursor - size/2
	if start < 0 {
		start = 0
	}
	end := start + size
	if end > total {
		end = total
		start = end - size
	}
	return start, end
}
