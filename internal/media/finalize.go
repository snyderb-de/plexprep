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
// replace=false: rename temp → "<name> (plexprep).mkv". Source untouched.
//
// replace=true: rename source → "<name>.original" (kept as backup), then
// rename temp → "<name>.mkv". The source is preserved, not deleted, so the
// result can be eyeballed before the backups are purged.
func Finalize(in, tmp string, replace bool) (string, error) {
	final := FinalPath(in, replace)

	if !replace {
		if err := os.Rename(tmp, final); err != nil {
			return "", err
		}
		return final, nil
	}

	// Back up the source first.
	bak := backupPath(in)
	if err := os.Rename(in, bak); err != nil {
		return "", fmt.Errorf("backup source: %w", err)
	}
	// Swap the encode into the source's name.
	if err := os.Rename(tmp, final); err != nil {
		// Roll back the backup so nothing is lost.
		_ = os.Rename(bak, in)
		return "", fmt.Errorf("place output: %w", err)
	}
	return final, nil
}
