package helper

import (
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"webp_server_go/config"
)

func TestGetId(t *testing.T) {
	config.Config.ImgPath = "./pics"
	config.Config.RemoteRawPath = "remote-raw"
	p := "https://example.com/image.jpg?width=200&height=300"

	t.Run("remote url", func(t *testing.T) {
		// Test case 1: Remote URL
		id, jointPath, santizedPath := getId(p, "")

		// Verify the return values
		expectedId := HashString(p)
		expectedPath := "remote-raw/8d8576343c4cb816.jpg?width=200&height=300"
		expectedSantizedPath := ""
		if id != expectedId || jointPath != expectedPath || santizedPath != expectedSantizedPath {
			t.Errorf("Test case 1 failed: Expected (%s, %s, %s), but got (%s, %s, %s)",
				expectedId, expectedPath, expectedSantizedPath, id, jointPath, santizedPath)
		}
	})
	t.Run("local path", func(t *testing.T) {
		// Test case 2: Local path
		p = "/image.jpg?width=400&height=500"
		id, jointPath, santizedPath := getId(p, "")

		// Verify the return values
		parsed, _ := url.Parse(p)
		expectedId := HashString(parsed.Path + "?width=400&height=500&max_width=&max_height=")
		expectedPath := path.Join(config.Config.ImgPath, parsed.Path)
		expectedSantizedPath := parsed.Path + "?width=400&height=500&max_width=&max_height="
		if id != expectedId || jointPath != expectedPath || santizedPath != expectedSantizedPath {
			t.Errorf("Test case 2 failed: Expected (%s, %s, %s), but got (%s, %s, %s)",
				expectedId, expectedPath, expectedSantizedPath, id, jointPath, santizedPath)
		}
	})
}

func TestWriteAndReadMetadataSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	imgDir := filepath.Join(tmpDir, "pics")
	metadataDir := filepath.Join(tmpDir, "metadata")

	config.Config.ImgPath = imgDir
	config.Config.MetadataPath = metadataDir
	config.Config.RemoteRawPath = filepath.Join(tmpDir, "remote-raw")

	requireNoError := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	requireNoError(os.MkdirAll(imgDir, 0o755))
	imagePath := filepath.Join(imgDir, "test.jpg")
	requireNoError(os.WriteFile(imagePath, []byte("fake image bytes"), 0o600))

	// Use URI-style local path to match getId() local branch behavior.
	uri := "/test.jpg?width=200"

	written, err := WriteMetadata(uri, "", config.LocalHostAlias)
	requireNoError(err)
	if written.Id == "" {
		t.Fatalf("expected metadata id to be set")
	}

	read, err := ReadMetadata(uri, "", config.LocalHostAlias)
	requireNoError(err)
	if read.Id != written.Id {
		t.Fatalf("expected same metadata id, got %s and %s", read.Id, written.Id)
	}
}

func TestReadMetadataReturnsErrorWhenMetadataPathUnavailable(t *testing.T) {
	tmpDir := t.TempDir()
	imgDir := filepath.Join(tmpDir, "pics")
	blockedPath := filepath.Join(tmpDir, "metadata-file")

	config.Config.ImgPath = imgDir
	config.Config.MetadataPath = blockedPath
	config.Config.RemoteRawPath = filepath.Join(tmpDir, "remote-raw")

	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		t.Fatalf("failed to create image dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(imgDir, "test.jpg"), []byte("fake image bytes"), 0o600); err != nil {
		t.Fatalf("failed to write test image: %v", err)
	}
	// Make metadata base path a regular file so MkdirAll(path.Join(base, subdir)) fails.
	if err := os.WriteFile(blockedPath, []byte("not a dir"), 0o600); err != nil {
		t.Fatalf("failed to create blocked metadata path: %v", err)
	}

	_, err := ReadMetadata("/test.jpg?width=100", "", config.LocalHostAlias)
	if err == nil {
		t.Fatalf("expected error when metadata path is unavailable")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "metadata") {
		t.Fatalf("expected metadata-related error, got: %v", err)
	}
}
