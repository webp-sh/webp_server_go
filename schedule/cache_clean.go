package schedule

import (
	"os"
	"path/filepath"
	"time"
	"webp_server_go/config"

	log "github.com/sirupsen/logrus"
)

func getDirSize(path string) (int64, error) {
	// Check if path is a directory and exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return 0, nil
	}
	var size int64
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// Delete the oldest file in the given path
func clearDirForOldestFiles(path string) error {
	oldestFile := ""
	oldestModTime := time.Now()

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Errorf("Error accessing path %s: %s\n", path, err.Error())
			return nil
		}

		if !info.IsDir() && info.ModTime().Before(oldestModTime) {
			oldestFile = path
			oldestModTime = info.ModTime()
		}
		return nil
	})

	if err != nil {
		log.Errorf("Error traversing directory: %s\n", err.Error())
		return err
	}

	if oldestFile != "" {
		err := os.Remove(oldestFile)
		if err != nil {
			log.Errorf("Error deleting file %s: %s\n", oldestFile, err.Error())
			return err
		}
		log.Infof("Deleted oldest file: %s\n", oldestFile)
	} else {
		log.Infoln("No files found in the directory.")
	}
	return nil
}

// Clear cache, size is in bytes that needs to be cleared out
// Will delete oldest files first, then second oldest, etc.
// Until all files size are less than maxCacheSizeBytes
func clearCacheFiles(path string, maxCacheSizeBytes int64) error {
	dirSize, err := getDirSize(path)
	if err != nil {
		log.Errorf("Error getting directory size: %s\n", err.Error())
		return err
	}

	for dirSize > maxCacheSizeBytes {
		err := clearDirForOldestFiles(path)
		if err != nil {
			log.Errorf("Error clearing directory: %s\n", err.Error())
			return err
		}
		dirSize, err = getDirSize(path)
		if err != nil {
			log.Errorf("Error getting directory size: %s\n", err.Error())
			return err
		}
	}
	return nil
}

func CleanCache() {
	log.Info("MaxCacheSize is not 0, starting cache cleaning service")
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// MB to bytes
			maxCacheSizeBytes := int64(config.Config.MaxCacheSize) * 1024 * 1024
			err := clearCacheFiles(config.Config.RemoteRawPath, maxCacheSizeBytes)
			if err != nil {
				log.Warn("Failed to clear remote raw cache")
			}
			err = clearCacheFiles(config.Config.ExhaustPath, maxCacheSizeBytes)
			if err != nil && err != os.ErrNotExist {
				log.Warn("Failed to clear remote raw cache")
			}
			err = clearCacheFiles(config.Config.MetadataPath, maxCacheSizeBytes)
			if err != nil && err != os.ErrNotExist {
				log.Warn("Failed to clear remote raw cache")
			}
		}
	}
}
