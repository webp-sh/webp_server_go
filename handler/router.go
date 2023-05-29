package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"webp_server_go/config"
	"webp_server_go/encoder"
	"webp_server_go/helper"

	"path"
	"strconv"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

func Convert(c *fiber.Ctx) error {
	//basic vars
	var (
		reqURI, _          = url.QueryUnescape(c.Path())        // /mypic/123.jpg
		reqURIwithQuery, _ = url.QueryUnescape(c.OriginalURL()) // /mypic/123.jpg?someother=200&somebugs=200
		imgFilename        = path.Base(reqURI)                  // pure filename, 123.jpg
	)
	// Sometimes reqURIwithQuery can be https://example.tld/mypic/123.jpg?someother=200&somebugs=200, we need to extract it.
	u, err := url.Parse(reqURIwithQuery)
	if err != nil {
		log.Errorln(err)
	}
	reqURIwithQuery = u.RequestURI()
	// delete ../ in reqURI to mitigate directory traversal
	reqURI = path.Clean(reqURI)
	reqURIwithQuery = path.Clean(reqURIwithQuery)

	// Begin Extra params
	var extraParams config.ExtraParams
	Width := c.Query("width")
	Height := c.Query("height")
	WidthInt, err := strconv.Atoi(Width)
	if err != nil {
		WidthInt = 0
	}
	HeightInt, err := strconv.Atoi(Height)
	if err != nil {
		HeightInt = 0
	}
	extraParams = config.ExtraParams{
		Width:  WidthInt,
		Height: HeightInt,
	}
	// End Extra params

	var rawImageAbs string
	if config.ProxyMode {
		rawImageAbs = config.Config.ImgPath + reqURIwithQuery // https://test.webp.sh/mypic/123.jpg?someother=200&somebugs=200
	} else {
		rawImageAbs = path.Join(config.Config.ImgPath, reqURI) // /home/xxx/mypic/123.jpg
	}

	if !helper.CheckAllowedType(imgFilename) {
		msg := "File extension not allowed! " + imgFilename
		log.Warn(msg)
		c.Status(http.StatusBadRequest)
		_ = c.Send([]byte(msg))
		return nil
	}

	goodFormat := helper.GuessSupportedFormat(&c.Request().Header)

	if config.ProxyMode {
		rawImageAbs, _ = proxyHandler(c, reqURIwithQuery)
	}

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
	avifAbs, webpAbs := helper.GenOptimizedAbsPath(rawImageAbs, config.Config.ExhaustPath, imgFilename, reqURI, extraParams)
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

	var finalFileName = helper.FindSmallestFiles(availableFiles)
	var finalFileExtension = path.Ext(finalFileName)
	if finalFileExtension == ".webp" {
		c.Set("Content-Type", "image/webp")
	} else if finalFileExtension == ".avif" {
		c.Set("Content-Type", "image/avif")
	}

	c.Set("X-Compression-Rate", helper.GetCompressionRate(rawImageAbs, finalFileName))
	return c.SendFile(finalFileName)
}

func proxyHandler(c *fiber.Ctx, reqURIwithQuery string) (string, error) {
	// https://test.webp.sh/mypic/123.jpg?someother=200&somebugs=200
	realRemoteAddr := config.Config.ImgPath + reqURIwithQuery

	// Ping Remote for status code and etag info
	log.Infof("Remote Addr is %s, fetching info...", realRemoteAddr)
	statusCode, etagValue, _ := helper.GetRemoteImageInfo(realRemoteAddr)

	// Since we cannot store file in format of "/mypic/123.jpg?someother=200&somebugs=200", we need to hash it.
	reqURIwithQueryHash := helper.Sha1Path(reqURIwithQuery) // 378e740ca56144b7587f3af9debeee544842879a
	etagValueHash := helper.Sha1Path(etagValue)             // 123e740ca56333b7587f3af9debeee5448428123

	localRawImagePath := path.Join(config.RemoteRaw, reqURIwithQueryHash+"-etag-"+etagValueHash) // For store the remote raw image, /home/webp_server/remote-raw/378e740ca56144b7587f3af9debeee544842879a-etag-123e740ca56333b7587f3af9debeee5448428123

	if statusCode == 200 {
		if helper.ImageExists(localRawImagePath) {
			return localRawImagePath, nil
		} else {
			// Temporary store of remote file.
			helper.CleanProxyCache(config.Config.ExhaustPath + reqURIwithQuery + "*")
			log.Info("Remote file not found in remote-raw path, fetching...")
			err := helper.FetchRemoteImage(localRawImagePath, realRemoteAddr)
			return localRawImagePath, err
		}
	} else {
		msg := fmt.Sprintf("Remote returned %d status code!", statusCode)
		_ = c.Send([]byte(msg))
		log.Warn(msg)
		_ = c.SendStatus(statusCode)
		helper.CleanProxyCache(config.Config.ExhaustPath + reqURIwithQuery + "*")
		return "", errors.New(msg)
	}
}

func switchProxyMode() {
	// Check for remote address
	matched, _ := regexp.MatchString(`^https?://`, config.Config.ImgPath)
	config.ProxyMode = false
	if matched {
		config.ProxyMode = true
	} else {
		_, err := os.Stat(config.Config.ImgPath)
		if err != nil {
			log.Fatalf("Your image path %s is incorrect.Please check and confirm.", config.Config.ImgPath)
		}
	}
}
