// webp_server_go - webp-server_test
// 2020-11-09 11:55
// Benny <benny.think@gmail.com>

package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	chromeUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.183 Safari/537.36"
	SafariUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1 Safari/605.1.15"
)

func TestConvert(t *testing.T) {
	setupParam()
	var testChromeLink = map[string]string{
		"http://127.0.0.1:3333/webp_server.jpg":                 "image/webp",
		"http://127.0.0.1:3333/webp_server.bmp":                 "image/webp",
		"http://127.0.0.1:3333/webp_server.png":                 "image/webp",
		"http://127.0.0.1:3333/empty.jpg":                       "text/plain; charset=utf-8",
		"http://127.0.0.1:3333/png.jpg":                         "image/webp",
		"http://127.0.0.1:3333/12314.jpg":                       "text/plain; charset=utf-8",
		"http://127.0.0.1:3333/dir1/inside.jpg":                 "image/webp",
		"http://127.0.0.1:3333/%e5%a4%aa%e7%a5%9e%e5%95%a6.png": "image/webp",
		"http://127.0.0.1:3333/太神啦.png":                         "image/webp",
	}

	var testSafariLink = map[string]string{
		"http://127.0.0.1:3333/webp_server.jpg": "image/jpeg",
		"http://127.0.0.1:3333/webp_server.bmp": "image/bmp",
		"http://127.0.0.1:3333/webp_server.png": "image/png",
		"http://127.0.0.1:3333/empty.jpg":       "text/plain; charset=utf-8",
		"http://127.0.0.1:3333/png.jpg":         "image/png",
		"http://127.0.0.1:3333/12314.jpg":       "text/plain; charset=utf-8",
		"http://127.0.0.1:3333/dir1/inside.jpg": "image/jpeg",
	}

	var app = fiber.New()
	app.Get("/*", convert)

	// test Chrome
	for url, respType := range testChromeLink {
		_, data := requestToServer(url, app, chromeUA)
		contentType := getFileContentType(data)
		assert.Equal(t, respType, contentType)
	}

	// test Safari
	for url, respType := range testSafariLink {
		_, data := requestToServer(url, app, SafariUA)
		contentType := getFileContentType(data)
		assert.Equal(t, respType, contentType)
	}

}

func TestConvertNotAllowed(t *testing.T) {
	setupParam()
	config.AllowedTypes = []string{"jpg", "png", "jpeg"}

	var app = fiber.New()
	app.Get("/*", convert)

	// not allowed, but we have the file
	url := "http://127.0.0.1:3333/webp_server.bmp"
	_, data := requestToServer(url, app, chromeUA)
	contentType := getFileContentType(data)
	assert.Equal(t, "image/bmp", contentType)

	// not allowed, random file
	url = url + "hagdgd"
	_, data = requestToServer(url, app, chromeUA)
	assert.Contains(t, string(data), "File extension not allowed")

}

func TestConvertProxyModeBad(t *testing.T) {
	setupParam()
	proxyMode = true

	var app = fiber.New()
	app.Get("/*", convert)

	// this is local image, should be 500
	url := "http://127.0.0.1:3333/webp_server.bmp"
	resp, _ := requestToServer(url, app, chromeUA)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

}

func TestConvertProxyModeWork(t *testing.T) {
	setupParam()
	proxyMode = true

	var app = fiber.New()
	app.Get("/*", convert)

	config.ImgPath = "https://webp.sh"
	url := "https://webp.sh/images/cover.jpg"

	resp, data := requestToServer(url, app, chromeUA)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "image/webp", getFileContentType(data))

}

func setupParam() {
	// setup parameters here...
	config.ImgPath = "./pics"
	config.ExhaustPath = "./exhaust"
	config.AllowedTypes = []string{"jpg", "png", "jpeg", "bmp"}
}

func requestToServer(url string, app *fiber.App, ua string) (*http.Response, []byte) {
	req := httptest.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", ua)
	resp, _ := app.Test(req, 60000)
	data, _ := ioutil.ReadAll(resp.Body)
	return resp, data
}
