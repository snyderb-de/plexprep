package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"plexprep/internal/media"
)

// TestWriteHTMLDrilldown verifies the single-file report ships a summary view
// plus one hidden detail panel per folder, with working toggle links.
func TestWriteHTMLDrilldown(t *testing.T) {
	mk := func(name string, n int) folderRow {
		r := &media.Report{Root: name, Files: n, Recommended: media.ProfileZeroTranscode, Why: "x"}
		for i := 0; i < n; i++ {
			r.Details = append(r.Details, media.FileDetail{
				Name: name + "-ep.mkv", Codec: "h264", Width: 1920, Height: 1080,
				OrigBytes: 1000, ProjBytes: 600, Action: "keep", Method: "copy", Why: "ok",
			})
		}
		r.OrigBytes = int64(n) * 1000
		r.ProjBytes = int64(n) * 600
		return folderRow{Name: name, Report: r}
	}
	rows := []folderRow{mk("(root)", 2), mk("Show A", 3)}
	p := filepath.Join(t.TempDir(), "r.html")
	if err := writeHTML(p, "Y:/Media", rows); err != nil {
		t.Fatal(err)
	}
	out, _ := os.ReadFile(p)
	s := string(out)
	for _, want := range []string{
		`id="v-summary" data-kind="folders"`, `id="v-0" data-kind="files"`, `id="v-1" data-kind="files"`,
		`href="#v-0"`, `href="#v-1"`, `href="#v-summary"`,
		`.view:target{display:block}`, // pure-CSS drill-down, no JS
		`data-f="shrink"`, `data-f="grow"`, `agg-reclaim`, // per-view filter
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q", want)
		}
	}
	// One summary table + one per folder panel.
	if c := strings.Count(s, "<table>"); c != 3 {
		t.Errorf("tables=%d want 3", c)
	}
}
