package main

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"plexprep/internal/media"
	"plexprep/internal/ui"

	wr "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails-bound backend. It reuses the same scan/convert engine as the
// CLI and streams progress to the frontend via Wails events.
type App struct {
	ctx       context.Context
	mu        sync.Mutex
	busy      bool
	abort     bool
	abortScan bool
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

// Browse opens a native folder picker; returns "" if cancelled.
func (a *App) Browse() (string, error) {
	return wr.OpenDirectoryDialog(a.ctx, wr.OpenDialogOptions{Title: "Pick a folder to scan"})
}

// BrowseFiles opens a native multi-file picker filtered to video containers.
func (a *App) BrowseFiles() ([]string, error) {
	return wr.OpenMultipleFilesDialog(a.ctx, wr.OpenDialogOptions{
		Title: "Pick video files",
		Filters: []wr.FileFilter{{
			DisplayName: "Video",
			Pattern:     "*.mkv;*.mp4;*.m4v;*.avi;*.mov;*.wmv;*.ts;*.m2ts;*.mpg;*.mpeg;*.vob;*.flv",
		}},
	})
}

// Scan probes the target, emitting "pp:scan" progress events, and returns the
// rendered interactive report HTML (embed mode).
func (a *App) Scan(path string, recursive bool) (string, error) {
	a.mu.Lock()
	a.abortScan = false
	a.mu.Unlock()

	total := 0
	if fi, err := os.Stat(path); err == nil {
		if fi.IsDir() {
			if all, e := media.FindVideos(path); e == nil {
				total = len(all)
			}
		} else {
			total = 1
		}
	}
	wr.EventsEmit(a.ctx, "pp:scan", map[string]any{"t": "begin", "total": total})

	done := 0
	last := time.Now()
	cb := func(name string) bool {
		done++
		if time.Since(last) > 50*time.Millisecond || done == total {
			last = time.Now()
			wr.EventsEmit(a.ctx, "pp:scan", map[string]any{"t": "probe", "done": done, "total": total, "name": name})
		}
		a.mu.Lock()
		ab := a.abortScan
		a.mu.Unlock()
		return ab
	}
	html, err := ui.ScanToHTML(path, recursive, cb)
	if err != nil {
		wr.EventsEmit(a.ctx, "pp:scan", map[string]any{"t": "error", "msg": err.Error()})
		return "", err
	}
	wr.EventsEmit(a.ctx, "pp:scan", map[string]any{"t": "done"})
	return html, nil
}

// ScanFiles probes an explicit file selection (from BrowseFiles) and returns
// the rendered report, emitting "pp:scan" progress.
func (a *App) ScanFiles(paths []string) (string, error) {
	a.mu.Lock()
	a.abortScan = false
	a.mu.Unlock()

	total := len(paths)
	wr.EventsEmit(a.ctx, "pp:scan", map[string]any{"t": "begin", "total": total})
	done := 0
	last := time.Now()
	cb := func(name string) bool {
		done++
		if time.Since(last) > 50*time.Millisecond || done == total {
			last = time.Now()
			wr.EventsEmit(a.ctx, "pp:scan", map[string]any{"t": "probe", "done": done, "total": total, "name": name})
		}
		a.mu.Lock()
		ab := a.abortScan
		a.mu.Unlock()
		return ab
	}
	html, err := ui.ScanFilesToHTML(paths, cb)
	if err != nil {
		wr.EventsEmit(a.ctx, "pp:scan", map[string]any{"t": "error", "msg": err.Error()})
		return "", err
	}
	wr.EventsEmit(a.ctx, "pp:scan", map[string]any{"t": "done"})
	return html, nil
}

// Abort requests the running conversion stop after the current file.
func (a *App) Abort() {
	a.mu.Lock()
	a.abort = true
	a.mu.Unlock()
}

// AbortScan requests the running scan stop after the current file probe.
func (a *App) AbortScan() {
	a.mu.Lock()
	a.abortScan = true
	a.mu.Unlock()
}

// Reveal opens the file's containing folder in the OS file manager.
func (a *App) Reveal(path string) {
	if path != "" {
		wr.BrowserOpenURL(a.ctx, "file://"+filepath.ToSlash(filepath.Dir(path)))
	}
}

// Convert runs the selected files sequentially, emitting "pp:convert" events.
func (a *App) Convert(paths []string, profile string, replace, del bool) {
	a.mu.Lock()
	if a.busy {
		a.mu.Unlock()
		wr.EventsEmit(a.ctx, "pp:convert", map[string]any{"t": "fail", "name": "", "err": "a conversion is already running"})
		return
	}
	a.busy = true
	a.abort = false
	a.mu.Unlock()
	defer func() { a.mu.Lock(); a.busy = false; a.mu.Unlock() }()

	prof := media.ProfileZeroTranscode
	switch profile {
	case "4k":
		prof = media.Profile4K
	case "audio":
		prof = media.ProfileAudioOnly
	}

	emit := func(v map[string]any) { wr.EventsEmit(a.ctx, "pp:convert", v) }
	total := len(paths)
	var ok, fail, skip int
	var reclaim int64

	for i, p := range paths {
		a.mu.Lock()
		aborted := a.abort
		a.mu.Unlock()
		if aborted {
			break
		}
		name := filepath.Base(p)
		emit(map[string]any{"t": "start", "idx": i + 1, "total": total, "name": name})

		if fi, err := os.Stat(p); err != nil || fi.IsDir() || !media.IsVideoExt(p) {
			fail++
			emit(map[string]any{"t": "fail", "name": name, "err": "not a video file"})
			continue
		}
		it, err := media.BuildItem(p, prof)
		if err != nil {
			fail++
			emit(map[string]any{"t": "fail", "name": name, "err": "probe: " + err.Error()})
			continue
		}
		if it.Plan.NoOp() {
			skip++
			emit(map[string]any{"t": "skip", "name": name, "reason": "already optimal"})
			continue
		}

		tmp := media.TempPath(p)
		t0 := time.Now()
		failed := false
		for pr := range media.Encode(a.ctx, it.Info, it.Plan, tmp) {
			if pr.Err != nil {
				failed = true
				fail++
				emit(map[string]any{"t": "fail", "name": name, "err": pr.Err.Error()})
				break
			}
			if pr.Done {
				break
			}
			eta := 0.0
			if pr.Fraction > 0.02 {
				eta = time.Since(t0).Seconds() * (1 - pr.Fraction) / pr.Fraction
			}
			emit(map[string]any{"t": "progress", "frac": pr.Fraction, "speed": pr.Speed, "eta": eta})
		}
		if failed {
			_ = os.Remove(tmp)
			continue
		}

		final, err := media.Finalize(p, tmp, replace, del)
		if err != nil {
			fail++
			emit(map[string]any{"t": "fail", "name": name, "err": "finalize: " + err.Error()})
			continue
		}
		var newSize int64
		if ni, _ := media.Probe(final); ni != nil {
			newSize = ni.SizeBytes
		}
		saved := it.Info.SizeBytes - newSize
		reclaim += saved
		ok++
		emit(map[string]any{"t": "done-file", "name": filepath.Base(final),
			"orig": it.Info.SizeBytes, "nu": newSize, "saved": saved})
	}
	emit(map[string]any{"t": "summary", "ok": ok, "fail": fail, "skip": skip, "reclaim": reclaim})
}
