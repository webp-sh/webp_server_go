package handler

import (
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
	"webp_server_go/config"
	"webp_server_go/helper"

	"github.com/gofiber/fiber/v2"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO7+f6kAAAAASUVORK5CYII="

func setupScenarioConfig(t *testing.T, baseDir string) {
	t.Helper()

	config.Config.ImgPath = baseDir
	config.Config.ExhaustPath = filepath.Join(baseDir, "_exhaust")
	config.Config.MetadataPath = filepath.Join(baseDir, "_metadata")
	config.Config.RemoteRawPath = filepath.Join(baseDir, "_remote_raw")
	config.Config.AllowedTypes = []string{"jpg", "png", "jpeg", "bmp", "heic", "avif", "gif"}
	config.Config.EnableWebP = true
	config.Config.EnableAVIF = false
	config.Config.EnableJXL = false
	config.Config.EnableExtraParams = true
	config.Config.Quality = 80
	config.Config.CacheTTL = 4320
	config.Config.ImageMap = map[string]string{}
	config.AllowAllExtensions = false
	config.RemoteCache = cache.New(cache.NoExpiration, 10*time.Minute)
}

func writeTinyImage(t *testing.T, destPath string, encoded string) {
	t.Helper()

	data, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(destPath), 0o755))
	require.NoError(t, os.WriteFile(destPath, data, 0o600))
}

func newScenarioApp() *fiber.App {
	app := fiber.New()
	app.Get("/*", Convert)
	return app
}

// TestScenarioLocalSingleDirectory verifies the single local directory scenario.
//
// ASCII flow:
//   client(/path/tsuki.png?width=320)
//                |
//                v
//       IMG_PATH=<local_root>
//                |
//                v
//   <local_root>/path/tsuki.png -> Convert -> image response
//
// Assertions:
// - request returns 200;
// - content type is image/*;
// - response bytes are detected as a valid image payload.
func TestScenarioLocalSingleDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	setupScenarioConfig(t, tmpDir)
	writeTinyImage(t, filepath.Join(tmpDir, "path", "tsuki.png"), tinyPNGBase64)

	resp, data := requestToServer("http://127.0.0.1:3333/path/tsuki.png?width=320", newScenarioApp(), chromeUA, acceptWebP)
	require.NotNil(t, resp)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "image/")
	assert.NotEmpty(t, helper.GetContentType(data))
}

// TestScenarioLocalMultiDirectoryWithIMGMap verifies local multi-directory routing with IMG_MAP.
//
// ASCII flow:
//   /avatars/u1.png  ---->  /avatars  ----> <avatars_dir>/u1.png
//   /products/p1.png ---->  /products ----> <products_dir>/p1.png
//   /products/miss.jpg ----> /products ----> not found (404)
//
// Assertions:
// - mapped resources under both prefixes return 200 with image content;
// - a missing resource under a mapped prefix returns 404.
func TestScenarioLocalMultiDirectoryWithIMGMap(t *testing.T) {
	tmpDir := t.TempDir()
	avatarsDir := filepath.Join(tmpDir, "avatars")
	productsDir := filepath.Join(tmpDir, "products")
	setupScenarioConfig(t, filepath.Join(tmpDir, "default_root"))

	config.Config.ImageMap = map[string]string{
		"/avatars":  avatarsDir,
		"/products": productsDir,
	}
	writeTinyImage(t, filepath.Join(avatarsDir, "u1.png"), tinyPNGBase64)
	writeTinyImage(t, filepath.Join(productsDir, "p1.png"), tinyPNGBase64)

	app := newScenarioApp()

	resp1, _ := requestToServer("http://127.0.0.1:3333/avatars/u1.png", app, chromeUA, acceptWebP)
	require.NotNil(t, resp1)
	defer resp1.Body.Close()
	assert.Equal(t, http.StatusOK, resp1.StatusCode)
	assert.Contains(t, resp1.Header.Get("Content-Type"), "image/")

	resp2, _ := requestToServer("http://127.0.0.1:3333/products/p1.png", app, chromeUA, acceptWebP)
	require.NotNil(t, resp2)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
	assert.Contains(t, resp2.Header.Get("Content-Type"), "image/")

	resp3, _ := requestToServer("http://127.0.0.1:3333/products/not-exists.jpg", app, chromeUA, acceptWebP)
	require.NotNil(t, resp3)
	defer resp3.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp3.StatusCode)
}

// TestScenarioRemoteBackendAddress verifies remote backend mode via IMG_PATH.
//
// ASCII flow:
//   client(/repo/path/file.jpg)
//             |
//             v
//   IMG_PATH=https://raw.githubusercontent.com
//             |
//             v
//   fetch upstream -> Convert/cache -> response
//
// Assertions:
// - existing upstream image returns 200 with image content;
// - missing upstream image returns 404.
func TestScenarioRemoteBackendAddress(t *testing.T) {
	tmpDir := t.TempDir()
	setupScenarioConfig(t, tmpDir)
	config.Config.ImgPath = "https://raw.githubusercontent.com"

	app := newScenarioApp()

	resp1, _ := requestToServer("http://127.0.0.1:3333/webp-sh/webp_server_go/master/pics/webp_server.jpg", app, chromeUA, acceptWebP)
	require.NotNil(t, resp1)
	defer resp1.Body.Close()
	assert.Equal(t, http.StatusOK, resp1.StatusCode)
	assert.Contains(t, resp1.Header.Get("Content-Type"), "image/")

	resp2, _ := requestToServer("http://127.0.0.1:3333/webp-sh/webp_server_go/master/pics/not-exists-404-check.jpg", app, chromeUA, acceptWebP)
	require.NotNil(t, resp2)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
}

// TestScenarioMixedLocalAndRemoteMappings verifies mixed local + remote IMG_MAP rules.
//
// ASCII flow:
//   /legacy/a.png ----> /legacy ----> <legacy_dir>/a.png ----\
//                                                         Convert -> image response
//   /cdn/...jpg  -----> /cdn    ----> https://raw.githubusercontent.com/... --/
//
// Assertions:
// - local mapped path returns 200 with image content;
// - remote mapped path returns 200 with image content;
// - both paths are handled by the same conversion pipeline entrypoint.
func TestScenarioMixedLocalAndRemoteMappings(t *testing.T) {
	tmpDir := t.TempDir()
	legacyDir := filepath.Join(tmpDir, "legacy-images")
	setupScenarioConfig(t, filepath.Join(tmpDir, "default_root"))
	writeTinyImage(t, filepath.Join(legacyDir, "a.png"), tinyPNGBase64)

	config.Config.ImageMap = map[string]string{
		"/legacy": legacyDir,
		"/cdn":    "https://raw.githubusercontent.com",
	}

	app := newScenarioApp()

	localResp, _ := requestToServer("http://127.0.0.1:3333/legacy/a.png", app, chromeUA, acceptWebP)
	require.NotNil(t, localResp)
	defer localResp.Body.Close()
	assert.Equal(t, http.StatusOK, localResp.StatusCode)
	assert.Contains(t, localResp.Header.Get("Content-Type"), "image/")

	remoteResp, _ := requestToServer("http://127.0.0.1:3333/cdn/webp-sh/webp_server_go/master/pics/webp_server.jpg", app, chromeUA, acceptWebP)
	require.NotNil(t, remoteResp)
	defer remoteResp.Body.Close()
	assert.Equal(t, http.StatusOK, remoteResp.StatusCode)
	assert.Contains(t, remoteResp.Header.Get("Content-Type"), "image/")
}
