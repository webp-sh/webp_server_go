// webp_server_go - webp-server_test
// 2020-11-09 11:55
// Benny <benny.think@gmail.com>

package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http/httptest"
	"testing"
)

func TestConvert(t *testing.T) {

	var testChromeLink = map[string]string{
		"http://127.0.0.1:3333/webp_server.jpg": "image/webp",
		"http://127.0.0.1:3333/webp_server.bmp": "image/webp",
		"http://127.0.0.1:3333/webp_server.png": "image/webp",
		"http://127.0.0.1:3333/empty.jpg":       "text/plain; charset=utf-8",
		"http://127.0.0.1:3333/png.jpg":         "image/webp",
		"http://127.0.0.1:3333/12314.jpg":       "text/plain; charset=utf-8",
		"http://127.0.0.1:3333/dir1/inside.jpg": "image/webp",
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

	// setup parameters here...
	config.ImgPath = "./pics"
	config.ExhaustPath = "./exhaust"
	config.AllowedTypes = []string{"jpg", "png", "jpeg", "bmp"}

	var app = fiber.New()
	app.Get("/*", convert)

	// test Chrome
	for url, respType := range testChromeLink {
		req := httptest.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.183 Safari/537.36")
		resp, _ := app.Test(req)
		data, _ := ioutil.ReadAll(resp.Body)
		contentType := getFileContentType(data)
		assert.Equal(t, respType, contentType)
	}

	// test Safari
	for url, respType := range testSafariLink {
		req := httptest.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1 Safari/605.1.15")
		resp, _ := app.Test(req)
		data, _ := ioutil.ReadAll(resp.Body)
		contentType := getFileContentType(data)
		assert.Equal(t, respType, contentType)
	}

}
