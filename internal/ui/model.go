package ui

import (
	"context"

	"plexprep/internal/media"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	statePath state = iota
	stateAnalyze
	stateProfile
	stateScanning
	stateReview
	stateConvert
	stateDone
)

// Model is the root Bubble Tea model.
type Model struct {
	state  state
	width  int
	height int

	ti      textinput.Model
	spin    spinner.Model
	bar     progress.Model
	overall progress.Model

	profileIdx int
	profile    media.Profile

	report *media.Report

	root    string
	found   []string
	scanIdx int
	probeErr int

	items  []*media.Item
	cursor int

	sel       []*media.Item
	convIdx   int
	curFrac   float64
	curSpeed  string
	ch        <-chan media.Progress
	ctx       context.Context
	cancel    context.CancelFunc
	doneOK    int
	doneErr   int
	savedReal int64
	convErrs  []string

	err error
}

var profiles = []media.Profile{
	media.ProfileZeroTranscode,
	media.Profile4K,
	media.ProfileAudioOnly,
}

var profileBlurb = map[media.Profile]string{
	media.ProfileZeroTranscode: "x264 CRF18 for legacy video, copy modern. +AAC. No quality loss, no transcoding.",
	media.Profile4K:            "x265/HEVC CRF20 for UHD. Keeps existing HEVC. +AAC fallback.",
	media.ProfileAudioOnly:     "Never touch video. Just add an AAC stereo track. Instant compat fix.",
}

// New builds the initial model, optionally pre-filled with a start path.
func New(startPath string) Model {
	ti := textinput.New()
	ti.Placeholder = `Z:\TV Shows\Baseball (1994)`
	ti.Prompt = "📁 "
	ti.Focus()
	ti.CharLimit = 4096
	ti.Width = 60
	if startPath != "" {
		ti.SetValue(startPath)
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(cCyan)

	bar := progress.New(progress.WithScaledGradient(string(cPurple), string(cCyan)))
	bar.Width = 44
	ov := progress.New(progress.WithScaledGradient(string(cPink), string(cGreen)))
	ov.Width = 44

	ctx, cancel := context.WithCancel(context.Background())

	return Model{
		state:   statePath,
		ti:      ti,
		spin:    sp,
		bar:     bar,
		overall: ov,
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spin.Tick)
}

// --- messages ---

type foundMsg struct {
	paths []string
	err   error
}
type probedMsg struct {
	item *media.Item
	err  error
}
type progMsg media.Progress
type startNextMsg struct{}
type analyzeMsg struct {
	report *media.Report
	err    error
}

// --- commands ---

func analyzeCmd(root string) tea.Cmd {
	return func() tea.Msg {
		r, err := media.Analyze(root)
		return analyzeMsg{r, err}
	}
}

func scanCmd(root string) tea.Cmd {
	return func() tea.Msg {
		paths, err := media.FindVideos(root)
		return foundMsg{paths, err}
	}
}

func probeCmd(path string, p media.Profile) tea.Cmd {
	return func() tea.Msg {
		it, err := media.BuildItem(path, p)
		return probedMsg{it, err}
	}
}

func (m *Model) listen() tea.Cmd {
	ch := m.ch
	return func() tea.Msg {
		pr, ok := <-ch
		if !ok {
			return progMsg{Done: true}
		}
		return progMsg(pr)
	}
}

func (m *Model) startEncode() tea.Cmd {
	it := m.sel[m.convIdx]
	out := media.OutputPath(it.Info.Path)
	m.ch = media.Encode(m.ctx, it.Info, it.Plan, out)
	m.curFrac = 0
	m.curSpeed = ""
	return m.listen()
}
