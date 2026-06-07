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

	rows := make([]folderRow, 0, len(subs))
	for i, sub := range subs {
		fmt.Printf("\r  analyzing %d/%d: %-40.40s", i+1, len(subs), filepath.Base(sub))
		rep, err := media.Analyze(sub)
		if err != nil {
			continue
		}
		rows = append(rows, folderRow{Name: filepath.Base(sub), Report: rep})
	}
	fmt.Printf("\r  analyzed %d subfolders%-40s\n", len(rows), "")

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
	fmt.Printf("  ✓ %s\n  ✓ %s\n", xlsxPath, htmlPath)
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

// methodEmoji maps a recommended profile to an emoji for the report.
func methodEmoji(p media.Profile) string {
	switch p {
	case media.Profile4K:
		return "🟣"
	case media.ProfileAudioOnly:
		return "🔊"
	default:
		return "🎬"
	}
}

func writeHTML(path, root string, rows []folderRow) error {
	// Totals first (used in the hero stat cards).
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
	fmt.Fprintf(&b, "<title>✨ plexprep report — %s</title>", html.EscapeString(filepath.Base(root)))
	b.WriteString(reportCSS)
	b.WriteString(`</head><body><div class="aurora"></div><main>`)

	// Hero
	b.WriteString(`<header class="hero">`)
	b.WriteString(`<h1 class="glow" data-text="✨ plexprep">✨ plexprep</h1>`)
	b.WriteString(`<div class="tagline">folder report</div>`)
	fmt.Fprintf(&b, `<div class="sub">📁 %s &nbsp;·&nbsp; %d subfolders &nbsp;·&nbsp; %s</div>`,
		html.EscapeString(root), len(rows), time.Now().Format("Jan 2 2006 · 15:04"))
	b.WriteString(`</header>`)

	// Stat cards
	b.WriteString(`<section class="cards">`)
	statCard(&b, "💾", "Reclaimable", media.HumanBytes(totOrig-totProj), fmt.Sprintf("%.0f%% smaller", savePct), "green")
	statCard(&b, "📦", "Current size", media.HumanBytes(totOrig), media.HumanBytes(totProj)+" after", "cyan")
	statCard(&b, "⏱️", "Est. encode", "~"+media.HumanDuration(totSecs), "varies w/ CPU", "purple")
	statCard(&b, "🎞️", "Files", fmt.Sprintf("%d", totFiles), fmt.Sprintf("%d need re-encode", totReenc), "yellow")
	b.WriteString(`</section>`)

	// Table
	b.WriteString(`<section class="tablewrap"><table><thead><tr>`)
	numCols := map[int]bool{1: true, 2: true, 5: true, 6: true, 7: true, 8: true, 9: true, 10: true, 11: true, 12: true}
	for i, h := range headers {
		cls := ""
		if numCols[i] {
			cls = ` class="num"`
		}
		fmt.Fprintf(&b, "<th%s>%s</th>", cls, html.EscapeString(h))
	}
	b.WriteString("</tr></thead><tbody>")

	maxOrig := int64(1)
	for _, row := range rows {
		if row.Report.OrigBytes > maxOrig {
			maxOrig = row.Report.OrigBytes
		}
	}

	for i, row := range rows {
		rp := row.Report
		saveCls := "save"
		if rp.SavedBytes() < 0 {
			saveCls = "grow"
		}
		four := "—"
		if rp.Files4K > 0 {
			four = fmt.Sprintf(`<span class="badge">%d ◆</span>`, rp.Files4K)
		}
		// Saving meter width (clamped 0..100).
		meter := rp.SavedPct()
		if meter < 0 {
			meter = 0
		}
		if meter > 100 {
			meter = 100
		}
		// Stagger the fade-in.
		fmt.Fprintf(&b, `<tr style="animation-delay:%dms">`, i*45)
		fmt.Fprintf(&b, `<td class="folder">📂 %s</td>`, html.EscapeString(row.Name))
		fmt.Fprintf(&b, `<td class="num">%d</td>`, rp.Files)
		fmt.Fprintf(&b, `<td class="num">%s</td>`, four)
		fmt.Fprintf(&b, `<td class="codecs">%s</td>`, html.EscapeString(rp.CodecSummary()))
		fmt.Fprintf(&b, `<td class="method">%s %s</td>`, methodEmoji(rp.Recommended), html.EscapeString(rp.Recommended.String()))
		fmt.Fprintf(&b, `<td class="num">%s</td>`, media.HumanBytes(rp.OrigBytes))
		fmt.Fprintf(&b, `<td class="num">%s</td>`, media.HumanBytes(rp.ProjBytes))
		fmt.Fprintf(&b, `<td class="num %s">%s</td>`, saveCls, media.HumanBytes(rp.SavedBytes()))
		fmt.Fprintf(&b, `<td class="num %s"><div class="meter"><span style="width:%.0f%%"></span></div>%.0f%%</td>`, saveCls, meter, rp.SavedPct())
		fmt.Fprintf(&b, `<td class="num">%s</td>`, media.HumanDuration(rp.EstSecs))
		fmt.Fprintf(&b, `<td class="num">%d</td>`, rp.ReencodeCount)
		fmt.Fprintf(&b, `<td class="num">%d</td>`, rp.AudioOnly)
		fmt.Fprintf(&b, `<td class="num">%d</td>`, rp.NoOp)
		fmt.Fprintf(&b, `<td class="why">%s</td>`, html.EscapeString(rp.Why))
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody><tfoot><tr>")
	fmt.Fprintf(&b, `<td class="folder">Σ TOTAL</td><td colspan="4"></td>`)
	fmt.Fprintf(&b, `<td class="num">%s</td><td class="num">%s</td>`, media.HumanBytes(totOrig), media.HumanBytes(totProj))
	fmt.Fprintf(&b, `<td class="num save">%s</td><td class="num save">%.0f%%</td>`, media.HumanBytes(totOrig-totProj), savePct)
	fmt.Fprintf(&b, `<td class="num">%s</td><td colspan="4"></td>`, media.HumanDuration(totSecs))
	b.WriteString("</tr></tfoot></table></section>")
	b.WriteString(`<footer>Sizes &amp; times are estimates · copies are near-instant · generated by <span class="glow-sm">plexprep</span> ✨</footer>`)
	b.WriteString("</main></body></html>")

	return os.WriteFile(path, []byte(b.String()), 0644)
}

func statCard(b *strings.Builder, emoji, label, big, small, tone string) {
	fmt.Fprintf(b, `<div class="card %s"><div class="ico">%s</div><div class="lbl">%s</div><div class="big">%s</div><div class="sml">%s</div></div>`,
		tone, emoji, html.EscapeString(label), html.EscapeString(big), html.EscapeString(small))
}

const reportCSS = `<style>
@import url('https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@400;500;700&family=JetBrains+Mono:wght@500&display=swap');
:root{
  --bg:#0d0e1a;--bg2:#12132440;--card:#181a2e;--fg:#d7defb;--dim:#7079a8;
  --pink:#ff6ac1;--purple:#a682ff;--cyan:#5ef6ff;--green:#6ee7a0;--yellow:#ffd866;--red:#ff6b6b;
}
*{box-sizing:border-box}
html{scroll-behavior:smooth}
body{margin:0;background:var(--bg);color:var(--fg);
  font-family:'Space Grotesk',system-ui,Segoe UI,sans-serif;font-size:14px;
  min-height:100vh;overflow-x:hidden;position:relative}
/* drifting aurora backdrop */
.aurora{position:fixed;inset:-30%;z-index:0;pointer-events:none;filter:blur(90px);opacity:.45;
  background:
    radial-gradient(40% 40% at 20% 30%,#ff6ac155,transparent 60%),
    radial-gradient(45% 45% at 80% 25%,#5ef6ff44,transparent 60%),
    radial-gradient(50% 50% at 60% 80%,#a682ff55,transparent 60%);
  animation:drift 22s ease-in-out infinite alternate}
@keyframes drift{0%{transform:translate(0,0) rotate(0deg) scale(1)}100%{transform:translate(4%,-3%) rotate(8deg) scale(1.12)}}
main{position:relative;z-index:1;max-width:1280px;margin:0 auto;padding:44px 28px 60px}
.hero{text-align:center;margin-bottom:36px}
h1.glow{font-size:54px;margin:0;font-weight:700;letter-spacing:-1px;position:relative;
  background:linear-gradient(92deg,var(--pink),var(--purple) 45%,var(--cyan));
  -webkit-background-clip:text;background-clip:text;color:transparent;
  filter:drop-shadow(0 0 18px #a682ff66);
  background-size:220% auto;animation:shimmer 6s linear infinite}
@keyframes shimmer{to{background-position:220% center}}
.tagline{font-size:18px;color:var(--dim);letter-spacing:6px;text-transform:uppercase;margin-top:2px}
.sub{color:var(--dim);margin-top:14px;font-size:13px}
/* stat cards */
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:16px;margin-bottom:30px}
.card{background:linear-gradient(160deg,#1c1f38,#14152600);border:1px solid #ffffff14;border-radius:16px;
  padding:18px 20px;position:relative;overflow:hidden;backdrop-filter:blur(6px);
  transition:transform .25s cubic-bezier(.2,.8,.2,1),box-shadow .25s;animation:rise .6s both}
.card:hover{transform:translateY(-4px)}
.card .ico{font-size:26px;filter:drop-shadow(0 0 10px currentColor)}
.card .lbl{color:var(--dim);text-transform:uppercase;letter-spacing:1.5px;font-size:11px;margin-top:8px}
.card .big{font-size:28px;font-weight:700;margin-top:2px}
.card .sml{color:var(--dim);font-size:12px;margin-top:2px}
.card.green{color:var(--green)}.card.green:hover{box-shadow:0 10px 40px -10px var(--green)}
.card.cyan{color:var(--cyan)}.card.cyan:hover{box-shadow:0 10px 40px -10px var(--cyan)}
.card.purple{color:var(--purple)}.card.purple:hover{box-shadow:0 10px 40px -10px var(--purple)}
.card.yellow{color:var(--yellow)}.card.yellow:hover{box-shadow:0 10px 40px -10px var(--yellow)}
.card .big,.card .sml,.card .lbl{color:var(--fg)}
@keyframes rise{from{opacity:0;transform:translateY(14px)}to{opacity:1;transform:none}}
/* table */
.tablewrap{border-radius:16px;overflow:auto;border:1px solid #ffffff12;
  box-shadow:0 24px 80px -30px #000;background:var(--card)}
table{border-collapse:collapse;width:100%;font-size:13px}
thead th{position:sticky;top:0;z-index:2;text-align:left;padding:14px 14px;white-space:nowrap;
  background:linear-gradient(92deg,#6c5ce7,#8b5cf6);color:#fff;font-weight:600;letter-spacing:.3px}
th.num{text-align:right}
tbody td{padding:11px 14px;border-bottom:1px solid #ffffff0d;white-space:nowrap}
td.num{text-align:right;font-family:'JetBrains Mono',monospace;font-variant-numeric:tabular-nums}
tbody tr{animation:fadein .5s both}
@keyframes fadein{from{opacity:0;transform:translateX(-8px)}to{opacity:1;transform:none}}
tbody tr:hover td{background:#ffffff0a}
tbody tr:hover .folder{color:var(--cyan);text-shadow:0 0 12px #5ef6ff88}
.folder{font-weight:600;transition:.2s}
.codecs{font-family:'JetBrains Mono',monospace;color:var(--dim)}
.method{color:var(--cyan)}
.why{white-space:normal;max-width:320px;color:var(--dim)}
.save{color:var(--green);font-weight:700}
.grow{color:var(--red);font-weight:700}
.badge{background:linear-gradient(92deg,var(--cyan),var(--purple));color:#0d0e1a;border-radius:6px;
  padding:2px 7px;font-weight:700;font-size:11px;box-shadow:0 0 12px #5ef6ff55}
/* saving meter */
.meter{display:inline-block;vertical-align:middle;width:54px;height:6px;border-radius:6px;background:#ffffff14;
  margin-right:8px;overflow:hidden}
.meter>span{display:block;height:100%;border-radius:6px;width:0;
  background:linear-gradient(92deg,var(--green),var(--cyan));box-shadow:0 0 10px var(--green);
  animation:fill 1.1s cubic-bezier(.2,.8,.2,1) both .3s}
@keyframes fill{from{width:0}}
tfoot td{padding:14px;background:#1d2040;font-weight:700;border-top:2px solid var(--purple)}
.glow-sm{background:linear-gradient(92deg,var(--pink),var(--cyan));-webkit-background-clip:text;background-clip:text;color:transparent}
footer{text-align:center;color:var(--dim);margin-top:22px;font-size:12px}
@media (prefers-reduced-motion:reduce){*{animation:none!important}}
</style>`
