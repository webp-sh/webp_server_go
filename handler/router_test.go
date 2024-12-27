package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
	"webp_server_go/config"
	"webp_server_go/helper"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

var (
	chromeUA   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.183 Safari/537.36"
	safariUA   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1 Safari/605.1.15"
	safari17UA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3.1 Safari/605.1.15" // <- Mac with Safari 17
	curlUA     = "curl/7.64.1"

	acceptWebP   = "image/webp,image/apng,image/*,*/*;q=0.8"
	acceptAvif   = "image/avif,image/*,*/*;q=0.8"
	acceptLegacy = "image/jpeg,image/png"
)

func setupParam() {
	// setup parameters here...
	config.Config.ImgPath = "../pics"
	config.Config.ExhaustPath = "../exhaust_test"
	config.Config.AllowedTypes = []string{"jpg", "png", "jpeg", "bmp", "heic", "avif"}
	config.Config.MetadataPath = "../metadata"
	config.Config.RemoteRawPath = "../remote-raw"
	config.ProxyMode = false
	config.Config.EnableWebP = true
	config.Config.EnableAVIF = false
	config.Config.Quality = 80
	config.Config.ImageMap = map[string]string{}
	config.RemoteCache = cache.New(cache.NoExpiration, 10*time.Minute)
}

func requestToServer(reqUrl string, app *fiber.App, ua, accept string) (*http.Response, []byte) {
	parsedUrl, _ := url.Parse(reqUrl)
	req := httptest.NewRequest("GET", parsedUrl.EscapedPath(), nil)
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", accept)
	req.Header.Set("Host", parsedUrl.Host)
	req.Host = parsedUrl.Host
	resp, err := app.Test(req, 120000)
	if err != nil {
		return nil, nil
	}
	data, _ := io.ReadAll(resp.Body)
	return resp, data
}

func TestServerHeaders(t *testing.T) {
	setupParam()
	var app = fiber.New()
	app.Use(etag.New(etag.Config{
		Weak: true,
	}))
	app.Get("/*", Convert)
	url := "http://127.0.0.1:3333/webp_server.bmp"

	// test for chrome
	response, _ := requestToServer(url, app, chromeUA, acceptWebP)
	defer response.Body.Close()
	ratio := response.Header.Get("X-Compression-Rate")
	etag := response.Header.Get("Etag")

	assert.NotEqual(t, "", ratio)
	assert.NotEqual(t, "", etag)

	// test for safari
	response, _ = requestToServer(url, app, safariUA, acceptLegacy)
	defer response.Body.Close()
	// ratio = response.Header.Get("X-Compression-Rate")
	etag = response.Header.Get("Etag")

	assert.NotEqual(t, "", etag)
}

func TestConvertDuplicates(t *testing.T) {
	setupParam()
	N := 3

	var testLink = map[string]string{
		"http://127.0.0.1:3333/webp_server.jpg":                 "image/webp",
		"http://127.0.0.1:3333/webp_server.bmp":                 "image/webp",
		"http://127.0.0.1:3333/webp_server.png":                 "image/webp",
		"http://127.0.0.1:3333/empty.jpg":                       "",
		"http://127.0.0.1:3333/png.jpg":                         "image/webp",
		"http://127.0.0.1:3333/12314.jpg":                       "",
		"http://127.0.0.1:3333/dir1/inside.jpg":                 "image/webp",
		"http://127.0.0.1:3333/%e5%a4%aa%e7%a5%9e%e5%95%a6.png": "image/webp",
		"http://127.0.0.1:3333/太神啦.png":                         "image/webp",
	}

	var app = fiber.New()
	app.Get("/*", Convert)

	// test Chrome
	for url, respType := range testLink {
		for range N {
			resp, data := requestToServer(url, app, chromeUA, acceptWebP)
			defer resp.Body.Close()
			contentType := helper.GetContentType(data)
			assert.Equal(t, respType, contentType)
		}
	}

}
func TestConvert(t *testing.T) {
	setupParam()
	// TODO: old-style test, better update it with accept headers
	var testChromeLink = map[string]string{
		"http://127.0.0.1:3333/webp_server.jpg":                 "image/webp",
		"http://127.0.0.1:3333/webp_server.bmp":                 "image/webp",
		"http://127.0.0.1:3333/webp_server.png":                 "image/webp",
		"http://127.0.0.1:3333/empty.jpg":                       "",
		"http://127.0.0.1:3333/png.jpg":                         "image/webp",
		"http://127.0.0.1:3333/12314.jpg":                       "",
		"http://127.0.0.1:3333/dir1/inside.jpg":                 "image/webp",
		"http://127.0.0.1:3333/%e5%a4%aa%e7%a5%9e%e5%95%a6.png": "image/webp",
		"http://127.0.0.1:3333/太神啦.png":                         "image/webp",
		// Source: https://filesamples.com/formats/heic
		"http://127.0.0.1:3333/sample3.heic": "image/webp", // webp because browser does not support heic
		// Source: https://raw.githubusercontent.com/link-u/avif-sample-images/refs/heads/master/kimono.avif
		"http://127.0.0.1:3333/kimono.avif": "image/webp", // webp because browser does not support avif
	}

	var testChromeAvifLink = map[string]string{
		"http://127.0.0.1:3333/webp_server.jpg":                 "image/avif",
		"http://127.0.0.1:3333/webp_server.bmp":                 "image/avif",
		"http://127.0.0.1:3333/webp_server.png":                 "image/avif",
		"http://127.0.0.1:3333/empty.jpg":                       "",
		"http://127.0.0.1:3333/png.jpg":                         "image/avif",
		"http://127.0.0.1:3333/12314.jpg":                       "",
		"http://127.0.0.1:3333/dir1/inside.jpg":                 "image/avif",
		"http://127.0.0.1:3333/%e5%a4%aa%e7%a5%9e%e5%95%a6.png": "image/avif",
		"http://127.0.0.1:3333/太神啦.png":                         "image/avif",
	}

	var testSafariLink = map[string]string{
		"http://127.0.0.1:3333/webp_server.jpg": "image/jpeg",
		"http://127.0.0.1:3333/webp_server.bmp": "image/png", // png instead oft bmp because ResizeItself() uses ExportNative()
		"http://127.0.0.1:3333/webp_server.png": "image/png",
		"http://127.0.0.1:3333/empty.jpg":       "",
		"http://127.0.0.1:3333/png.jpg":         "image/png",
		"http://127.0.0.1:3333/12314.jpg":       "",
		"http://127.0.0.1:3333/dir1/inside.jpg": "image/jpeg",
	}

	var app = fiber.New()
	app.Get("/*", Convert)

	// // test Chrome
	for url, respType := range testChromeLink {
		resp, data := requestToServer(url, app, chromeUA, acceptWebP)
		defer resp.Body.Close()
		contentType := helper.GetContentType(data)
		assert.Equal(t, respType, contentType)
	}

	// test Safari
	for url, respType := range testSafariLink {
		resp, data := requestToServer(url, app, safariUA, acceptLegacy)
		defer resp.Body.Close()
		contentType := helper.GetContentType(data)
		assert.Equal(t, respType, contentType)
	}

	// test Avif is processed in proxy mode
	config.Config.EnableAVIF = true
	for url, respType := range testChromeAvifLink {
		resp, data := requestToServer(url, app, chromeUA, acceptAvif)
		defer resp.Body.Close()
		contentType := helper.GetContentType(data)
		assert.NotNil(t, respType)
		assert.Equal(t, respType, contentType)
	}
}

func TestConvertNotAllowed(t *testing.T) {
	setupParam()
	config.Config.AllowedTypes = []string{"jpg", "png", "jpeg"}

	var app = fiber.New()
	app.Get("/*", Convert)

	// not allowed, but we have the file, this should return File extension not allowed
	url := "http://127.0.0.1:3333/webp_server.bmp"
	resp, data := requestToServer(url, app, chromeUA, acceptWebP)
	defer resp.Body.Close()
	assert.Contains(t, string(data), "File extension not allowed")

	// not allowed, but we have the file, this should return File extension not allowed
	url = "http://127.0.0.1:3333/config.json"
	resp, data = requestToServer(url, app, chromeUA, acceptWebP)
	defer resp.Body.Close()
	assert.Contains(t, string(data), "File extension not allowed")

	// not allowed, random file
	url = url + "hagdgd"
	resp, data = requestToServer(url, app, chromeUA, acceptWebP)
	defer resp.Body.Close()
	assert.Contains(t, string(data), "File extension not allowed")
}

func TestConvertPassThrough(t *testing.T) {
	setupParam()
	config.Config.AllowedTypes = []string{"*"}

	var app = fiber.New()
	app.Get("/*", Convert)

	// not allowed, but we have the file, this should return File extension not allowed
	url := "http://127.0.0.1:3333/config.json"
	resp, data := requestToServer(url, app, chromeUA, acceptWebP)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Contains(t, string(data), "HOST")
}

func TestConvertProxyModeBad(t *testing.T) {
	setupParam()
	config.ProxyMode = true

	var app = fiber.New()
	app.Get("/*", Convert)

	// this is local random image, should be 404
	url := "http://127.0.0.1:3333/webp_8888server.bmp"
	resp, _ := requestToServer(url, app, chromeUA, acceptWebP)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// this is local random image, test using cURL, should be 404, ref: https://github.com/webp-sh/webp_server_go/issues/197
	resp1, _ := requestToServer(url, app, curlUA, acceptWebP)
	defer resp1.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp1.StatusCode)

}

func TestConvertProxyModeWork(t *testing.T) {
	setupParam()
	config.ProxyMode = true
	config.Config.ImgPath = "https://docs.webp.sh"

	var app = fiber.New()
	app.Get("/*", Convert)

	url := "http://127.0.0.1:3333/images/webp_server.jpg"

	resp, data := requestToServer(url, app, chromeUA, acceptWebP)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "image/webp", helper.GetContentType(data))

	// test proxyMode with Safari
	resp, data = requestToServer(url, app, safariUA, acceptLegacy)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "image/jpeg", helper.GetContentType(data))
}

func TestConvertProxyImgMap(t *testing.T) {
	setupParam()
	config.ProxyMode = false
	config.Config.ImageMap = map[string]string{
		"/2":                            "../pics/dir1",
		"/3":                            "../pics3",             // Invalid path, does not exists
		"www.invalid-path.com":          "https://docs.webp.sh", // Invalid, it does not start with '/'
		"/www.weird-path.com":           "https://docs.webp.sh",
		"/www.even-more-werid-path.com": "https://docs.webp.sh/images",
		"http://example.com":            "https://docs.webp.sh",
	}

	var app = fiber.New()
	app.Get("/*", Convert)

	var testUrls = map[string]string{
		"http://127.0.0.1:3333/webp_server.jpg":                              "image/webp",
		"http://127.0.0.1:3333/2/inside.jpg":                                 "image/webp",
		"http://127.0.0.1:3333/www.weird-path.com/images/webp_server.jpg":    "image/webp",
		"http://127.0.0.1:3333/www.even-more-werid-path.com/webp_server.jpg": "image/webp",
		"http://example.com//images/webp_server.jpg":                         "image/webp",
	}

	var testUrlsLegacy = map[string]string{
		"http://127.0.0.1:3333/webp_server.jpg":     "image/jpeg",
		"http://127.0.0.1:3333/2/inside.jpg":        "image/jpeg",
		"http://example.com/images/webp_server.jpg": "image/jpeg",
	}

	var testUrlsInvalid = map[string]string{
		"http://127.0.0.1:3333/3/does-not-exist.jpg":         "", // Dir mapped does not exist
		"http://127.0.0.1:3333/www.weird-path.com/cover.jpg": "", // Host mapped, final URI invalid
	}

	for url, respType := range testUrls {
		resp, data := requestToServer(url, app, chromeUA, acceptWebP)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, respType, helper.GetContentType(data))
	}

	// tests with Safari
	for url, respType := range testUrlsLegacy {
		resp, data := requestToServer(url, app, safariUA, acceptLegacy)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, respType, helper.GetContentType(data))
	}

	for url, respType := range testUrlsInvalid {
		resp, data := requestToServer(url, app, safariUA, acceptLegacy)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		assert.Equal(t, respType, helper.GetContentType(data))
	}
}

func TestConvertProxyImgMapCWD(t *testing.T) {
	setupParam()
	config.ProxyMode = false
	config.Config.ImgPath = ".." // equivalent to "" when not testing
	config.Config.ImageMap = map[string]string{
		"/1":                     "../pics/dir1",
		"/2":                     "../pics",
		"/3":                     "../pics", // Invalid path, does not exists
		"http://www.example.com": "https://docs.webp.sh",
	}

	var app = fiber.New()
	app.Get("/*", Convert)

	var testUrls = map[string]string{
		"http://127.0.0.1:3333/1/inside.jpg":            "image/webp",
		"http://127.0.0.1:3333/2/webp_server.jpg":       "image/webp",
		"http://127.0.0.1:3333/3/webp_server.jpg":       "image/webp",
		"http://www.example.com/images/webp_server.jpg": "image/webp",
	}

	for url, respType := range testUrls {
		resp, data := requestToServer(url, app, chromeUA, acceptWebP)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, respType, helper.GetContentType(data))
	}
}

func TestConvertBigger(t *testing.T) {
	setupParam()
	config.Config.Quality = 100

	var app = fiber.New()
	app.Get("/*", Convert)

	url := "http://127.0.0.1:3333/big.jpg"
	resp, data := requestToServer(url, app, chromeUA, acceptWebP)
	defer resp.Body.Close()
	assert.Equal(t, "image/jpeg", resp.Header.Get("content-type"))
	assert.Equal(t, "image/jpeg", helper.GetContentType(data))
	_ = os.RemoveAll(config.Config.ExhaustPath)
}
