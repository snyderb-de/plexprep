package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTargetsAndListFile(t *testing.T) {
	d := t.TempDir()
	mk := func(n string) string { p := filepath.Join(d, n); os.WriteFile(p, []byte("x"), 0644); return p }
	a := mk("A.mkv")
	b := mk("B.mp4")
	mk("note.txt")            // non-video, ignored when in a dir
	sub := filepath.Join(d, "sub"); os.Mkdir(sub, 0755)
	c := filepath.Join(sub, "C.mkv"); os.WriteFile(c, []byte("x"), 0644)

	// dir expands recursively to videos only; explicit file added once
	got, err := ResolveTargets([]string{d, a})
	if err != nil { t.Fatal(err) }
	if len(got) != 3 { t.Fatalf("got %d want 3: %v", len(got), got) }

	// explicit non-video file errors
	if _, err := ResolveTargets([]string{filepath.Join(d, "note.txt")}); err == nil {
		t.Error("expected error for non-video file")
	}
	_ = b

	// list file: skip blanks + comments + quotes
	lf := filepath.Join(d, "list.txt")
	os.WriteFile(lf, []byte("# comment\n\n\""+a+"\"\n"+c+"\n"), 0644)
	lines, err := ReadListFile(lf)
	if err != nil { t.Fatal(err) }
	if len(lines) != 2 || lines[0] != a || lines[1] != c {
		t.Fatalf("ReadListFile = %v", lines)
	}
}
