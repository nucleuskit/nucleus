package verify

import (
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"sort"
)

type fileSnapshot struct {
	exists bool
	hash   [sha256.Size]byte
}

func runTidyCommand(dir string) verifyStep {
	before, err := snapshotModuleFiles(dir)
	if err != nil {
		return verifyStep{
			Phase:   phaseTidy,
			Command: "go mod tidy",
			OK:      false,
			Error:   "read module files failed",
		}
	}
	step := runGoCommand(dir, phaseTidy, []string{"mod", "tidy"})
	after, err := snapshotModuleFiles(dir)
	if err != nil {
		step.OK = false
		step.Error = "read module files failed"
		return step
	}
	step.ChangedPaths = changedModuleFiles(before, after)
	if step.OK && len(step.ChangedPaths) > 0 {
		step.OK = false
		step.Error = "go mod tidy changed module files"
	}
	return step
}

func snapshotModuleFiles(dir string) (map[string]fileSnapshot, error) {
	files := []string{"go.mod", "go.sum"}
	snapshots := make(map[string]fileSnapshot, len(files))
	for _, name := range files {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				snapshots[name] = fileSnapshot{}
				continue
			}
			return nil, err
		}
		snapshots[name] = fileSnapshot{
			exists: true,
			hash:   sha256.Sum256(data),
		}
	}
	return snapshots, nil
}

func changedModuleFiles(before map[string]fileSnapshot, after map[string]fileSnapshot) []string {
	seen := map[string]struct{}{}
	for name := range before {
		seen[name] = struct{}{}
	}
	for name := range after {
		seen[name] = struct{}{}
	}
	changed := make([]string, 0, len(seen))
	for name := range seen {
		left := before[name]
		right := after[name]
		if left.exists != right.exists || left.hash != right.hash {
			changed = append(changed, name)
		}
	}
	sort.Strings(changed)
	return changed
}
