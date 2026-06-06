package handler

import (
	"net/http"
	"net/url"
	"slices"
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

	requestPath := c.Path()
	requestPathDecoded, _ := url.QueryUnescape(requestPath)
	// For invalid or traversal-like paths, always return 404.
	if !strings.HasPrefix(requestPath, "/") || hasTraversalSegments(requestPathDecoded) {
		return sendNotFound(c)
	}

	var (
		err         error
		reqHostname = c.Hostname()
		reqHost     = c.Protocol() + "://" + reqHostname // http://www.example.com:8000
		reqHeader   = &c.Request().Header

		reqURIRaw, _          = url.QueryUnescape(c.Path())        // /mypic/123.jpg
		reqURIwithQueryRaw, _ = url.QueryUnescape(c.OriginalURL()) // /mypic/123.jpg?someother=200&somebugs=200
		reqURI                = path.Clean(reqURIRaw)              // delete ../ in reqURI to mitigate directory traversal
		reqURIwithQuery       = path.Clean(reqURIwithQueryRaw)     // Sometimes reqURIwithQuery can be https://example.tld/mypic/123.jpg?someother=200&somebugs=200, we need to extract it

		filename = path.Base(reqURI)

		meta = c.Query("meta") // Meta request

		width, _     = strconv.Atoi(c.Query("width"))      // Extra Params
		height, _    = strconv.Atoi(c.Query("height"))     // Extra Params
		maxHeight, _ = strconv.Atoi(c.Query("max_height")) // Extra Params
		maxWidth, _  = strconv.Atoi(c.Query("max_width"))  // Extra Params
		extraParams  = config.ExtraParams{
			Width:     width,
			Height:    height,
			MaxWidth:  maxWidth,
			MaxHeight: maxHeight,
		}
	)

	log.Debugf("Incoming connection from %s %s %s", c.IP(), reqHostname, reqURIwithQuery)

	if !helper.CheckAllowedExtension(filename) {
		msg := "File extension not allowed! " + filename
		log.Warn(msg)
		c.Status(http.StatusBadRequest)
		_ = c.SendString(msg)
		return nil
	}

	state := requestState{
		mode:            requestModeLocalDefault,
		reqURI:          reqURI,
		reqURIWithQuery: reqURIwithQuery,
		targetHostName:  config.LocalHostAlias,
		targetHost:      config.Config.ImgPath,
	}
	if isRemoteTarget(config.Config.ImgPath) {
		state.mode = requestModeRemoteDefault
	}
	resolveRequestState(reqHost, reqHostname, &state)

	if state.mode == requestModeRemoteDefault {
		// Don't deal with the encoding to avoid upstream compatibilities
		state.reqURI = c.Path()
		state.reqURIWithQuery = c.OriginalURL()
	}

	if state.isRemote() {
		// Remove first leading slash from reqURIwithQuery if present
		state.reqURIWithQuery = strings.TrimPrefix(state.reqURIWithQuery, "/")
		state.realRemoteAddr = state.targetHost + "/" + state.reqURIWithQuery
	}

	// Check if the file extension is allowed and not with image extension
	// In this case we will serve the file directly
	// Since here we've already sent non-image file, "raw" is not supported by default in the following code
	if config.AllowAllExtensions && !helper.CheckImageExtension(filename) {
		if !state.isRemote() {
			localFilename, err := resolveLocalRequestPath(state)
			if err != nil {
				return sendNotFound(c)
			}
			return c.SendFile(localFilename)
		} else {
			// If the file is not in the ImgPath, we'll have to use the proxy mode to download it
			_ = fetchRemoteImg(state.realRemoteAddr, state.targetHostName)
			localFilename := path.Join(config.Config.RemoteRawPath, state.targetHostName, helper.HashString(state.realRemoteAddr)) + path.Ext(state.realRemoteAddr)
			return c.SendFile(localFilename)
		}
	}

	var rawImageAbs string
	var metadata = config.MetaFile{}
	if state.isRemote() {
		// this is remote mode, we'll have to use this url to download and save it to local path, which also gives us rawImageAbs
		// https://test.webp.sh/mypic/123.jpg?someother=200&somebugs=200

		metadata = fetchRemoteImg(state.realRemoteAddr, state.targetHostName)
		rawImageAbs = path.Join(config.Config.RemoteRawPath, state.targetHostName, metadata.Id) + path.Ext(state.realRemoteAddr)
	} else {
		rawImageAbs, _ = resolveLocalRequestPath(state)
	}

	// Check the original image for existence,
	if rawImageAbs == "" || !helper.ImageExists(rawImageAbs) {
		helper.DeleteMetadata(state.reqURIWithQuery, state.targetHostName)
		return sendNotFound(c)
	}

	if !state.isRemote() {
		metadata, err = helper.ReadMetadata(state.reqURIWithQuery, "", state.targetHostName)
		if err != nil {
			log.Warnf("failed to read metadata for %s: %s", state.reqURIWithQuery, err)
			metadata, err = helper.WriteMetadata(state.reqURIWithQuery, "", state.targetHostName)
			if err != nil {
				log.Warnf("failed to build metadata for %s: %s", state.reqURIWithQuery, err)
			}
		}
		// detect if source file has changed
		if metadata.Checksum != helper.HashFile(rawImageAbs) {
			log.Info("Source file has changed, re-encoding...")
			metadata, err = helper.WriteMetadata(state.reqURIWithQuery, "", state.targetHostName)
			if err != nil {
				log.Warnf("failed to refresh metadata for %s: %s", state.reqURIWithQuery, err)
			}
			cleanProxyCache(path.Join(config.Config.ExhaustPath, state.targetHostName, metadata.Id))
		}
	}

	// If meta request, return the metadata
	if meta == "full" {
		return c.JSON(fiber.Map{
			"height":     metadata.ImageMeta.Height,
			"width":      metadata.ImageMeta.Width,
			"size":       metadata.ImageMeta.Size,
			"format":     metadata.ImageMeta.Format,
			"colorspace": metadata.ImageMeta.Colorspace,
			"num_pages":  metadata.ImageMeta.NumPages,
			"blurhash":   metadata.ImageMeta.Blurhash,
		})
	}

	supportedFormats := helper.GuessSupportedFormat(reqHeader)
	// resize itself and return if only raw(jpg,jpeg,png,gif) is supported
	if supportedFormats["jpg"] == true &&
		supportedFormats["jpeg"] == true &&
		supportedFormats["png"] == true &&
		supportedFormats["gif"] == true &&
		supportedFormats["webp"] == false &&
		supportedFormats["avif"] == false &&
		supportedFormats["jxl"] == false &&
		supportedFormats["heic"] == false {
		dest := path.Join(config.Config.ExhaustPath, state.targetHostName, metadata.Id)
		if !helper.ImageExists(dest) {
			encoder.ResizeItself(rawImageAbs, dest, extraParams)
		}
		return c.SendFile(dest)
	}

	avifAbs, webpAbs, jxlAbs := helper.GenOptimizedAbsPath(metadata, state.targetHostName)
	// Do the convertion based on supported formats and config
	encoder.ConvertFilter(rawImageAbs, jxlAbs, avifAbs, webpAbs, extraParams, supportedFormats, nil)

	var availableFiles = []string{}
	// If source image is in jpg/jpeg/png/gif, we can add it to the available files
	if slices.Contains([]string{"jpg", "jpeg", "png", "gif"}, helper.GetImageExtension(rawImageAbs)) {
		availableFiles = append(availableFiles, rawImageAbs)
	}
	if supportedFormats["avif"] {
		availableFiles = append(availableFiles, avifAbs)
	}
	if supportedFormats["webp"] {
		availableFiles = append(availableFiles, webpAbs)
	}
	if supportedFormats["jxl"] {
		availableFiles = append(availableFiles, jxlAbs)
	}

	finalFilename := helper.FindSmallestFiles(availableFiles)
	contentType := helper.GetFileContentType(finalFilename)
	c.Set("Content-Type", contentType)

	c.Set("X-Compression-Rate", helper.GetCompressionRate(rawImageAbs, finalFilename))
	return c.SendFile(finalFilename)
}
