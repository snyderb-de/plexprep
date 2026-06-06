package media

import (
	"io/fs"
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
		ext := strings.ToLower(filepath.Ext(path))
		if !videoExts[ext] {
			return nil
		}
		if strings.Contains(filepath.Base(path), "(plexprep)") {
			return nil
		}
		out = append(out, path)
		return nil
	})
	sort.Strings(out)
	return out, err
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

// OutputPath returns the sibling output filename for an item.
func OutputPath(in string) string {
	ext := filepath.Ext(in)
	base := strings.TrimSuffix(in, ext)
	// Force MKV output (holds AC3 + image subs + multi-audio cleanly).
	return base + " (plexprep).mkv"
}
