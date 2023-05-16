package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

var (
	chromeUA     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.183 Safari/537.36"
	acceptWebP   = "image/webp,image/apng,image/*,*/*;q=0.8"
	acceptAvif   = "image/avif,image/*,*/*;q=0.8"
	acceptLegacy = "image/jpeg"
	safariUA     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1 Safari/605.1.15"
)

func setupParam() {
	// setup parameters here...
	config.ImgPath = "./pics"
	config.ExhaustPath = "./exhaust_test"
	config.AllowedTypes = []string{"jpg", "png", "jpeg", "bmp"}

	proxyMode = false
	remoteRaw = "remote-raw"
}

func requestToServer(url string, app *fiber.App, ua, accept string) (*http.Response, []byte) {
	req := httptest.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", accept)
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
	app.Get("/*", convert)
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
		"http://127.0.0.1:3333/太神啦.png":                       "image/webp",
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
		"http://127.0.0.1:3333/太神啦.png":                       "image/avif",
	}

	var testSafariLink = map[string]string{
		"http://127.0.0.1:3333/webp_server.jpg": "image/jpeg",
		"http://127.0.0.1:3333/webp_server.bmp": "image/bmp",
		"http://127.0.0.1:3333/webp_server.png": "image/png",
		"http://127.0.0.1:3333/empty.jpg":       "",
		"http://127.0.0.1:3333/png.jpg":         "image/png",
		"http://127.0.0.1:3333/12314.jpg":       "",
		"http://127.0.0.1:3333/dir1/inside.jpg": "image/jpeg",
	}

	var app = fiber.New()
	app.Get("/*", convert)

	// test Chrome
	for url, respType := range testChromeLink {
		resp, data := requestToServer(url, app, chromeUA, acceptWebP)
		defer resp.Body.Close()
		contentType := getFileContentType(data)
		assert.Equal(t, respType, contentType)
	}

	// test Safari
	for url, respType := range testSafariLink {
		resp, data := requestToServer(url, app, safariUA, acceptLegacy)
		defer resp.Body.Close()
		contentType := getFileContentType(data)
		assert.Equal(t, respType, contentType)
	}

	// test Avif is processed in proxy mode
	config.EnableAVIF = true
	for url, respType := range testChromeAvifLink {
		resp, data := requestToServer(url, app, chromeUA, acceptAvif)
		defer resp.Body.Close()
		contentType := getFileContentType(data)
		assert.NotNil(t, respType)
		assert.Equal(t, respType, contentType)
	}
}

func TestConvertNotAllowed(t *testing.T) {
	setupParam()
	config.AllowedTypes = []string{"jpg", "png", "jpeg"}

	var app = fiber.New()
	app.Get("/*", convert)

	// not allowed, but we have the file, this should return File extension not allowed
	url := "http://127.0.0.1:3333/webp_server.bmp"
	resp, data := requestToServer(url, app, chromeUA, acceptWebP)
	defer resp.Body.Close()
	assert.Contains(t, string(data), "File extension not allowed")

	// not allowed, random file
	url = url + "hagdgd"
	resp, data = requestToServer(url, app, chromeUA, acceptWebP)
	defer resp.Body.Close()
	assert.Contains(t, string(data), "File extension not allowed")

}

func TestConvertProxyModeBad(t *testing.T) {
	setupParam()
	proxyMode = true

	var app = fiber.New()
	app.Get("/*", convert)

	// this is local random image, should be 404
	url := "http://127.0.0.1:3333/webp_8888server.bmp"
	resp, _ := requestToServer(url, app, chromeUA, acceptWebP)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

}

func TestConvertProxyModeWork(t *testing.T) {
	setupParam()
	proxyMode = true

	var app = fiber.New()
	app.Get("/*", convert)

	config.ImgPath = "https://webp.sh"
	url := "https://webp.sh/images/cover.jpg"

	resp, data := requestToServer(url, app, chromeUA, acceptWebP)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "image/webp", getFileContentType(data))

	// test proxyMode with Safari
	resp, data = requestToServer(url, app, safariUA, acceptLegacy)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "image/jpeg", getFileContentType(data))
}

func TestConvertBigger(t *testing.T) {
	setupParam()
	config.Quality = 100

	var app = fiber.New()
	app.Get("/*", convert)

	url := "http://127.0.0.1:3333/big.jpg"
	resp, data := requestToServer(url, app, chromeUA, acceptWebP)
	defer resp.Body.Close()
	assert.Equal(t, "image/jpeg", resp.Header.Get("content-type"))
	assert.Equal(t, "image/jpeg", getFileContentType(data))
	_ = os.RemoveAll(config.ExhaustPath)
}
