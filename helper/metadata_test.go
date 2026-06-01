package helper

import (
	"net/url"
	"path"
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
