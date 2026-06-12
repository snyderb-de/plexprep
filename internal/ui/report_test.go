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
		`.view:target{display:block}`,                     // pure-CSS drill-down, no JS
		`data-f="shrink"`, `data-f="grow"`, `agg-reclaim`, // per-view filter
		`class="pick-cb"`, `data-path=`, `data-folder="v-0"`, `id="selbar"`, // convert-list builder
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

func TestRenderHTMLEmbedKeepsFragmentLinksInSrcdoc(t *testing.T) {
	mk := func(name string) folderRow {
		r := &media.Report{Root: name, Files: 1, Recommended: media.ProfileZeroTranscode, Why: "x"}
		r.Details = append(r.Details, media.FileDetail{
			Name: name + "-ep.mkv", Codec: "h264", Width: 1920, Height: 1080,
			OrigBytes: 1000, ProjBytes: 600, Action: "keep", Method: "copy", Why: "ok",
		})
		r.OrigBytes, r.ProjBytes = 1000, 600
		return folderRow{Name: name, Report: r}
	}
	rows := []folderRow{mk("(root)"), mk("Show A")}

	embed := renderHTML("Y:/Media", rows, modeEmbed)
	if !strings.Contains(embed, `<base href="about:srcdoc">`) {
		t.Fatal("embed report must set an about:srcdoc base for iframe fragment links")
	}
	baseAt := strings.Index(embed, `<base href="about:srcdoc">`)
	linkAt := strings.Index(embed, `href="#v-0"`)
	if linkAt < 0 {
		t.Fatal(`missing drill-down link href="#v-0"`)
	}
	if baseAt > linkAt {
		t.Fatal("srcdoc base must appear before fragment links")
	}

	for name, mode := range map[string]renderMode{"static": modeStatic, "serve": modeServe} {
		if strings.Contains(renderHTML("Y:/Media", rows, mode), `<base href="about:srcdoc">`) {
			t.Fatalf("%s report should not use the srcdoc base", name)
		}
	}
}

// TestRenderHTMLSingleRowFlat verifies a lone "(root)" row (folder with loose
// files and no subfolders) renders as a flat file table with no
// folder-summary wrapper or drill-down link.
func TestRenderHTMLSingleRowFlat(t *testing.T) {
	r := &media.Report{Root: "(root)", Files: 1, Recommended: media.ProfileZeroTranscode, Why: "x"}
	r.Details = append(r.Details, media.FileDetail{
		Name: "movie.mkv", Codec: "h264", Width: 1920, Height: 1080,
		OrigBytes: 1000, ProjBytes: 600, Action: "keep", Method: "copy", Why: "ok",
	})
	r.OrigBytes, r.ProjBytes = 1000, 600
	rows := []folderRow{{Name: "(root)", Report: r}}

	s := renderHTML("Y:/Media", rows, modeEmbed)

	if !strings.Contains(s, `id="v-summary" data-kind="files"`) {
		t.Error(`want id="v-summary" data-kind="files"`)
	}
	for _, unwanted := range []string{`data-kind="folders"`, `id="v-0"`, `href="#v-0"`, `(root)`} {
		if strings.Contains(s, unwanted) {
			t.Errorf("unexpected %q in single-row output", unwanted)
		}
	}
	if c := strings.Count(s, "<table>"); c != 1 {
		t.Errorf("tables=%d want 1", c)
	}
}
