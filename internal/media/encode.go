package media

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// BuildArgs constructs the ffmpeg argument list for a file + plan.
func BuildArgs(mi *MediaInfo, p Plan, outPath string) []string {
	args := []string{"-y", "-i", mi.Path}

	// Map video first.
	args = append(args, "-map", "0:v:0")
	// Map every original audio track.
	args = append(args, "-map", "0:a")
	// If adding AAC, map the first audio again as the encode source.
	if p.AddAAC {
		args = append(args, "-map", "0:a:0")
	}
	// Keep subtitles if any (MKV holds image subs fine).
	if len(mi.Subs) > 0 {
		args = append(args, "-map", "0:s?")
	}

	// Video filter (deinterlace).
	if p.Deinterlace {
		args = append(args, "-vf", "yadif")
	}

	// Video codec.
	if p.ReencodeVideo {
		args = append(args, "-c:v", p.TargetCodec,
			"-crf", strconv.Itoa(p.CRF),
			"-preset", "slow",
			"-pix_fmt", "yuv420p")
		if p.TargetCodec == "libx265" {
			args = append(args, "-tag:v", "hvc1") // Apple/Plex-friendly HEVC tag
		}
	} else {
		args = append(args, "-c:v", "copy")
	}

	// Audio: copy all originals.
	args = append(args, "-c:a", "copy")
	// Appended AAC track is the last audio output stream.
	if p.AddAAC {
		idx := len(mi.Audio) // 0-based index of the appended stream among audio outputs
		args = append(args,
			fmt.Sprintf("-c:a:%d", idx), "aac",
			fmt.Sprintf("-b:a:%d", idx), "192k",
			fmt.Sprintf("-ac:a:%d", idx), "2",
			fmt.Sprintf("-metadata:s:a:%d", idx), "title=Stereo (AAC fallback)",
			fmt.Sprintf("-disposition:a:%d", idx), "0",
		)
	}

	// Copy subtitles.
	if len(mi.Subs) > 0 {
		args = append(args, "-c:s", "copy")
	}

	// Progress to stdout for parsing, quiet otherwise.
	args = append(args, "-progress", "pipe:1", "-nostats", "-loglevel", "error", "-nostdin", outPath)
	return args
}

// Progress is emitted as encoding advances.
type Progress struct {
	Fraction float64 // 0..1
	Speed    string  // e.g. "4.2x"
	Done     bool
	Err      error
}

// Encode runs ffmpeg and streams Progress on the returned channel until close.
func Encode(ctx context.Context, mi *MediaInfo, p Plan, outPath string) <-chan Progress {
	ch := make(chan Progress, 8)
	go func() {
		defer close(ch)

		args := BuildArgs(mi, p, outPath)
		cmd := exec.CommandContext(ctx, "ffmpeg", args...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			ch <- Progress{Err: err, Done: true}
			return
		}
		var stderr strings.Builder
		cmd.Stderr = &stderr

		if err := cmd.Start(); err != nil {
			ch <- Progress{Err: err, Done: true}
			return
		}

		totalUs := mi.Duration * 1_000_000
		sc := bufio.NewScanner(stdout)
		var speed string
		for sc.Scan() {
			line := sc.Text()
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			k, v = strings.TrimSpace(k), strings.TrimSpace(v)
			switch k {
			case "speed":
				speed = v
			case "out_time_us", "out_time_ms":
				us, _ := strconv.ParseFloat(v, 64)
				if k == "out_time_ms" {
					us *= 1000 // ffmpeg mislabels; out_time_ms is microseconds in some builds
				}
				frac := 0.0
				if totalUs > 0 {
					frac = us / totalUs
				}
				if frac > 1 {
					frac = 1
				}
				ch <- Progress{Fraction: frac, Speed: speed}
			}
		}

		err = cmd.Wait()
		if err != nil {
			ch <- Progress{Err: fmt.Errorf("ffmpeg: %v: %s", err, tail(stderr.String())), Done: true}
			return
		}
		ch <- Progress{Fraction: 1, Speed: speed, Done: true}
	}()
	return ch
}

func tail(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 300 {
		return "..." + s[len(s)-300:]
	}
	return s
}
