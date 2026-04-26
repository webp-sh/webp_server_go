package schedule

import (
	"os"
	"path/filepath"
	"sort"
	"time"
	"webp_server_go/config"

	log "github.com/sirupsen/logrus"
)

// getDirSize returns total size in bytes of all regular files under path.
// If path does not exist, returns 0 without error.
func getDirSize(path string) (int64, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return 0, nil
	}

	var size int64
	err := filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// fileInfo holds path and modification time for sorting.
type fileInfo struct {
	path string
	mod  time.Time
	size int64
}

// listFiles returns a slice of regular files under path with their mod times.
// If path does not exist, returns empty slice and nil error.
func listFiles(path string) ([]fileInfo, error) {
	var files []fileInfo
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return files, nil
	}

	err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			// log and continue walking
			log.Debugf("walk error %s: %v", p, err)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			log.Debugf("stat error %s: %v", p, err)
			return nil
		}
		files = append(files, fileInfo{path: p, mod: info.ModTime(), size: info.Size()})
		return nil
	})
	return files, err
}

// removeOldest removes files from the directory in ascending mod-time order
// until the total size is <= maxCacheSizeBytes.
func removeOldest(dir string, maxCacheSizeBytes int64) error {
	dirSize, err := getDirSize(dir)
	if err != nil {
		return err
	}
	if dirSize <= maxCacheSizeBytes {
		return nil
	}

	files, err := listFiles(dir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}

	// sort by modification time ascending (oldest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].mod.Before(files[j].mod)
	})

	for _, f := range files {
		if dirSize <= maxCacheSizeBytes {
			break
		}
		if err := os.Remove(f.path); err != nil {
			log.Errorf("failed to delete file %s: %v", f.path, err)
			// continue trying other files
			continue
		}
		dirSize -= f.size
		log.Infof("deleted cached file: %s", f.path)
	}
	return nil
}

// CleanCache periodically enforces MaxCacheSize on configured cache paths.
// Runs until the process exits.
func CleanCache() {
	if config.Config.MaxCacheSize == 0 {
		return
	}
	log.Info("starting cache cleaning service")
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		maxBytes := int64(config.Config.MaxCacheSize) * 1024 * 1024
		paths := []string{
			config.Config.RemoteRawPath,
			config.Config.ExhaustPath,
			config.Config.MetadataPath,
		}
		for _, p := range paths {
			if err := removeOldest(p, maxBytes); err != nil {
				// ignore not-exist errors, warn on others
				if !os.IsNotExist(err) {
					log.Warnf("failed to clear cache at %s: %v", p, err)
				}
			}
		}
	}
}

// DeleteDeadCache removes stale temporary vips-* directories older than threshold.
func DeleteDeadCache() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	threshold := time.Now().Add(-10 * time.Minute)
	tempBase := filepath.Join(os.TempDir(), "vips-")

	for range ticker.C {
		_ = filepath.WalkDir(tempBase, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				info, err := d.Info()
				if err != nil {
					return nil
				}
				if info.ModTime().Before(threshold) {
					log.Warnf("deleting stale temp dir: %s", p)
					_ = os.RemoveAll(p)
					return filepath.SkipDir
				}
			}
			return nil
		})
	}
}
