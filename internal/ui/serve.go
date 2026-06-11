package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"plexprep/internal/media"
	"plexprep/internal/style"
)

// server holds the serve-mode state: a single in-flight convert job guarded by
// busy, and an abort flag the status page can flip.
type server struct {
	root         string
	mu           sync.Mutex
	busy         bool
	abort        bool
	cancelEncode context.CancelFunc
	scans        map[string]string // scan id -> rendered report HTML
}

// Serve starts the local web UI: a picker, the interactive report, and a live
// convert dashboard. It binds 127.0.0.1 only and opens the default browser.
func Serve(root string, port int) error {
	s := &server{root: root}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/report", s.handleReport)
	mux.HandleFunc("/view", s.handleView)
	mux.HandleFunc("/api/ls", s.handleLS)
	mux.HandleFunc("/api/scan", s.handleScan)
	mux.HandleFunc("/api/convert", s.handleConvert)
	mux.HandleFunc("/api/abort", s.handleAbort)
	mux.HandleFunc("/api/abort-now", s.handleAbortNow)

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	url := fmt.Sprintf("http://%s", ln.Addr().String())

	banner("--serve")
	start := url
	if root != "" {
		start = url + "/report?recursive=1&path=" + urlQuery(root)
	}
	fmt.Println(style.Frame("SERVING", []string{
		style.Mid.S("url   : ") + style.Bright.B(start),
		style.Mid.S("bind  : ") + style.Green.S("127.0.0.1 (local only)"),
		style.Dim.S("Ctrl+C to stop"),
	}))
	fmt.Println()

	go func() {
		time.Sleep(350 * time.Millisecond)
		openBrowser(start)
	}()
	return http.Serve(ln, mux)
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if s.root != "" {
		http.Redirect(w, r, "/report?recursive=1&path="+urlQuery(s.root), http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, pickerHTML)
}

// handleReport returns the scanning shell immediately; the shell calls
// /api/scan (which streams progress) and then loads the cached /view. This
// keeps the browser responsive instead of blocking on a whole-tree probe.
func (s *server) handleReport(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, scanShell)
}

// handleView serves a finished scan's rendered report by id.
func (s *server) handleView(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	s.mu.Lock()
	html := s.scans[id]
	s.mu.Unlock()
	if html == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

// handleScan probes the target, streaming newline-JSON progress, then caches
// the rendered report and emits its view id.
func (s *server) handleScan(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	recursive := r.URL.Query().Get("recursive") != "0"
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, _ := w.(http.Flusher)
	enc := json.NewEncoder(w)
	emit := func(v any) {
		_ = enc.Encode(v)
		if flusher != nil {
			flusher.Flush()
		}
	}

	// Count total up front (fast walk) so the bar has a denominator.
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
	emit(map[string]any{"t": "begin", "total": total})

	done := 0
	last := time.Now()
	cb := func(name string) bool {
		done++
		if time.Since(last) > 60*time.Millisecond || done == total {
			last = time.Now()
			emit(map[string]any{"t": "probe", "done": done, "total": total, "name": name})
		}
		return false
	}

	rows, err := buildRows(path, recursive, cb)
	if err != nil {
		emit(map[string]any{"t": "error", "msg": err.Error()})
		return
	}
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	s.mu.Lock()
	if s.scans == nil {
		s.scans = map[string]string{}
	}
	s.scans[id] = renderHTML(path, rows, modeServe)
	s.mu.Unlock()
	emit(map[string]any{"t": "done", "id": id})
}

// ErrScanAborted is returned by ScanToHTML / ScanFilesToHTML when cb signals
// cancellation mid-probe.
var ErrScanAborted = errors.New("scan cancelled")

// ScanToHTML scans a target and returns the rendered interactive report HTML
// (serve mode). cb fires after each file probe for progress UI; cb returning
// true cancels the scan early. Exported for the desktop (Wails) front end,
// which reuses the same report engine.
func ScanToHTML(path string, recursive bool, cb func(name string) bool) (string, error) {
	rows, err := buildRows(path, recursive, cb)
	if err != nil {
		return "", err
	}
	return renderHTML(path, rows, modeEmbed), nil
}

// ScanFilesToHTML scans an explicit list of files (e.g. a native multi-select)
// into one "(selection)" report, embed mode. cb fires per probe; cb returning
// true cancels the scan early.
func ScanFilesToHTML(paths []string, cb func(name string) bool) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("no files selected")
	}
	rep := media.AnalyzePathsCB("(selection)", paths, cb)
	if rep.Aborted {
		return "", ErrScanAborted
	}
	if rep.Files == 0 {
		return "", fmt.Errorf("no readable video files in selection")
	}
	rows := []folderRow{{Name: "(selection)", Report: rep}}
	return renderHTML("(selection)", rows, modeEmbed), nil
}

// buildRows analyzes a target into report rows. A file → one row; a folder →
// a "(root)" row of loose files plus (when recursive) one row per subfolder.
// cb fires after each file probe (nil ok); cb returning true cancels the scan
// early (returns ErrScanAborted).
func buildRows(path string, recursive bool, cb func(name string) bool) ([]folderRow, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		if !media.IsVideoExt(path) {
			return nil, fmt.Errorf("%s is not a video file", filepath.Base(path))
		}
		rep := media.AnalyzePathsCB(filepath.Dir(path), []string{path}, cb)
		if rep.Aborted {
			return nil, ErrScanAborted
		}
		return []folderRow{{Name: filepath.Base(path), Report: rep}}, nil
	}

	var rows []folderRow
	if top, err := media.FindVideosTop(path); err == nil && len(top) > 0 {
		rep := media.AnalyzePathsCB(path, top, cb)
		if rep.Aborted {
			return nil, ErrScanAborted
		}
		rows = append(rows, folderRow{Name: "(root)", Report: rep})
	}
	if !recursive {
		if len(rows) == 0 {
			return nil, fmt.Errorf("no video files directly in %s (enable recursive to scan subfolders)", path)
		}
		return rows, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var subs []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			subs = append(subs, filepath.Join(path, e.Name()))
		}
	}
	sort.Strings(subs)
	for _, sub := range subs {
		files, _ := media.FindVideos(sub)
		if len(files) == 0 {
			continue
		}
		rep := media.AnalyzePathsCB(sub, files, cb)
		if rep.Aborted {
			return nil, ErrScanAborted
		}
		if rep.Files == 0 {
			continue
		}
		rows = append(rows, folderRow{Name: filepath.Base(sub), Report: rep})
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no readable video files under %s", path)
	}
	return rows, nil
}

// lsEntry is one directory-browser row.
type lsEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size,omitempty"`
}

func (s *server) handleLS(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Query().Get("path")
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)

	// Empty path: Windows → drive list; Unix → filesystem root.
	if p == "" {
		if runtime.GOOS == "windows" {
			var dirs []lsEntry
			for c := 'A'; c <= 'Z'; c++ {
				d := string(c) + `:\`
				if _, err := os.Stat(d); err == nil {
					dirs = append(dirs, lsEntry{Name: d, Path: d})
				}
			}
			enc.Encode(map[string]any{"path": "", "parent": nil, "crumbs": []any{}, "dirs": dirs, "files": []any{}})
			return
		}
		p = "/"
	}

	p = filepath.Clean(p)
	entries, err := os.ReadDir(p)
	if err != nil {
		enc.Encode(map[string]any{"error": err.Error()})
		return
	}
	var dirs, files []lsEntry
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(p, name)
		if e.IsDir() {
			dirs = append(dirs, lsEntry{Name: name, Path: full})
			continue
		}
		if !media.IsVideoExt(name) {
			continue
		}
		info, _ := e.Info()
		var sz int64
		if info != nil {
			sz = info.Size()
		}
		files = append(files, lsEntry{Name: name, Path: full, Size: sz})
	}
	sort.Slice(dirs, func(i, j int) bool { return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name) })
	sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name) })

	var parent any
	if pp := filepath.Dir(p); pp != p {
		parent = pp
	} else if runtime.GOOS == "windows" {
		parent = "" // back to drive list
	}

	enc.Encode(map[string]any{
		"path": p, "parent": parent, "crumbs": crumbs(p),
		"dirs": dirs, "files": files,
	})
}

func crumbs(p string) []lsEntry {
	var out []lsEntry
	cur := p
	for {
		out = append([]lsEntry{{Name: filepath.Base(cur), Path: cur}}, out...)
		parent := filepath.Dir(cur)
		if parent == cur {
			// volume root (e.g. "C:\" or "/")
			out[0].Name = cur
			break
		}
		cur = parent
	}
	return out
}

type convertReq struct {
	Root    string   `json:"root"`
	Profile string   `json:"profile"`
	Replace bool     `json:"replace"`
	Delete  bool     `json:"delete"`
	Paths   []string `json:"paths"`
}

func (s *server) handleAbort(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.abort = true
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// handleAbortNow stops the conversion immediately, killing the in-flight
// ffmpeg process and discarding its partial output.
func (s *server) handleAbortNow(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.abort = true
	cancel := s.cancelEncode
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleConvert runs the selected files sequentially, streaming newline-JSON
// progress events the status page consumes.
func (s *server) handleConvert(w http.ResponseWriter, r *http.Request) {
	var req convertReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		http.Error(w, "a conversion is already running", http.StatusConflict)
		return
	}
	s.busy = true
	s.abort = false
	s.mu.Unlock()
	defer func() { s.mu.Lock(); s.busy = false; s.mu.Unlock() }()

	profile := media.ProfileZeroTranscode
	switch req.Profile {
	case "4k":
		profile = media.Profile4K
	case "audio":
		profile = media.ProfileAudioOnly
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, _ := w.(http.Flusher)
	enc := json.NewEncoder(w)
	emit := func(v any) {
		_ = enc.Encode(v)
		if flusher != nil {
			flusher.Flush()
		}
	}

	ctx := r.Context()
	total := len(req.Paths)
	var ok, fail, skip int
	var reclaim int64

	for i, p := range req.Paths {
		s.mu.Lock()
		aborted := s.abort
		s.mu.Unlock()
		if aborted || ctx.Err() != nil {
			break
		}
		name := filepath.Base(p)
		emit(map[string]any{"t": "start", "idx": i + 1, "total": total, "name": name})

		// Safety: only operate on existing video files.
		if fi, err := os.Stat(p); err != nil || fi.IsDir() || !media.IsVideoExt(p) {
			fail++
			emit(map[string]any{"t": "fail", "name": name, "err": "not a video file"})
			continue
		}

		it, err := media.BuildItem(p, profile)
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
		ectx, cancel := context.WithCancel(ctx)
		s.mu.Lock()
		s.cancelEncode = cancel
		s.mu.Unlock()

		failed := false
		for pr := range media.Encode(ectx, it.Info, it.Plan, tmp) {
			if pr.Err != nil {
				failed = true
				s.mu.Lock()
				abortedNow := s.abort
				s.mu.Unlock()
				if !abortedNow {
					fail++
					emit(map[string]any{"t": "fail", "name": name, "err": pr.Err.Error()})
				}
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
		cancel()
		s.mu.Lock()
		s.cancelEncode = nil
		abortedNow := s.abort
		s.mu.Unlock()
		if failed {
			_ = os.Remove(tmp)
			if abortedNow {
				break
			}
			continue
		}

		final, err := media.Finalize(p, tmp, req.Replace, req.Delete)
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

// --- helpers ---

func urlQuery(s string) string {
	r := strings.NewReplacer(" ", "%20", "\\", "%5C", "&", "%26", "#", "%23", "?", "%3F", "+", "%2B")
	return r.Replace(s)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
