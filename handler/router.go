package handler

import (
	"net/http"
	"net/url"
	"strings"
	"webp_server_go/config"
	"webp_server_go/encoder"
	"webp_server_go/helper"

	"path"
	"strconv"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

func Convert(c *fiber.Ctx) error {
	// this function need to do:
	// 1. get request path, query string
	// 2. generate rawImagePath, could be local path or remote url(possible with query string)
	// 3. pass it to encoder, get the result, send it back

	var (
		reqURI, _          = url.QueryUnescape(c.Path())        // /mypic/123.jpg
		reqURIwithQuery, _ = url.QueryUnescape(c.OriginalURL()) // /mypic/123.jpg?someother=200&somebugs=200
		filename           = path.Base(reqURI)                  // TODO: could be const? pure filename, 123.jpg
	)

	if !helper.CheckAllowedType(filename) {
		msg := "File extension not allowed! " + filename
		log.Warn(msg)
		c.Status(http.StatusBadRequest)
		_ = c.Send([]byte(msg))
		return nil
	}

	// Sometimes reqURIwithQuery can be https://example.tld/mypic/123.jpg?someother=200&somebugs=200, we need to extract it.
	// delete ../ in reqURI to mitigate directory traversal
	reqURI = path.Clean(reqURI)
	reqURIwithQuery = path.Clean(reqURIwithQuery)

	WidthInt, err := strconv.Atoi(c.Query("width"))
	if err != nil {
		WidthInt = 0
	}
	HeightInt, err := strconv.Atoi(c.Query("height"))
	if err != nil {
		HeightInt = 0
	}
	var extraParams = config.ExtraParams{
		Width:  WidthInt,
		Height: HeightInt,
	}

	var rawImageAbs string
	if config.ProxyMode {
		// this is proxyMode, we'll have to use this url to download and save it to local path, which also gives us rawImageAbs
		// https://test.webp.sh/mypic/123.jpg?someother=200&somebugs=200
		rawImageAbs = fetchRemoteImg(config.Config.ImgPath + reqURIwithQuery)

	} else {
		// not proxyMode, we'll use local path
		rawImageAbs = path.Join(config.Config.ImgPath, reqURI) // /home/xxx/mypic/123.jpg
	}

	goodFormat := helper.GuessSupportedFormat(&c.Request().Header)

	// Check the original image for existence,
	if !helper.ImageExists(rawImageAbs) {
		msg := "image not found"
		_ = c.Send([]byte(msg))
		log.Warn(msg)
		_ = c.SendStatus(404)
		return nil
	}

	// generate with timestamp to make sure files are update-to-date
	// If extraParams not enabled, exhaust path will be /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp
	// If extraParams enabled, and given request at tsuki.jpg?width=200, exhaust path will be /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp_width=200&height=0
	// If extraParams enabled, and given request at tsuki.jpg, exhaust path will be /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp_width=0&height=0
	avifAbs, webpAbs := helper.GenOptimizedAbsPath(rawImageAbs, config.Config.ExhaustPath, filename, reqURI, extraParams)
	encoder.ConvertFilter(rawImageAbs, avifAbs, webpAbs, extraParams, nil)

	var availableFiles = []string{rawImageAbs}
	for _, v := range goodFormat {
		if v == "avif" {
			availableFiles = append(availableFiles, avifAbs)
		}
		if v == "webp" {
			availableFiles = append(availableFiles, webpAbs)
		}
	}

	finalFilename := helper.FindSmallestFiles(availableFiles)
	if strings.HasSuffix(finalFilename, ".webp ") {
		c.Set("Content-Type", "image/webp")
	} else if strings.HasSuffix(finalFilename, ".avif") {
		c.Set("Content-Type", "image/avif")
	}

	c.Set("X-Compression-Rate", helper.GetCompressionRate(rawImageAbs, finalFilename))
	return c.SendFile(finalFilename)
}
