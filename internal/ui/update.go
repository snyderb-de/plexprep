package ui

import (
	"plexprep/internal/media"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinnerTickWrap:
		// handled below via default spinner update
	}

	switch msg := msg.(type) {
	case analyzeMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.report = msg.report
		// Pre-select the recommended profile in the picker.
		for i, p := range profiles {
			if p == msg.report.Recommended {
				m.profileIdx = i
			}
		}
		m.state = stateAnalyze
		return m, nil

	case foundMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.found = msg.paths
		m.scanIdx = 0
		if len(m.found) == 0 {
			m.state = stateReview
			return m, nil
		}
		return m, probeCmd(m.found[0], m.profile)

	case probedMsg:
		if msg.err == nil && msg.item != nil {
			m.items = append(m.items, msg.item)
		} else {
			m.probeErr++
		}
		m.scanIdx++
		if m.scanIdx < len(m.found) {
			return m, probeCmd(m.found[m.scanIdx], m.profile)
		}
		m.state = stateReview
		m.cursor = 0
		return m, nil

	case progMsg:
		return m.handleProg(media.Progress(msg))
	}

	// Let spinner + progress bars consume their frames.
	var cmds []tea.Cmd
	var cmd tea.Cmd
	m.spin, cmd = m.spin.Update(msg)
	cmds = append(cmds, cmd)
	if m.state == statePath {
		m.ti, cmd = m.ti.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

type spinnerTickWrap struct{}

func (m Model) handleProg(pr media.Progress) (tea.Model, tea.Cmd) {
	if pr.Err != nil {
		m.doneErr++
		m.convErrs = append(m.convErrs, m.sel[m.convIdx].Info.Path+": "+pr.Err.Error())
		return m.advance()
	}
	if pr.Done {
		// Swap the finished temp file into place (sibling, or in-place replace).
		it := m.sel[m.convIdx]
		tmp := media.TempPath(it.Info.Path)
		final, err := media.Finalize(it.Info.Path, tmp, m.replace, m.purge)
		if err != nil {
			m.doneErr++
			m.convErrs = append(m.convErrs, it.Info.Path+": finalize: "+err.Error())
			return m.advance()
		}
		if newInfo, err := media.Probe(final); err == nil {
			m.savedReal += it.Info.SizeBytes - newInfo.SizeBytes
		}
		m.doneOK++
		return m.advance()
	}
	m.curFrac = pr.Fraction
	if pr.Speed != "" {
		m.curSpeed = pr.Speed
	}
	return m, m.listen()
}

func (m Model) advance() (tea.Model, tea.Cmd) {
	m.convIdx++
	if m.convIdx >= len(m.sel) {
		m.state = stateDone
		return m, nil
	}
	return m, m.startEncode()
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.cancel()
		return m, tea.Quit
	case "q":
		if m.state != statePath { // don't eat 'q' while typing a path
			m.cancel()
			return m, tea.Quit
		}
	}

	switch m.state {
	case statePath:
		switch msg.String() {
		case "enter":
			root := m.ti.Value()
			if root == "" {
				return m, nil
			}
			m.root = root
			m.state = stateScanning // reuse scanning view as "analyzing…"
			m.report = nil
			return m, tea.Batch(m.spin.Tick, analyzeCmd(root))
		}
		var cmd tea.Cmd
		m.ti, cmd = m.ti.Update(msg)
		return m, cmd

	case stateAnalyze:
		switch msg.String() {
		case "enter": // accept recommended profile, start the real scan
			m.profile = m.report.Recommended
			m.state = stateScanning
			m.items = nil
			m.found = nil
			return m, tea.Batch(m.spin.Tick, scanCmd(m.root))
		case "c": // choose a profile manually
			m.state = stateProfile
		case "esc":
			m.state = statePath
		}
		return m, nil

	case stateProfile:
		switch msg.String() {
		case "up", "k":
			if m.profileIdx > 0 {
				m.profileIdx--
			}
		case "down", "j":
			if m.profileIdx < len(profiles)-1 {
				m.profileIdx++
			}
		case "enter":
			m.profile = profiles[m.profileIdx]
			m.state = stateScanning
			m.items = nil
			return m, tea.Batch(m.spin.Tick, scanCmd(m.root))
		case "esc":
			m.state = statePath
		}
		return m, nil

	case stateReview:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case " ":
			if m.cursor < len(m.items) {
				m.items[m.cursor].Selected = !m.items[m.cursor].Selected
			}
		case "a":
			for _, it := range m.items {
				it.Selected = true
			}
		case "n":
			for _, it := range m.items {
				it.Selected = false
			}
		case "r":
			m.replace = !m.replace
		case "d":
			m.purge = !m.purge
		case "esc":
			m.state = stateProfile
		case "enter":
			m.sel = nil
			for _, it := range m.items {
				if it.Selected && !it.Plan.NoOp() {
					m.sel = append(m.sel, it)
				}
			}
			if len(m.sel) == 0 {
				return m, nil
			}
			m.state = stateConvert
			m.convIdx = 0
			return m, tea.Batch(m.spin.Tick, m.startEncode())
		}
		return m, nil

	case stateConvert:
		// only ctrl+c/q (handled above)
		return m, nil

	case stateDone:
		switch msg.String() {
		case "enter", "esc":
			m.cancel()
			return m, tea.Quit
		}
		return m, nil
	}
	return m, nil
}
