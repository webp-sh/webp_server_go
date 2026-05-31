package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
	"webp_server_go/config"

	"github.com/gofiber/fiber/v2"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTraversalEnv(t *testing.T) (imgPath string, secretFile string) {
	t.Helper()

	baseDir := t.TempDir()
	imgPath = filepath.Join(baseDir, "pics")
	secretDir := filepath.Join(baseDir, "secret")
	require.NoError(t, os.MkdirAll(imgPath, 0o755))
	require.NoError(t, os.MkdirAll(secretDir, 0o755))

	allowed := filepath.Join(imgPath, "allowed.jpg")
	secretFile = filepath.Join(secretDir, "leaked.jpg")

	src, err := os.ReadFile("../pics/webp_server.jpg")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(allowed, src, 0o644))
	require.NoError(t, os.WriteFile(secretFile, src, 0o644))

	config.Config.ImgPath = imgPath
	config.Config.ExhaustPath = filepath.Join(baseDir, "exhaust")
	config.Config.AllowedTypes = []string{"jpg", "png", "jpeg", "bmp", "heic", "avif"}
	config.Config.MetadataPath = filepath.Join(baseDir, "metadata")
	config.Config.RemoteRawPath = filepath.Join(baseDir, "remote-raw")
	config.ProxyMode = false
	config.Config.EnableWebP = true
	config.Config.EnableAVIF = false
	config.Config.Quality = 80
	config.Config.ImageMap = map[string]string{}
	config.RemoteCache = cache.New(cache.NoExpiration, 10*time.Minute)

	return imgPath, secretFile
}

func requestRawPath(app *fiber.App, rawPath string, query string) (*http.Response, []byte) {
	target := rawPath
	if query != "" {
		target = rawPath + "?" + query
	}
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set("User-Agent", chromeUA)
	req.Header.Set("Accept", acceptWebP)
	req.Header.Set("Host", "127.0.0.1:3333")
	req.Host = "127.0.0.1:3333"
	resp, err := app.Test(req, 120000)
	if err != nil {
		return nil, nil
	}
	data, _ := io.ReadAll(resp.Body)
	return resp, data
}

func TestDirectoryTraversalMetadataBlocked(t *testing.T) {
	setupTraversalEnv(t)

	app := fiber.New()
	app.Get("/*", Convert)

	traversalCases := []string{
		"/%252E%252E%252Fsecret/leaked.jpg",
		"/%2E%2E%2Fsecret/leaked.jpg",
		"/../secret/leaked.jpg",
		"/..%2Fsecret/leaked.jpg",
	}

	for _, rawPath := range traversalCases {
		t.Run(rawPath, func(t *testing.T) {
			resp, body := requestRawPath(app, rawPath, "meta=full")
			require.NotNil(t, resp)

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
				"traversal request must be rejected, body=%s", string(body))
			assert.NotContains(t, string(body), `"width"`, "must not leak metadata JSON")
		})
	}
}

func TestDirectoryTraversalNormalRequestBlocked(t *testing.T) {
	setupTraversalEnv(t)

	app := fiber.New()
	app.Get("/*", Convert)

	resp, _ := requestRawPath(app, "/%252E%252E%252Fsecret/leaked.jpg", "")
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAllowedImageStillWorks(t *testing.T) {
	setupTraversalEnv(t)

	app := fiber.New()
	app.Get("/*", Convert)

	resp, body := requestRawPath(app, "/allowed.jpg", "meta=full")
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var meta map[string]any
	require.NoError(t, json.Unmarshal(body, &meta))
	assert.NotZero(t, meta["width"])
	assert.NotEmpty(t, meta["format"])
}
