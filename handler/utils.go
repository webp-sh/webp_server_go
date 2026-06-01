package handler

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

func sendNotFound(c *fiber.Ctx) error {
	msg := "Image not found!"
	_ = c.Send([]byte(msg))
	log.Warn(msg)
	_ = c.SendStatus(404)
	return nil
}

func resolveSafeLocalPath(baseDir string, reqPath string) (string, error) {
	decoded, err := url.PathUnescape(reqPath)
	if err != nil {
		return "", err
	}
	if hasTraversalSegments(decoded) {
		return "", fiber.ErrNotFound
	}
	cleaned := path.Clean("/" + decoded)
	relative := strings.TrimPrefix(cleaned, "/")
	candidate := filepath.Join(baseDir, filepath.FromSlash(relative))
	return ensurePathWithinBase(baseDir, candidate)
}

func resolveSafeMappedPath(baseDir string, reqPath string) (string, error) {
	if baseDir == "" {
		return "", fiber.ErrNotFound
	}
	decoded, err := url.PathUnescape(reqPath)
	if err != nil {
		return "", err
	}
	candidate := filepath.Clean(filepath.FromSlash(decoded))
	return ensurePathWithinBase(baseDir, candidate)
}

func ensurePathWithinBase(baseDir string, candidate string) (string, error) {
	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(baseAbs, candidateAbs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fiber.ErrNotFound
	}
	return candidateAbs, nil
}

func hasTraversalSegments(pathValue string) bool {
	decoded, err := url.PathUnescape(pathValue)
	if err == nil {
		pathValue = decoded
	}
	pathValue = strings.ReplaceAll(pathValue, "\\", "/")
	for _, segment := range strings.Split(pathValue, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}
