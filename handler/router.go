package handler

import (
	"net/http"
	"net/url"
	"regexp"
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
		reqHostname        = c.Hostname()
		reqHost            = c.Protocol() + "://" + reqHostname // http://www.example.com:8000
		reqURI, _          = url.QueryUnescape(c.Path())        // /mypic/123.jpg
		reqURIwithQuery, _ = url.QueryUnescape(c.OriginalURL()) // /mypic/123.jpg?someother=200&somebugs=200
		filename           = path.Base(reqURI)
		realRemoteAddr     = ""
		targetHostName     = config.LocalHostAlias
		targetHost         = config.Config.ImgPath
		proxyMode          = config.ProxyMode
		mapMode            = false
	)

	log.Debugf("Incoming connection from %s %s %s", c.IP(), reqHostname, reqURIwithQuery)

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

	width, _ := strconv.Atoi(c.Query("width"))
	height, _ := strconv.Atoi(c.Query("height"))

	var extraParams = config.ExtraParams{
		Width:  width,
		Height: height,
	}

	// Rewrite the target backend if a mapping rule matches the hostname
	if hostMap, hostMapFound := config.Config.ImageMap[reqHost]; hostMapFound {
		log.Debugf("Found host mapping %s -> %s", reqHostname, hostMap)
		targetHostUrl, _ := url.Parse(hostMap)
		targetHostName = targetHostUrl.Host
		targetHost = targetHostUrl.Scheme + "://" + targetHostUrl.Host
		proxyMode = true
	} else {
		// There's not matching host mapping, now check for any URI map that apply
		httpRegexpMatcher := regexp.MustCompile(config.HttpRegexp)
		for uriMap, uriMapTarget := range config.Config.ImageMap {
			if strings.HasPrefix(reqURI, uriMap) {
				log.Debugf("Found URI mapping %s -> %s", uriMap, uriMapTarget)
				mapMode = true

				// if uriMapTarget we use the proxy mode to fetch the remote
				if httpRegexpMatcher.Match([]byte(uriMapTarget)) {
					targetHostUrl, _ := url.Parse(uriMapTarget)
					targetHostName = targetHostUrl.Host
					targetHost = targetHostUrl.Scheme + "://" + targetHostUrl.Host
					reqURI          = strings.Replace(reqURI,          uriMap, targetHostUrl.Path, 1)
					reqURIwithQuery = strings.Replace(reqURIwithQuery, uriMap, targetHostUrl.Path, 1)
					proxyMode = true
				} else {
					reqURI          = strings.Replace(reqURI,          uriMap, uriMapTarget, 1)
					reqURIwithQuery = strings.Replace(reqURIwithQuery, uriMap, uriMapTarget, 1)
				}
				break
			}
		}

	}

	if proxyMode {

		if !mapMode {
			// Don't deal with the encoding to avoid upstream compatibilities
			reqURI = c.Path()
			reqURIwithQuery = c.OriginalURL()
		}

		log.Tracef("reqURIwithQuery is %s", reqURIwithQuery)

		// Replace host in the URL
		// realRemoteAddr = strings.Replace(reqURIwithQuery, reqHost, targetHost, 1)
		realRemoteAddr = targetHost + reqURIwithQuery
		log.Debugf("realRemoteAddr is %s", realRemoteAddr)
	}
	
	var rawImageAbs string
	var metadata = config.MetaFile{}
	if proxyMode {
		// this is proxyMode, we'll have to use this url to download and save it to local path, which also gives us rawImageAbs
		// https://test.webp.sh/mypic/123.jpg?someother=200&somebugs=200
		
		metadata = fetchRemoteImg(realRemoteAddr, targetHostName)
		rawImageAbs = path.Join(config.RemoteRaw, targetHostName,  metadata.Id)
	} else {
		// not proxyMode, we'll use local path
		metadata = helper.ReadMetadata(reqURIwithQuery, "", targetHostName)
		rawImageAbs = path.Join(config.Config.ImgPath, reqURI)
		// detect if source file has changed
		if metadata.Checksum != helper.HashFile(rawImageAbs) {
			log.Info("Source file has changed, re-encoding...")
			helper.WriteMetadata(reqURIwithQuery, "", targetHostName)
			cleanProxyCache(path.Join(config.Config.ExhaustPath, targetHostName, metadata.Id))
		}
	}

	goodFormat := helper.GuessSupportedFormat(&c.Request().Header)
	// resize itself and return if only one format(raw) is supported
	if len(goodFormat) == 1 {
		dest := path.Join(config.Config.ExhaustPath, targetHostName, metadata.Id)
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

	avifAbs, webpAbs := helper.GenOptimizedAbsPath(metadata, targetHostName)
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
