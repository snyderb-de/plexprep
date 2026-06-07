package media

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TempPath is where an encode is written before it is swapped into place.
// Always .mkv (our forced output container).
func TempPath(in string) string {
	dir := filepath.Dir(in)
	base := strings.TrimSuffix(filepath.Base(in), filepath.Ext(in))
	return filepath.Join(dir, base+".plexprep-tmp.mkv")
}

// FinalPath is the destination for a finished encode.
//   - replace=false → sibling "<name> (plexprep).mkv"
//   - replace=true  → "<name>.mkv" (takes the source's name)
func FinalPath(in string, replace bool) string {
	dir := filepath.Dir(in)
	base := strings.TrimSuffix(filepath.Base(in), filepath.Ext(in))
	if replace {
		return filepath.Join(dir, base+".mkv")
	}
	return filepath.Join(dir, base+" (plexprep).mkv")
}

// backupPath returns a non-colliding "<in>.original" path for the source.
func backupPath(in string) string {
	cand := in + ".original"
	for i := 1; ; i++ {
		if _, err := os.Stat(cand); os.IsNotExist(err) {
			return cand
		}
		cand = fmt.Sprintf("%s.original.%d", in, i)
	}
}

// Finalize moves a finished temp encode to its final location.
//
// replace=false: rename temp → "<name> (plexprep).mkv".
//
// replace=true: the output takes the source's name (as .mkv). The source is
// first renamed to "<name>.original" (so the swap is atomic and reversible).
//
// purge: delete the source immediately after the output is safely in place,
// freeing its space mid-batch instead of keeping a backup. Irreversible. The
// source is only ever removed once the new file exists, never before.
func Finalize(in, tmp string, replace, purge bool) (string, error) {
	final := FinalPath(in, replace)

	if !replace {
		if err := os.Rename(tmp, final); err != nil {
			return "", err
		}
		if purge {
			_ = os.Remove(in) // output is a sibling; drop the original
		}
		return final, nil
	}

	// Move the source aside first so the swap is reversible on failure.
	bak := backupPath(in)
	if err := os.Rename(in, bak); err != nil {
		return "", fmt.Errorf("backup source: %w", err)
	}
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Rename(bak, in) // roll back, nothing lost
		return "", fmt.Errorf("place output: %w", err)
	}
	if purge {
		_ = os.Remove(bak) // new file is in place; drop the backup now
	}
	return final, nil
}
