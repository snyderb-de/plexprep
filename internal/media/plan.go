package media

import "strings"

// Profile is a conversion strategy chosen by the user.
type Profile int

const (
	ProfileZeroTranscode Profile = iota // SD/HD: x264 CRF18 only if legacy, else copy video. +AAC.
	Profile4K                           // UHD: x265 CRF20 if legacy, else copy. +AAC.
	ProfileAudioOnly                    // copy video always, just add AAC.
)

func (p Profile) String() string {
	switch p {
	case ProfileZeroTranscode:
		return "Zero-Transcode (SD/HD)"
	case Profile4K:
		return "4K UHD (HEVC)"
	case ProfileAudioOnly:
		return "Audio-only fix"
	}
	return "?"
}

// legacyVideo is the set of codecs that force Plex transcoding and should be
// re-encoded to a modern codec.
var legacyVideo = map[string]bool{
	"mpeg2video": true,
	"mpeg1video": true,
	"vc1":        true,
	"mpeg4":      true,
	"msmpeg4v3":  true,
	"msmpeg4v2":  true,
	"wmv3":       true,
	"wmv2":       true,
	"h263":       true,
	"flv1":       true,
	"vp6f":       true,
}

// Plan is the decided action for one file under a profile.
type Plan struct {
	ReencodeVideo bool
	TargetCodec   string // libx264 / libx265 / "" (copy)
	CRF           int
	Deinterlace   bool
	AddAAC        bool
	Reasons       []string

	OrigBytes      int64
	ProjectedBytes int64
}

// SavedBytes is the projected reduction (can be negative if it grows).
func (p Plan) SavedBytes() int64 { return p.OrigBytes - p.ProjectedBytes }

// SavedPct is the projected percent reduction.
func (p Plan) SavedPct() float64 {
	if p.OrigBytes == 0 {
		return 0
	}
	return float64(p.SavedBytes()) / float64(p.OrigBytes) * 100
}

// NoOp reports whether the file needs nothing (already optimal for this profile).
func (p Plan) NoOp() bool { return !p.ReencodeVideo && !p.AddAAC }

func (mi *MediaInfo) is4K() bool {
	return mi.Video.Width >= 3000 || mi.Video.Height >= 1700
}

func (mi *MediaInfo) interlaced() bool {
	switch strings.ToLower(mi.Video.FieldOrder) {
	case "tt", "bb", "tb", "bt":
		return true
	}
	return false
}

func (mi *MediaInfo) hasAACStereo() bool {
	for _, a := range mi.Audio {
		if a.CodecName == "aac" && a.Channels <= 2 {
			return true
		}
	}
	return false
}

// BuildPlan decides what to do with a file under a profile and estimates size.
func BuildPlan(mi *MediaInfo, profile Profile) Plan {
	p := Plan{OrigBytes: mi.SizeBytes}
	legacy := legacyVideo[mi.Video.CodecName]

	switch profile {
	case ProfileAudioOnly:
		// never touch video
	case Profile4K:
		if legacy {
			p.ReencodeVideo = true
			p.TargetCodec = "libx265"
			p.CRF = 20
			p.Reasons = append(p.Reasons, "legacy "+mi.Video.CodecName+" → HEVC")
		} else {
			p.Reasons = append(p.Reasons, "video copied ("+mi.Video.CodecName+" ok)")
		}
	default: // ProfileZeroTranscode
		if legacy {
			p.ReencodeVideo = true
			p.TargetCodec = "libx264"
			p.CRF = 18
			p.Reasons = append(p.Reasons, "legacy "+mi.Video.CodecName+" → H.264")
		} else {
			p.Reasons = append(p.Reasons, "video copied ("+mi.Video.CodecName+" direct-plays)")
		}
	}

	if p.ReencodeVideo && mi.interlaced() {
		p.Deinterlace = true
		p.Reasons = append(p.Reasons, "deinterlace")
	}

	if !mi.hasAACStereo() {
		p.AddAAC = true
		p.Reasons = append(p.Reasons, "+AAC stereo fallback")
	} else {
		p.Reasons = append(p.Reasons, "AAC already present")
	}

	p.ProjectedBytes = projectSize(mi, p)
	return p
}

// projectSize estimates output bytes. Heuristic, labelled ~est in the UI.
func projectSize(mi *MediaInfo, p Plan) int64 {
	dur := mi.Duration
	if dur <= 0 {
		return mi.SizeBytes
	}

	// Video component.
	var videoBits float64
	if p.ReencodeVideo {
		eff := codecEfficiency(mi.Video.CodecName, p.TargetCodec)
		videoBits = float64(mi.VideoBitrate) * eff * dur
	} else {
		videoBits = float64(mi.VideoBitrate) * dur
	}

	// Original audio is always kept.
	var audioBits float64
	for _, a := range mi.Audio {
		br := parseBitrate(a.BitRate)
		if br == 0 {
			br = 256000
		}
		audioBits += float64(br) * dur
	}
	// Added AAC stereo @192k.
	if p.AddAAC {
		audioBits += 192000 * dur
	}

	bytes := (videoBits + audioBits) / 8
	bytes *= 1.01 // container overhead
	return int64(bytes)
}

// codecEfficiency is the rough output/input video-bitrate ratio needed to hold
// equivalent quality when moving from src codec to the target encoder.
func codecEfficiency(src, target string) float64 {
	switch target {
	case "libx265":
		// to HEVC
		switch src {
		case "mpeg2video", "mpeg1video":
			return 0.40
		default:
			return 0.55
		}
	default: // libx264
		switch src {
		case "mpeg2video", "mpeg1video":
			return 0.50
		case "vc1", "wmv3", "wmv2":
			return 0.60
		default:
			return 0.65
		}
	}
}
