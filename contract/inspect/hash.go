package inspect

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const FreshnessMarker = ".nucleus-source.sha256"

func ContractSourceHash(dir string) (string, error) {
	paths := []string{
		filepath.Join(dir, filepath.FromSlash(contractPathOpenAPI)),
		filepath.Join(dir, filepath.FromSlash(contractPathErrors)),
	}
	protoDir := filepath.Join(dir, filepath.FromSlash(contractProtoDir))
	if entries, err := os.ReadDir(protoDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), contractProtoFileSuffix) {
				paths = append(paths, filepath.Join(protoDir, entry.Name()))
			}
		}
	}
	sort.Strings(paths)

	hash := sha256.New()
	found := false
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", err
		}
		found = true
		_, _ = io.WriteString(hash, filepath.ToSlash(strings.TrimPrefix(path, dir)))
		if _, err := io.Copy(hash, file); err != nil {
			_ = file.Close()
			return "", err
		}
		_ = file.Close()
	}
	if !found {
		return "", nil
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func GeneratedMarkerPath(dir, target string) string {
	targetPath := filepath.Join(dir, target)
	if info, err := os.Stat(targetPath); err == nil && info.IsDir() {
		return filepath.Join(targetPath, FreshnessMarker)
	}
	return targetPath + ".sha256"
}

func ReadGeneratedHash(dir, target string) (string, error) {
	data, err := os.ReadFile(GeneratedMarkerPath(dir, target))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
