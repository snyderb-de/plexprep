package media

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Stream is one ffprobe stream entry (subset of fields we care about).
type Stream struct {
	Index         int               `json:"index"`
	CodecName     string            `json:"codec_name"`
	CodecType     string            `json:"codec_type"`
	Width         int               `json:"width"`
	Height        int               `json:"height"`
	FieldOrder    string            `json:"field_order"`
	RFrameRate    string            `json:"r_frame_rate"`
	Channels      int               `json:"channels"`
	ChannelLayout string            `json:"channel_layout"`
	BitRate       string            `json:"bit_rate"`
	Tags          map[string]string `json:"tags"`
}

// Format is the ffprobe format block.
type Format struct {
	Duration string            `json:"duration"`
	BitRate  string            `json:"bit_rate"`
	Size     string            `json:"size"`
	Tags     map[string]string `json:"tags"`
}

type probeOutput struct {
	Streams []Stream `json:"streams"`
	Format  Format   `json:"format"`
}

// MediaInfo is the digested view of a media file used by the planner.
type MediaInfo struct {
	Path      string
	SizeBytes int64
	Duration  float64 // seconds

	Video Stream
	Audio []Stream
	Subs  []Stream

	VideoBitrate int64 // bits/sec, best-effort
}

// Probe runs ffprobe and returns a MediaInfo. Empty/unreadable -> error.
func Probe(path string) (*MediaInfo, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)
	noWindow(cmd)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("%w: %s", err, msg)
		}
		return nil, err
	}
	var po probeOutput
	if err := json.Unmarshal(out, &po); err != nil {
		return nil, err
	}

	mi := &MediaInfo{Path: path}
	mi.Duration, _ = strconv.ParseFloat(po.Format.Duration, 64)
	mi.SizeBytes, _ = strconv.ParseInt(po.Format.Size, 10, 64)

	for _, s := range po.Streams {
		switch s.CodecType {
		case "video":
			// Skip cover-art / attached pics (tiny, mjpeg/png with no duration).
			if mi.Video.CodecName == "" && s.CodecName != "mjpeg" && s.CodecName != "png" {
				mi.Video = s
			}
		case "audio":
			mi.Audio = append(mi.Audio, s)
		case "subtitle":
			mi.Subs = append(mi.Subs, s)
		}
	}

	mi.VideoBitrate = deriveVideoBitrate(mi)
	return mi, nil
}

// deriveVideoBitrate finds the true average video bitrate. Priority:
//  1. BPS container tag (MakeMKV/mkvmerge write the measured per-stream average)
//  2. derive from filesize minus audio (always the real average)
//  3. stream bit_rate field — last resort; for MPEG-2 it is often the VBV
//     ceiling (e.g. 8 Mbps) rather than the actual average, so it overestimates.
func deriveVideoBitrate(mi *MediaInfo) int64 {
	for k, v := range mi.Video.Tags {
		if strings.HasPrefix(strings.ToUpper(k), "BPS") {
			if br := parseBitrate(v); br > 0 {
				return br
			}
		}
	}
	// Derive: total - audio.
	if mi.Duration > 0 && mi.SizeBytes > 0 {
		total := float64(mi.SizeBytes) * 8 / mi.Duration
		var audio float64
		for _, a := range mi.Audio {
			if br := parseBitrate(a.BitRate); br > 0 {
				audio += float64(br)
			} else {
				audio += 256000 // assume ~256k if unknown
			}
		}
		if v := total - audio; v > 0 {
			return int64(v)
		}
	}
	// Last resort: stream bit_rate (may be a VBV ceiling, overestimates).
	if br := parseBitrate(mi.Video.BitRate); br > 0 {
		return br
	}
	return 0
}

func parseBitrate(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "N/A" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}
