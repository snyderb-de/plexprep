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
	p := BuildPlan(mi("mpeg2video", 720, 480, "tt", ac3()), ProfileZeroTranscode, 0)
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
	p := BuildPlan(mi("h264", 1920, 1080, "progressive", aac()), ProfileZeroTranscode, 0)
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
	p := BuildPlan(mi("mpeg2video", 3840, 2160, "progressive", ac3()), Profile4K, 0)
	if !p.ReencodeVideo || p.TargetCodec != "libx265" {
		t.Errorf("4K legacy should re-encode to libx265, got %+v", p)
	}
}

func TestPlanAudioOnlyNeverReencodes(t *testing.T) {
	p := BuildPlan(mi("mpeg2video", 720, 480, "tt", ac3()), ProfileAudioOnly, 0)
	if p.ReencodeVideo {
		t.Error("audio-only profile must never re-encode video")
	}
	if !p.AddAAC {
		t.Error("audio-only should still add AAC when missing")
	}
}

func TestPlanCompatibleStereoSkipsAAC(t *testing.T) {
	p := BuildPlan(mi("h264", 1920, 1080, "progressive", ac3Stereo()), ProfileAudioOnly, 0)
	if p.AddAAC {
		t.Error("ac3 stereo already direct-plays; should not add AAC fallback")
	}
	if !p.NoOp() {
		t.Errorf("audio-only with compatible stereo should be a no-op, got %+v", p)
	}
}

func TestPlanIncompatibleAudioGetsAAC(t *testing.T) {
	p := BuildPlan(mi("h264", 1920, 1080, "progressive", dts()), ProfileAudioOnly, 0)
	if !p.AddAAC {
		t.Error("dts-only (multichannel, no stereo fallback) should add AAC")
	}
}

func TestPlanGrowWarning(t *testing.T) {
	p := BuildPlan(mi("h264", 1920, 1080, "progressive", dts()), ProfileAudioOnly, 0)
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

func TestPlanShrinkRoutesCodec(t *testing.T) {
	// h264 1080p -> libx264, CRF passed through, always re-encodes (never no-op).
	p := BuildPlan(mi("h264", 1920, 1080, "progressive", aac()), ProfileShrink, 23)
	if !p.ReencodeVideo || p.TargetCodec != "libx264" {
		t.Errorf("shrink h264 1080p should re-encode to libx264, got %+v", p)
	}
	if p.CRF != 23 {
		t.Errorf("shrink should pass CRF through, got %d", p.CRF)
	}
	if p.NoOp() {
		t.Error("shrink must never be a no-op")
	}

	// hevc -> libx265 (same family).
	if p := BuildPlan(mi("hevc", 1920, 1080, "progressive", aac()), ProfileShrink, 24); p.TargetCodec != "libx265" {
		t.Errorf("shrink hevc should use libx265, got %q", p.TargetCodec)
	}

	// 4K source uses libx265 regardless of source codec (x264 at UHD is huge).
	if p := BuildPlan(mi("h264", 3840, 2160, "progressive", aac()), ProfileShrink, 22); p.TargetCodec != "libx265" {
		t.Errorf("shrink 4K should use libx265 even from h264, got %q", p.TargetCodec)
	}
}

func TestShrinkSameFamilyNoPhantomSavings(t *testing.T) {
	// Re-encoding h264->libx264 at the baseline CRF must not project a free
	// ~35% reduction (the old cross-codec efficiency bug). At CRF18 the video
	// bitrate is held, so adding an AAC track should make it grow, not shrink.
	p := BuildPlan(mi("h264", 1920, 1080, "progressive", dts()), ProfileShrink, 18)
	if p.SavedBytes() > 0 {
		t.Errorf("same-family CRF18 re-encode should not project savings, got saved=%d", p.SavedBytes())
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
