package media

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Encoder throughput in megapixels/sec on a typical modern desktop CPU.
// Calibrated from observed runs (SD x265 -preset slow ≈ 3.4× realtime ≈ 35 Mpx/s).
// Copy is effectively IO-bound, so treated as near-instant.
const (
	tputX264Slow = 85.0   // Mpx/s
	tputX265Slow = 32.0   // Mpx/s
	tputCopy     = 2500.0 // Mpx/s (remux only)
)

// fps returns the video frame rate, defaulting to 25 if unknown.
func (mi *MediaInfo) fps() float64 {
	r := mi.Video.RFrameRate
	if r == "" {
		r = "25/1"
	}
	if n, d, ok := strings.Cut(r, "/"); ok {
		nn, _ := strconv.ParseFloat(n, 64)
		dd, _ := strconv.ParseFloat(d, 64)
		if dd > 0 && nn > 0 {
			return nn / dd
		}
	}
	return 25
}

// EstEncodeSeconds estimates wall-clock encode time for a plan.
func EstEncodeSeconds(mi *MediaInfo, p Plan) float64 {
	if mi.Duration <= 0 {
		return 0
	}
	frames := mi.fps() * mi.Duration
	if frames <= 0 {
		frames = 25 * mi.Duration
	}
	mpxPerFrame := float64(mi.Video.Width*mi.Video.Height) / 1_000_000
	if mpxPerFrame <= 0 {
		mpxPerFrame = 0.35 // assume SD if dimensions missing
	}
	totalMpx := frames * mpxPerFrame

	tput := tputCopy
	if p.ReencodeVideo {
		if p.TargetCodec == "libx265" {
			tput = tputX265Slow
		} else {
			tput = tputX264Slow
		}
	}
	secs := totalMpx / tput
	if p.Deinterlace {
		secs *= 1.15 // yadif overhead
	}
	return secs
}

// FileDetail is one file's analysis under the folder's recommended profile,
// used to render the report drill-down page.
type FileDetail struct {
	Name      string
	Path      string // full source path (for the report's convert-list export)
	Codec     string
	Width     int
	Height    int
	Is4K      bool
	OrigBytes int64
	ProjBytes int64
	EstSecs   float64
	Action    string // "re-encode" / "add-AAC" / "keep"
	Method    string // libx264 / libx265 / copy
	Why       string
}

func (d FileDetail) SavedBytes() int64 { return d.OrigBytes - d.ProjBytes }
func (d FileDetail) SavedPct() float64 {
	if d.OrigBytes == 0 {
		return 0
	}
	return float64(d.SavedBytes()) / float64(d.OrigBytes) * 100
}

// Report is the folder-level analysis.
type Report struct {
	Root        string
	Files       int
	ProbeErrors int

	Recommended Profile
	Why         string

	OrigBytes int64
	ProjBytes int64
	EstSecs   float64

	ReencodeCount int
	AudioOnly     int
	NoOp          int
	Files4K       int

	Codecs  map[string]int // source video codec -> count
	Details []FileDetail   // per-file breakdown (report drill-down)
}

func (r Report) SavedBytes() int64 { return r.OrigBytes - r.ProjBytes }
func (r Report) SavedPct() float64 {
	if r.OrigBytes == 0 {
		return 0
	}
	return float64(r.SavedBytes()) / float64(r.OrigBytes) * 100
}

// Analyze probes a folder recursively, recommends a profile, and totals
// savings + time.
func Analyze(root string) (*Report, error) {
	paths, err := FindVideos(root)
	if err != nil {
		return nil, err
	}
	return AnalyzePaths(root, paths), nil
}

// AnalyzePaths analyzes a fixed set of files (already discovered) under a label.
func AnalyzePaths(label string, paths []string) *Report {
	return AnalyzePathsCB(label, paths, nil)
}

// AnalyzePathsCB is AnalyzePaths with a per-file callback invoked after each
// probe (used to drive scan-progress UI). cb may be nil.
func AnalyzePathsCB(label string, paths []string, cb func(name string)) *Report {
	r := &Report{Root: label, Codecs: map[string]int{}}

	// First pass: probe everything, learn the content mix.
	infos := make([]*MediaInfo, 0, len(paths))
	var bytes4K, bytesLegacy, bytesTotal int64
	anyLegacy := false
	for _, p := range paths {
		mi, err := Probe(p)
		if cb != nil {
			cb(filepath.Base(p))
		}
		if err != nil {
			r.ProbeErrors++
			continue
		}
		infos = append(infos, mi)
		r.Files++
		r.Codecs[mi.Video.CodecName]++
		bytesTotal += mi.SizeBytes
		if mi.is4K() {
			r.Files4K++
			bytes4K += mi.SizeBytes
		}
		if legacyVideo[mi.Video.CodecName] {
			anyLegacy = true
			bytesLegacy += mi.SizeBytes
		}
	}
	if r.Files == 0 {
		r.Why = "no readable video files found"
		return r
	}

	// Recommend a profile from the content mix.
	switch {
	case anyLegacy && bytes4K > bytesTotal/2:
		r.Recommended = Profile4K
		r.Why = "majority is 4K with legacy video → HEVC keeps quality without bloating UHD"
	case anyLegacy:
		r.Recommended = ProfileZeroTranscode
		r.Why = fmt.Sprintf("legacy video present (%s) → H.264 for universal direct-play, no quality loss",
			HumanBytes(bytesLegacy))
	default:
		r.Recommended = ProfileAudioOnly
		r.Why = "video already modern → only add AAC so audio never transcodes"
	}

	// Second pass: total savings + time under the recommended profile.
	for _, mi := range infos {
		plan := BuildPlan(mi, r.Recommended)
		secs := EstEncodeSeconds(mi, plan)
		r.OrigBytes += mi.SizeBytes
		r.ProjBytes += plan.ProjectedBytes
		r.EstSecs += secs

		action := "add-AAC"
		switch {
		case plan.NoOp():
			r.NoOp++
			action = "keep"
		case plan.ReencodeVideo:
			r.ReencodeCount++
			action = "re-encode"
		default:
			r.AudioOnly++
		}

		method := "copy"
		if plan.TargetCodec != "" {
			method = plan.TargetCodec
		}
		r.Details = append(r.Details, FileDetail{
			Name:      filepath.Base(mi.Path),
			Path:      mi.Path,
			Codec:     mi.Video.CodecName,
			Width:     mi.Video.Width,
			Height:    mi.Video.Height,
			Is4K:      mi.is4K(),
			OrigBytes: mi.SizeBytes,
			ProjBytes: plan.ProjectedBytes,
			EstSecs:   secs,
			Action:    action,
			Method:    method,
			Why:       strings.Join(plan.Reasons, " · "),
		})
	}
	return r
}

// CodecSummary renders the source-codec histogram, busiest first.
func (r Report) CodecSummary() string {
	type kv struct {
		k string
		v int
	}
	var ks []kv
	for k, v := range r.Codecs {
		ks = append(ks, kv{k, v})
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i].v > ks[j].v })
	var parts []string
	for _, p := range ks {
		parts = append(parts, fmt.Sprintf("%s×%d", p.k, p.v))
	}
	return strings.Join(parts, "  ")
}

// HumanDuration formats seconds as e.g. "1h 23m" or "4m 12s".
func HumanDuration(s float64) string {
	if s <= 0 {
		return "~0s"
	}
	total := int(s + 0.5)
	h := total / 3600
	m := (total % 3600) / 60
	sec := total % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm %ds", m, sec)
	default:
		return fmt.Sprintf("%ds", sec)
	}
}
