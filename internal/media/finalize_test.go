package media

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTmp(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func TestPathHelpers(t *testing.T) {
	in := filepath.Join("x", "Movie.avi")
	if got := TempPath(in); filepath.Base(got) != "Movie.plexprep-tmp.mkv" {
		t.Errorf("TempPath = %q", got)
	}
	if got := FinalPath(in, false); filepath.Base(got) != "Movie (plexprep).mkv" {
		t.Errorf("FinalPath sibling = %q", got)
	}
	if got := FinalPath(in, true); filepath.Base(got) != "Movie.mkv" {
		t.Errorf("FinalPath replace = %q", got)
	}
}

func TestFinalizeSibling(t *testing.T) {
	d := t.TempDir()
	in := writeTmp(t, d, "Movie.avi", "source")
	tmp := writeTmp(t, d, "Movie.plexprep-tmp.mkv", "encoded")

	final, err := Finalize(in, tmp, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(final) != "Movie (plexprep).mkv" {
		t.Errorf("final = %q", final)
	}
	if !exists(final) {
		t.Error("output missing")
	}
	if !exists(in) {
		t.Error("source should be kept in sibling mode")
	}
	if exists(tmp) {
		t.Error("temp should be gone")
	}
}

func TestFinalizeSiblingPurge(t *testing.T) {
	d := t.TempDir()
	in := writeTmp(t, d, "Movie.avi", "source")
	tmp := writeTmp(t, d, "Movie.plexprep-tmp.mkv", "encoded")

	final, err := Finalize(in, tmp, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if !exists(final) {
		t.Error("output missing")
	}
	if exists(in) {
		t.Error("source should be deleted with purge")
	}
}

func TestFinalizeReplace(t *testing.T) {
	d := t.TempDir()
	in := writeTmp(t, d, "Movie.mkv", "source")
	tmp := writeTmp(t, d, "Movie.plexprep-tmp.mkv", "encoded")

	final, err := Finalize(in, tmp, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(final) != "Movie.mkv" {
		t.Errorf("final = %q", final)
	}
	if !exists(final) {
		t.Error("output missing")
	}
	bak := filepath.Join(d, "Movie.mkv.original")
	if !exists(bak) {
		t.Error("backup .original should exist in replace mode")
	}
	// The output must be the encoded content, not the old source.
	b, _ := os.ReadFile(final)
	if string(b) != "encoded" {
		t.Errorf("final content = %q, want encoded", b)
	}
}

func TestFinalizeReplacePurge(t *testing.T) {
	d := t.TempDir()
	in := writeTmp(t, d, "Movie.mkv", "source")
	tmp := writeTmp(t, d, "Movie.plexprep-tmp.mkv", "encoded")

	final, err := Finalize(in, tmp, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if !exists(final) {
		t.Error("output missing")
	}
	if exists(filepath.Join(d, "Movie.mkv.original")) {
		t.Error("backup should be deleted with purge")
	}
	// Exactly one file should remain.
	ents, _ := os.ReadDir(d)
	if len(ents) != 1 {
		t.Errorf("expected 1 file, got %d", len(ents))
	}
}

func TestBackupPathCollision(t *testing.T) {
	d := t.TempDir()
	in := writeTmp(t, d, "Movie.mkv", "s")
	writeTmp(t, d, "Movie.mkv.original", "existing")

	got := backupPath(in)
	if filepath.Base(got) != "Movie.mkv.original.1" {
		t.Errorf("collision backup = %q, want Movie.mkv.original.1", got)
	}
}
