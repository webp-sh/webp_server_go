package helper

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var ErrPathTraversal = errors.New("path traversal detected")

const maxUnescapeRounds = 10

// FullyUnescapePath decodes percent-encoding until the string is stable.
func FullyUnescapePath(s string) (string, error) {
	prev := s
	for range maxUnescapeRounds {
		decoded, err := url.PathUnescape(prev)
		if err != nil {
			return "", ErrPathTraversal
		}
		if decoded == prev {
			return decoded, nil
		}
		prev = decoded
	}
	return prev, nil
}

// ResolveUnderBase resolves a URL request path under baseDir.
// It returns the absolute filesystem path and a relative path using forward slashes.
func ResolveUnderBase(baseDir, requestPath string) (absPath string, relPath string, err error) {
	if strings.Contains(requestPath, "\x00") {
		return "", "", ErrPathTraversal
	}

	decoded, err := FullyUnescapePath(requestPath)
	if err != nil {
		return "", "", err
	}

	decoded = filepath.ToSlash(decoded)
	decoded = strings.TrimPrefix(decoded, "/")
	if decoded == "" {
		return "", "", ErrPathTraversal
	}

	cleaned := filepath.Clean(decoded)
	if strings.HasPrefix(cleaned, "..") {
		return "", "", ErrPathTraversal
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", "", err
	}
	realBase, err := resolvePathWithSymlinks(absBase)
	if err != nil {
		return "", "", err
	}

	candidate := filepath.Join(absBase, cleaned)
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", "", err
	}
	realCandidate, err := resolvePathWithSymlinks(absCandidate)
	if err != nil {
		return "", "", err
	}

	rel, err := filepath.Rel(absBase, absCandidate)
	if err != nil {
		return "", "", ErrPathTraversal
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", ErrPathTraversal
	}
	realRel, err := filepath.Rel(realBase, realCandidate)
	if err != nil {
		return "", "", ErrPathTraversal
	}
	if realRel == ".." || strings.HasPrefix(realRel, ".."+string(filepath.Separator)) {
		return "", "", ErrPathTraversal
	}

	return absCandidate, filepath.ToSlash(rel), nil
}

// RelPathUnderBase returns the relative path of absPath under baseDir.
func RelPathUnderBase(baseDir, absPath string) (string, error) {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	absCandidate, err := filepath.Abs(absPath)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(absBase, absCandidate)
	if err != nil {
		return "", ErrPathTraversal
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrPathTraversal
	}
	realBase, err := resolvePathWithSymlinks(absBase)
	if err != nil {
		return "", err
	}
	realCandidate, err := resolvePathWithSymlinks(absCandidate)
	if err != nil {
		return "", err
	}
	realRel, err := filepath.Rel(realBase, realCandidate)
	if err != nil {
		return "", ErrPathTraversal
	}
	if realRel == ".." || strings.HasPrefix(realRel, ".."+string(filepath.Separator)) {
		return "", ErrPathTraversal
	}
	return filepath.ToSlash(rel), nil
}

// DecodeRequestPath fully unescapes a URL path and returns a slash-normalized path
// with a leading slash, for prefix matching (e.g. IMG_MAP).
func DecodeRequestPath(requestPath string) (string, error) {
	decoded, err := FullyUnescapePath(requestPath)
	if err != nil {
		return "", err
	}
	decoded = filepath.ToSlash(decoded)
	if !strings.HasPrefix(decoded, "/") {
		decoded = "/" + decoded
	}
	return filepath.Clean(decoded), nil
}

// BuildQueryKey builds the canonical metadata cache query suffix.
func BuildQueryKey(width, height, maxWidth, maxHeight string) string {
	return "width=" + width + "&height=" + height + "&max_width=" + maxWidth + "&max_height=" + maxHeight
}

func resolvePathWithSymlinks(absPath string) (string, error) {
	cleanPath := filepath.Clean(absPath)
	resolved, err := filepath.EvalSymlinks(cleanPath)
	if err == nil {
		return resolved, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}

	parent := filepath.Dir(cleanPath)
	resolvedParent, parentErr := filepath.EvalSymlinks(parent)
	if parentErr != nil {
		if os.IsNotExist(parentErr) {
			return cleanPath, nil
		}
		return "", parentErr
	}
	return filepath.Join(resolvedParent, filepath.Base(cleanPath)), nil
}
