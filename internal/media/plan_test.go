package media

import (
	"strings"
	"testing"
)

func mi(codec string, w, h int, field string, audio ...Stream) *MediaInfo {
	return &MediaInfo{
		SizeBytes:    1_000_000_000,
		Duration:     3600,
		VideoBitrate: 5_000_000,
		Video:        Stream{CodecName: codec, Width: w, Height: h, FieldOrder: field},
		Audio:        audio,
	}
}

func aac() Stream       { return Stream{CodecName: "aac", CodecType: "audio", Channels: 2} }
func ac3() Stream       { return Stream{CodecName: "ac3", CodecType: "audio", Channels: 6} }
func ac3Stereo() Stream { return Stream{CodecName: "ac3", CodecType: "audio", Channels: 2} }
func dts() Stream       { return Stream{CodecName: "dts", CodecType: "audio", Channels: 6} }

func TestPlanLegacyZeroTranscode(t *testing.T) {
	p := BuildPlan(mi("mpeg2video", 720, 480, "tt", ac3()), ProfileZeroTranscode)
	if !p.ReencodeVideo || p.TargetCodec != "libx264" {
		t.Errorf("legacy mpeg2 should re-encode to libx264, got %+v", p)
	}
	if !p.Deinterlace {
		t.Error("interlaced source should deinterlace")
	}
	if !p.AddAAC {
		t.Error("ac3-only source should add AAC fallback")
	}
	if p.NoOp() {
		t.Error("should not be a no-op")
	}
}

func TestPlanModernCopied(t *testing.T) {
	p := BuildPlan(mi("h264", 1920, 1080, "progressive", aac()), ProfileZeroTranscode)
	if p.ReencodeVideo {
		t.Error("h264 should be copied, not re-encoded")
	}
	if p.AddAAC {
		t.Error("aac already present; should not add another")
	}
	if !p.NoOp() {
		t.Error("modern h264 + aac under zero-transcode should be a no-op")
	}
}

func TestPlan4KUsesHEVC(t *testing.T) {
	p := BuildPlan(mi("mpeg2video", 3840, 2160, "progressive", ac3()), Profile4K)
	if !p.ReencodeVideo || p.TargetCodec != "libx265" {
		t.Errorf("4K legacy should re-encode to libx265, got %+v", p)
	}
}

func TestPlanAudioOnlyNeverReencodes(t *testing.T) {
	p := BuildPlan(mi("mpeg2video", 720, 480, "tt", ac3()), ProfileAudioOnly)
	if p.ReencodeVideo {
		t.Error("audio-only profile must never re-encode video")
	}
	if !p.AddAAC {
		t.Error("audio-only should still add AAC when missing")
	}
}

func TestPlanCompatibleStereoSkipsAAC(t *testing.T) {
	p := BuildPlan(mi("h264", 1920, 1080, "progressive", ac3Stereo()), ProfileAudioOnly)
	if p.AddAAC {
		t.Error("ac3 stereo already direct-plays; should not add AAC fallback")
	}
	if !p.NoOp() {
		t.Errorf("audio-only with compatible stereo should be a no-op, got %+v", p)
	}
}

func TestPlanIncompatibleAudioGetsAAC(t *testing.T) {
	p := BuildPlan(mi("h264", 1920, 1080, "progressive", dts()), ProfileAudioOnly)
	if !p.AddAAC {
		t.Error("dts-only (multichannel, no stereo fallback) should add AAC")
	}
}

func TestPlanGrowWarning(t *testing.T) {
	p := BuildPlan(mi("h264", 1920, 1080, "progressive", dts()), ProfileAudioOnly)
	if p.SavedBytes() >= 0 {
		t.Fatalf("expected projected size to grow when adding AAC with no video savings, got %+v", p)
	}
	found := false
	for _, r := range p.Reasons {
		if strings.Contains(r, "grows by") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a grow-warning reason, got %v", p.Reasons)
	}
}

func TestSavedPct(t *testing.T) {
	p := Plan{OrigBytes: 1000, ProjectedBytes: 600}
	if p.SavedBytes() != 400 {
		t.Errorf("SavedBytes = %d", p.SavedBytes())
	}
	if p.SavedPct() != 40 {
		t.Errorf("SavedPct = %f", p.SavedPct())
	}
}
