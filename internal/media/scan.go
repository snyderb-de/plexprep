package media

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// videoExts are the container extensions we consider.
var videoExts = map[string]bool{
	".mkv": true, ".mp4": true, ".m4v": true, ".avi": true,
	".mov": true, ".wmv": true, ".ts": true, ".m2ts": true,
	".mpg": true, ".mpeg": true, ".vob": true, ".flv": true,
}

// Item pairs a probed file with its plan under the chosen profile.
type Item struct {
	Info     *MediaInfo
	Plan     Plan
	Is4K     bool
	Selected bool
}

// FindVideos walks root recursively and returns candidate video paths,
// skipping our own "(plexprep)" outputs.
func FindVideos(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries, keep walking
		}
		if d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		// Skip dotfiles and macOS AppleDouble sidecars ("._Movie.mkv"): they
		// carry video extensions on non-Mac drives but are tiny junk that
		// ffprobe can't read.
		if strings.HasPrefix(base, ".") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !videoExts[ext] {
			return nil
		}
		if strings.Contains(base, "(plexprep)") {
			return nil
		}
		out = append(out, path)
		return nil
	})
	sort.Strings(out)
	return out, err
}

// FindVideosTop returns video files directly inside root (non-recursive),
// skipping dotfiles and our own outputs. Used for the report's "(root)" row.
func FindVideosTop(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		base := e.Name()
		if strings.HasPrefix(base, ".") {
			continue
		}
		if !videoExts[strings.ToLower(filepath.Ext(base))] {
			continue
		}
		if strings.Contains(base, "(plexprep)") {
			continue
		}
		out = append(out, filepath.Join(root, base))
	}
	sort.Strings(out)
	return out, nil
}

// IsVideoExt reports whether name has a recognized video container extension.
func IsVideoExt(name string) bool {
	return videoExts[strings.ToLower(filepath.Ext(name))]
}

// ResolveTargets turns a mixed list of files and directories into a flat,
// de-duplicated list of video paths. Directories are walked recursively
// (FindVideos); files are taken as-is if they carry a video extension. Order
// follows the input, with directory contents sorted.
func ResolveTargets(targets []string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	for _, t := range targets {
		fi, err := os.Stat(t)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", t, err)
		}
		if fi.IsDir() {
			sub, err := FindVideos(t)
			if err != nil {
				return nil, err
			}
			for _, p := range sub {
				add(p)
			}
			continue
		}
		if !videoExts[strings.ToLower(filepath.Ext(t))] {
			return nil, fmt.Errorf("%s: not a video file", t)
		}
		add(t)
	}
	return out, nil
}

// ReadListFile reads newline-separated paths from a file, skipping blanks and
// "#" comment lines. Surrounding quotes/whitespace are trimmed per line.
func ReadListFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		s := strings.TrimSpace(strings.Trim(strings.TrimSpace(line), `"`))
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

// BuildItem probes a path and plans it under profile.
func BuildItem(path string, profile Profile) (*Item, error) {
	mi, err := Probe(path)
	if err != nil {
		return nil, err
	}
	it := &Item{Info: mi, Is4K: mi.is4K()}
	it.Plan = BuildPlan(mi, profile)
	it.Selected = !it.Plan.NoOp()
	return it, nil
}
