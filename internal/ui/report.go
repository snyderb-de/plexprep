package ui

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"plexprep/internal/media"
	"plexprep/internal/style"

	"github.com/xuri/excelize/v2"
)

// folderRow is one 1st-level subfolder's analysis.
type folderRow struct {
	Name   string
	Report *media.Report
}

// ReportFolders analyzes every immediate subfolder of root and writes an
// .xlsx and .html summary (one row per subfolder). out is an optional output
// basename/path; empty → "<root>/plexprep-report".
func ReportFolders(root, out string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	var subs []string
	for _, e := range entries {
		if e.IsDir() {
			subs = append(subs, filepath.Join(root, e.Name()))
		}
	}
	sort.Strings(subs)
	if len(subs) == 0 {
		return fmt.Errorf("no subfolders under %s", root)
	}

	banner(fmt.Sprintf(`--report "%s"`, root))

	rows := make([]folderRow, 0, len(subs))
	for i, sub := range subs {
		fmt.Printf("\r  %s %s %s   ", style.Amber.S("▸"),
			style.Mid.S(fmt.Sprintf("[%d/%d]", i+1, len(subs))),
			style.Bright.S(style.Trunc(filepath.Base(sub), 50)))
		rep, err := media.Analyze(sub)
		if err != nil {
			continue
		}
		rows = append(rows, folderRow{Name: filepath.Base(sub), Report: rep})
	}
	fmt.Printf("\r  %s analyzed %s subfolders%-40s\n", style.Green.B("✓"), style.Bright.B(fmt.Sprintf("%d", len(rows))), "")

	base := out
	if base == "" {
		base = filepath.Join(root, "plexprep-report")
	} else {
		base = strings.TrimSuffix(base, filepath.Ext(base))
	}
	xlsxPath := base + ".xlsx"
	htmlPath := base + ".html"

	if err := writeXLSX(xlsxPath, root, rows); err != nil {
		return fmt.Errorf("xlsx: %w", err)
	}
	if err := writeHTML(htmlPath, root, rows); err != nil {
		return fmt.Errorf("html: %w", err)
	}
	fmt.Println(style.Frame("REPORT WRITTEN", []string{
		style.Mid.S("xlsx : ") + style.Green.S(xlsxPath),
		style.Mid.S("html : ") + style.Green.S(htmlPath),
	}))
	fmt.Println()
	return nil
}

var headers = []string{
	"Folder", "Files", "4K", "Source codecs", "Recommended method",
	"Original", "Est. output", "Save", "Save %", "Est. encode time",
	"Re-encode", "Audio-only", "Already-good", "Why",
}

func writeXLSX(path, root string, rows []folderRow) error {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Summary"
	f.SetSheetName(f.GetSheetName(0), sheet)

	hdrStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"6C5CE7"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	totStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"E8E6FB"}, Pattern: 1},
	})

	for c, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(c+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	f.SetCellStyle(sheet, "A1", mustCell(len(headers), 1), hdrStyle)

	var totOrig, totProj int64
	var totSecs float64
	r := 2
	for _, row := range rows {
		rp := row.Report
		totOrig += rp.OrigBytes
		totProj += rp.ProjBytes
		totSecs += rp.EstSecs
		vals := []interface{}{
			row.Name, rp.Files, rp.Files4K, rp.CodecSummary(), rp.Recommended.String(),
			media.HumanBytes(rp.OrigBytes), media.HumanBytes(rp.ProjBytes),
			media.HumanBytes(rp.SavedBytes()), fmt.Sprintf("%.0f%%", rp.SavedPct()),
			media.HumanDuration(rp.EstSecs),
			rp.ReencodeCount, rp.AudioOnly, rp.NoOp, rp.Why,
		}
		for c, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(c+1, r)
			f.SetCellValue(sheet, cell, v)
		}
		r++
	}

	// TOTAL row
	savePct := 0.0
	if totOrig > 0 {
		savePct = float64(totOrig-totProj) / float64(totOrig) * 100
	}
	totVals := []interface{}{
		"TOTAL", "", "", "", "",
		media.HumanBytes(totOrig), media.HumanBytes(totProj),
		media.HumanBytes(totOrig - totProj), fmt.Sprintf("%.0f%%", savePct),
		media.HumanDuration(totSecs), "", "", "", "",
	}
	for c, v := range totVals {
		cell, _ := excelize.CoordinatesToCellName(c+1, r)
		f.SetCellValue(sheet, cell, v)
	}
	f.SetCellStyle(sheet, mustCell(1, r), mustCell(len(headers), r), totStyle)

	// Column widths + freeze header.
	f.SetColWidth(sheet, "A", "A", 34)
	f.SetColWidth(sheet, "D", "E", 26)
	f.SetColWidth(sheet, "F", "J", 14)
	f.SetColWidth(sheet, "N", "N", 50)
	f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})

	return f.SaveAs(path)
}

func mustCell(col, row int) string {
	c, _ := excelize.CoordinatesToCellName(col, row)
	return c
}

// methodTag returns a short uppercase code for the recommended profile.
func methodTag(p media.Profile) string {
	switch p {
	case media.Profile4K:
		return "x265"
	case media.ProfileAudioOnly:
		return "aac"
	default:
		return "x264"
	}
}

// asciiBar renders a fixed-width [██████░░░░] meter for a percent.
func asciiBar(pct float64) string {
	const n = 10
	p := pct
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	filled := int(p/100*float64(n) + 0.5)
	return strings.Repeat("█", filled) + strings.Repeat("░", n-filled)
}

// saveTier maps a saving to a color class: grows red, then yellow→blue→green.
func saveTier(savedBytes int64, pct float64) string {
	switch {
	case savedBytes < 0:
		return "t-grow"
	case pct >= 35:
		return "t-high"
	case pct >= 15:
		return "t-mid"
	default:
		return "t-low"
	}
}

// savedCell renders the savings/growth cell contents (bar + figure).
func savedCell(savedBytes int64, pct float64) string {
	if savedBytes < 0 {
		// File would grow — show the increase, no negative percent.
		return fmt.Sprintf(`<span class="meter">░░░░░░░░░░</span> ▲ <span class="bytes">+%s larger</span>`,
			media.HumanBytes(-savedBytes))
	}
	return fmt.Sprintf(`<span class="meter">%s</span> %.0f%% <span class="bytes">%s</span>`,
		asciiBar(pct), pct, media.HumanBytes(savedBytes))
}

// htmlCols are the HTML table's columns (fewer + merged vs the XLSX detail set,
// so the table fits without horizontal scroll).
var htmlCols = []string{
	"folder", "files", "codecs", "method", "size", "→ est", "saved", "time", "work", "why",
}

func writeHTML(path, root string, rows []folderRow) error {
	var totOrig, totProj int64
	var totSecs float64
	var totFiles, totReenc int
	for _, row := range rows {
		rp := row.Report
		totOrig += rp.OrigBytes
		totProj += rp.ProjBytes
		totSecs += rp.EstSecs
		totFiles += rp.Files
		totReenc += rp.ReencodeCount
	}
	savePct := 0.0
	if totOrig > 0 {
		savePct = float64(totOrig-totProj) / float64(totOrig) * 100
	}

	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8">`)
	b.WriteString(`<meta name="viewport" content="width=device-width,initial-scale=1">`)
	fmt.Fprintf(&b, "<title>plexprep report :: %s</title>", html.EscapeString(filepath.Base(root)))
	b.WriteString(reportCSS)
	b.WriteString(`</head><body><div class="crt"></div><main><div class="term">`)

	// Title bar
	b.WriteString(`<div class="bar"><span class="dot r"></span><span class="dot y"></span><span class="dot g"></span>`)
	b.WriteString(`<span class="bartitle">plexprep — folder report</span></div>`)

	// Prompt + summary readout
	b.WriteString(`<div class="body">`)
	fmt.Fprintf(&b, `<div class="prompt"><span class="usr">bag@plexprep</span>:<span class="pwd">~</span>$ plexprep --report %s<span class="cur"></span></div>`,
		html.EscapeString(root))

	fmt.Fprintf(&b, `<pre class="summary">┌─ SUMMARY ─────────────────────────────────────────────┐
 root      : %s
 scanned   : %d subfolders · %d files · %d need re-encode
 size      : %s  ->  %s
 reclaim   : <span class="save">%s</span>  <span class="save">(%.0f%%)</span>
 est. time : ~%s   <span class="muted">(varies w/ CPU; copies instant)</span>
 generated : %s
└───────────────────────────────────────────────────────┘</pre>`,
		html.EscapeString(root), len(rows), totFiles, totReenc,
		media.HumanBytes(totOrig), media.HumanBytes(totProj),
		media.HumanBytes(totOrig-totProj), savePct,
		media.HumanDuration(totSecs),
		time.Now().Format("2006-01-02 15:04:05"))

	// Table
	b.WriteString(`<table><colgroup>`)
	widths := []string{"17%", "5%", "11%", "6%", "8%", "8%", "17%", "6%", "10%", "12%"}
	for _, w := range widths {
		fmt.Fprintf(&b, `<col style="width:%s">`, w)
	}
	b.WriteString(`</colgroup><thead><tr>`)
	numCols := map[int]bool{1: true, 4: true, 5: true, 7: true}
	for i, h := range htmlCols {
		cls := ""
		if numCols[i] {
			cls = ` class="num"`
		}
		fmt.Fprintf(&b, "<th%s>%s</th>", cls, html.EscapeString(h))
	}
	b.WriteString("</tr></thead><tbody>")

	for _, row := range rows {
		rp := row.Report
		k4 := ""
		if rp.Files4K > 0 {
			k4 = fmt.Sprintf(` <span class="k4">4K·%d</span>`, rp.Files4K)
		}
		work := fmt.Sprintf(
			`<span class="we">%d</span> re-encode<br><span class="wa">%d</span> add-AAC<br><span class="wk">%d</span> keep`,
			rp.ReencodeCount, rp.AudioOnly, rp.NoOp)

		b.WriteString("<tr>")
		fmt.Fprintf(&b, `<td class="folder" data-l="folder">%s%s</td>`, html.EscapeString(row.Name), k4)
		fmt.Fprintf(&b, `<td class="num" data-l="files">%d</td>`, rp.Files)
		fmt.Fprintf(&b, `<td class="codecs" data-l="codecs">%s</td>`, html.EscapeString(rp.CodecSummary()))
		fmt.Fprintf(&b, `<td class="method" data-l="method">%s</td>`, html.EscapeString(methodTag(rp.Recommended)))
		fmt.Fprintf(&b, `<td class="num" data-l="size">%s</td>`, media.HumanBytes(rp.OrigBytes))
		fmt.Fprintf(&b, `<td class="num" data-l="→ est">%s</td>`, media.HumanBytes(rp.ProjBytes))
		fmt.Fprintf(&b, `<td class="%s" data-l="saved">%s</td>`,
			saveTier(rp.SavedBytes(), rp.SavedPct()), savedCell(rp.SavedBytes(), rp.SavedPct()))
		fmt.Fprintf(&b, `<td class="num" data-l="time">%s</td>`, media.HumanDuration(rp.EstSecs))
		fmt.Fprintf(&b, `<td class="work" data-l="work">%s</td>`, work)
		fmt.Fprintf(&b, `<td class="why" data-l="why">%s</td>`, html.EscapeString(rp.Why))
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody><tfoot><tr>")
	totSaved := totOrig - totProj
	fmt.Fprintf(&b, `<td class="folder">TOTAL</td><td class="num">%d</td><td colspan="2"></td>`, totFiles)
	fmt.Fprintf(&b, `<td class="num">%s</td><td class="num">%s</td>`, media.HumanBytes(totOrig), media.HumanBytes(totProj))
	fmt.Fprintf(&b, `<td class="%s">%s</td>`, saveTier(totSaved, savePct), savedCell(totSaved, savePct))
	fmt.Fprintf(&b, `<td class="num">%s</td><td colspan="2"></td>`, media.HumanDuration(totSecs))
	b.WriteString("</tr></tfoot></table>")

	b.WriteString(`<div class="legend">` +
		`<b>work</b> &mdash; <span class="we">re-encode</span> video · <span class="wa">add-AAC</span> only · <span class="wk">keep</span> as-is` +
		` &nbsp;&nbsp;|&nbsp;&nbsp; <b>bar</b> &mdash; <span class="t-low">low</span> → <span class="t-mid">mid</span> → <span class="t-high">high</span> savings · <span class="t-grow">▲ larger</span></div>`)
	b.WriteString(`<div class="done">// estimates only · copies are near-instant&nbsp;<span class="cur"></span></div>`)
	b.WriteString(`</div></div></main></body></html>`)

	return os.WriteFile(path, []byte(b.String()), 0644)
}

const reportCSS = `<style>
@import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;700&display=swap');
:root{
  --bg:#080a08;--panel:#0c0f0c;--fg:#3ad968;--bright:#a8ffc4;--dim:#1f6b3a;--mid:#2a874b;
  --amber:#ffcc33;--blue:#4db5ff;--red:#ff5f56;--line:#13351f;
}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--fg);
  font-family:'JetBrains Mono',ui-monospace,'Cascadia Code',Consolas,monospace;
  font-size:13px;line-height:1.45;min-height:100vh;overflow-x:hidden}
/* faint CRT scanlines */
.crt{position:fixed;inset:0;z-index:5;pointer-events:none;opacity:.5;
  background:repeating-linear-gradient(0deg,#0000 0 2px,#00000055 2px 3px)}
main{width:100%;max-width:1900px;margin:0 auto;padding:clamp(14px,3vw,36px)}
.term{border:1px solid var(--line);border-radius:6px;background:var(--panel);
  box-shadow:0 0 0 1px #0a1a10,0 18px 60px -30px #000}
.bar{display:flex;align-items:center;gap:8px;padding:8px 12px;border-bottom:1px solid var(--line);
  background:#0a0d0a}
.dot{width:11px;height:11px;border-radius:50%}
.dot.r{background:#ff5f56}.dot.y{background:#ffbd2e}.dot.g{background:#27c93f}
.bartitle{margin-left:8px;color:var(--mid);font-size:12px;letter-spacing:.5px}
.body{padding:16px 18px 20px}
.prompt{color:var(--bright);margin-bottom:12px;word-break:break-all}
.usr{color:var(--amber)}.pwd{color:var(--mid)}
.cur{display:inline-block;width:8px;height:14px;background:var(--fg);margin-left:3px;
  vertical-align:-2px;animation:blink 1.1s step-end infinite}
@keyframes blink{50%{opacity:0}}
.summary{margin:0 0 18px;color:var(--fg);white-space:pre;overflow-x:auto;
  border:0;font-size:12.5px}
.summary .save{color:var(--bright)}
.muted{color:var(--dim)}
table{border-collapse:collapse;width:100%;table-layout:fixed;font-size:12px}
thead th{text-align:left;padding:5px 8px;color:var(--bg);background:var(--fg);
  text-transform:uppercase;letter-spacing:.5px;font-weight:700;border:1px solid var(--bg)}
th.num{text-align:right}
tbody td,tfoot td{padding:5px 8px;border-bottom:1px solid var(--line);
  vertical-align:top;overflow:hidden;text-overflow:ellipsis}
td.num{text-align:right;font-variant-numeric:tabular-nums}
.codecs,.why{white-space:normal;word-break:break-word;color:var(--mid)}
.folder{color:var(--bright);font-weight:700;white-space:normal;word-break:break-word}
.method{color:var(--amber)}
.work{color:var(--mid)}
.k4{color:var(--bg);background:var(--amber);padding:0 4px;border-radius:2px;font-size:10px;font-weight:700}
.save{color:var(--bright)}
.meter{font-family:inherit;letter-spacing:-1px}
/* savings tiers: yellow -> blue -> green, red if it grows */
.t-low,.t-low .meter{color:var(--amber)}
.t-mid,.t-mid .meter{color:var(--blue)}
.t-high,.t-high .meter{color:var(--fg)}
.t-grow,.t-grow .meter{color:var(--red)}
.bytes{color:var(--dim)}
/* work column */
.work{color:var(--mid);font-size:11px;line-height:1.35}
.we{color:var(--amber);font-weight:700}
.wa{color:var(--blue);font-weight:700}
.wk{color:var(--fg);font-weight:700}
.legend{margin-top:12px;color:var(--mid);font-size:11px}
.legend b{color:var(--bright);font-weight:700}
.legend .we,.legend .wa,.legend .wk,.legend .t-low,.legend .t-mid,.legend .t-high,.legend .t-grow{font-weight:700}
tbody tr:hover td{background:#0f1f14}
tbody tr:hover .folder{color:var(--bg);background:var(--fg)}
tfoot td{border-top:1px solid var(--fg);border-bottom:0;color:var(--bright);font-weight:700;padding-top:8px}
.done{margin-top:16px;color:var(--dim);font-size:12px}
@media (prefers-reduced-motion:reduce){.cur{animation:none}}
/* fluid type as it widens */
@media (min-width:1500px){table{font-size:13px}.body{padding:20px 24px 26px}}
/* collapse table into stacked cards on narrow screens */
@media (max-width:760px){
  .summary{font-size:11px}
  table,thead,tbody,tfoot,tr,td{display:block;width:auto}
  colgroup{display:none}
  thead{position:absolute;left:-9999px}
  tr{border:1px solid var(--line);border-radius:4px;margin-bottom:10px;padding:4px 2px;background:#0a0d0a}
  tbody tr:hover td{background:none}
  td{display:flex;justify-content:space-between;gap:12px;text-align:right!important;
     border-bottom:1px dashed var(--line);padding:5px 10px;overflow:visible}
  td:last-child{border-bottom:0}
  td::before{content:attr(data-l);color:var(--amber);text-transform:uppercase;
     font-size:10px;letter-spacing:.5px;text-align:left;opacity:.8}
  .folder,.codecs,.why{white-space:normal;word-break:break-word;text-align:right}
  tbody tr:hover .folder{background:none;color:var(--bright)}
  tfoot tr{border-top:1px solid var(--fg)}
}
</style>`
