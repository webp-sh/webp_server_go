package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

func convert(c *fiber.Ctx) error {
	//basic vars
	var reqURI, _ = url.QueryUnescape(c.Path()) // /mypic/123.jpg

	// delete ../ in reqURI to mitigate directory traversal
	reqURI = path.Clean(reqURI)

	var rawImageAbs string
	if proxyMode {
		rawImageAbs = config.ImgPath + reqURI
	} else {
		rawImageAbs = path.Join(config.ImgPath, reqURI) // /home/xxx/mypic/123.jpg
	}
	var imgFilename = path.Base(reqURI) // pure filename, 123.jpg
	log.Debugf("Incoming connection from %s %s", c.IP(), imgFilename)

	if !checkAllowedType(imgFilename) {
		msg := "File extension not allowed! " + imgFilename
		log.Warn(msg)
		c.Status(http.StatusBadRequest)
		_ = c.Send([]byte(msg))
		return nil
	}

	goodFormat := guessSupportedFormat(&c.Request().Header)

	// old browser only, send the original image or fetch from remote and send.
	if len(goodFormat) == 1 {
		c.Set("ETag", genEtag(rawImageAbs))
		if proxyMode {
			localRemoteTmpPath := remoteRaw + reqURI
			_ = fetchRemoteImage(localRemoteTmpPath, rawImageAbs)
			return c.SendFile(localRemoteTmpPath)
		} else {
			return c.SendFile(rawImageAbs)
		}
	}

	if proxyMode {
		return proxyHandler(c, reqURI)
	}

	// Check the original image for existence,
	if !imageExists(rawImageAbs) {
		msg := "image not found"
		_ = c.Send([]byte(msg))
		log.Warn(msg)
		_ = c.SendStatus(404)
		return nil
	}

	// generate with timestamp to make sure files are update-to-date
	avifAbs, webpAbs := genOptimizedAbsPath(rawImageAbs, config.ExhaustPath, imgFilename, reqURI)
	convertFilter(rawImageAbs, avifAbs, webpAbs, nil)

	var availableFiles = []string{rawImageAbs}
	for _, v := range goodFormat {
		if "avif" == v {
			availableFiles = append(availableFiles, avifAbs)
		}
		if "webp" == v {
			availableFiles = append(availableFiles, webpAbs)
		}
	}

	var finalFileName = findSmallestFiles(availableFiles)
	var finalFileExtension = path.Ext(finalFileName)
	if finalFileExtension == ".webp" {
		c.Set("Content-Type", "image/webp")
	} else if finalFileExtension == ".avif" {
		c.Set("Content-Type", "image/avif")
	}

	etag := genEtag(finalFileName)
	c.Set("ETag", etag)
	c.Set("X-Compression-Rate", getCompressionRate(rawImageAbs, finalFileName))
	return c.SendFile(finalFileName)
}

func proxyHandler(c *fiber.Ctx, reqURI string) error {
	// https://test.webp.sh/node.png
	realRemoteAddr := config.ImgPath + reqURI
	// Ping Remote for status code and etag info
	log.Infof("Remote Addr is %s fetching", realRemoteAddr)
	statusCode, etagValue, remoteLength := getRemoteImageInfo(realRemoteAddr)
	if statusCode == 200 {
		// Check local path: /node.png-etag-<etagValue>
		localEtagWebPPath := config.ExhaustPath + reqURI + "-etag-" + etagValue
		if imageExists(localEtagWebPPath) {
			chooseProxy(remoteLength, localEtagWebPPath)
			return c.SendFile(localEtagWebPPath)
		} else {
			// Temporary store of remote file.
			cleanProxyCache(config.ExhaustPath + reqURI + "*")
			localRawImagePath := remoteRaw + reqURI
			_ = fetchRemoteImage(localRawImagePath, realRemoteAddr)
			_ = os.MkdirAll(path.Dir(localEtagWebPPath), 0755)
			encodeErr := webpEncoder(localRawImagePath, localEtagWebPPath, config.Quality)
			if encodeErr != nil {
				// Send as it is.
				return c.SendFile(localRawImagePath)
			}
			chooseProxy(remoteLength, localEtagWebPPath)
			return c.SendFile(localEtagWebPPath)
		}
	} else {
		msg := fmt.Sprintf("Remote returned %d status code!", statusCode)
		_ = c.Send([]byte(msg))
		log.Warn(msg)
		_ = c.SendStatus(statusCode)
		cleanProxyCache(config.ExhaustPath + reqURI + "*")
		return errors.New(msg)
	}
}
