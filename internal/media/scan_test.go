package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindVideosSkipsJunk(t *testing.T) {
	d := t.TempDir()
	mk := func(name string) {
		if err := os.WriteFile(filepath.Join(d, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mk("Episode 1.mkv")              // real
	mk("Episode 2.mp4")              // real
	mk("._Episode 1.mkv")            // macOS AppleDouble sidecar -> skip
	mk(".DS_Store")                  // dotfile -> skip
	mk("Episode 1 (plexprep).mkv")   // our own output -> skip
	mk("poster.jpg")                 // non-video -> skip

	got, err := FindVideos(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("found %d videos, want 2: %v", len(got), got)
	}
	for _, p := range got {
		b := filepath.Base(p)
		if b != "Episode 1.mkv" && b != "Episode 2.mp4" {
			t.Errorf("unexpected file included: %s", b)
		}
	}
}
