package media

import (
	"fmt"
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

	Codecs map[string]int // source video codec -> count
}

func (r Report) SavedBytes() int64 { return r.OrigBytes - r.ProjBytes }
func (r Report) SavedPct() float64 {
	if r.OrigBytes == 0 {
		return 0
	}
	return float64(r.SavedBytes()) / float64(r.OrigBytes) * 100
}

// Analyze probes a folder, recommends a profile, and totals savings + time.
func Analyze(root string) (*Report, error) {
	paths, err := FindVideos(root)
	if err != nil {
		return nil, err
	}
	r := &Report{Root: root, Codecs: map[string]int{}}

	// First pass: probe everything, learn the content mix.
	infos := make([]*MediaInfo, 0, len(paths))
	var bytes4K, bytesLegacy, bytesTotal int64
	anyLegacy := false
	for _, p := range paths {
		mi, err := Probe(p)
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
		return r, nil
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
		r.OrigBytes += mi.SizeBytes
		r.ProjBytes += plan.ProjectedBytes
		r.EstSecs += EstEncodeSeconds(mi, plan)
		switch {
		case plan.NoOp():
			r.NoOp++
		case plan.ReencodeVideo:
			r.ReencodeCount++
		default:
			r.AudioOnly++
		}
	}
	return r, nil
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
