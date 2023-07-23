package handler

import (
	"net/http"
	"net/url"
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
		filename           = path.Base(reqURI)
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

	width, err := strconv.Atoi(c.Query("w"))
	if err != nil {
		width = 0
	}
	height, err := strconv.Atoi(c.Query("h"))
	if err != nil {
		height = 0
	}
	var extraParams = config.ExtraParams{
		Width:  width,
		Height: height,
	}

	var rawImageAbs string
	var metadata = config.MetaFile{}
	if config.ProxyMode {
		// this is proxyMode, we'll have to use this url to download and save it to local path, which also gives us rawImageAbs
		// https://test.webp.sh/mypic/123.jpg?someother=200&somebugs=200
		metadata = fetchRemoteImg(config.Config.ImgPath + reqURIwithQuery)
		rawImageAbs = path.Join(config.RemoteRaw, metadata.Id)
	} else {
		// not proxyMode, we'll use local path
		metadata = helper.ReadMetadata(reqURIwithQuery, "")
		rawImageAbs = path.Join(config.Config.ImgPath, reqURI)
		// detect if source file has changed
		if metadata.Checksum != helper.HashFile(rawImageAbs) {
			log.Info("Source file has changed, re-encoding...")
			helper.WriteMetadata(reqURIwithQuery, "")
			cleanProxyCache(path.Join(config.Config.ExhaustPath, metadata.Id))
		}
	}

	goodFormat := helper.GuessSupportedFormat(&c.Request().Header)
	// resize itself and return if only one format(raw) is supported
	if len(goodFormat) == 1 {
		dest := path.Join(config.Config.ExhaustPath, metadata.Id)
		if !helper.ImageExists(dest) {
			encoder.ResizeItself(rawImageAbs, dest, extraParams)
		}
		return c.SendFile(dest)
	}

	// Check the original image for existence,
	if !helper.ImageExists(rawImageAbs) {
		msg := "image not found"
		_ = c.Send([]byte(msg))
		log.Warn(msg)
		_ = c.SendStatus(404)
		return nil
	}

	avifAbs, webpAbs := helper.GenOptimizedAbsPath(metadata)
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
	contentType := helper.GetFileContentType(finalFilename)
	c.Set("Content-Type", contentType)

	c.Set("X-Compression-Rate", helper.GetCompressionRate(rawImageAbs, finalFilename))
	return c.SendFile(finalFilename)
}
