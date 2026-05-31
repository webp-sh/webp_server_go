package handler

import (
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"webp_server_go/config"
	"webp_server_go/encoder"
	"webp_server_go/helper"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

func rejectInvalidPath(c *fiber.Ctx) error {
	msg := "Invalid path"
	log.Warn(msg)
	c.Status(http.StatusBadRequest)
	return c.SendString(msg)
}

func Convert(c *fiber.Ctx) error {
	if !strings.HasPrefix(c.Path(), "/") {
		_ = c.SendStatus(http.StatusBadRequest)
		return nil
	}

	var (
		reqHostname = c.Hostname()
		reqHost     = c.Protocol() + "://" + reqHostname
		reqHeader   = &c.Request().Header

		realRemoteAddr = ""
		targetHostName = config.LocalHostAlias
		targetHost     = config.Config.ImgPath
		proxyMode      = config.ProxyMode
		mapMode        = false
		matchedURIMap  = ""
		matchedMapBase = ""

		meta = c.Query("meta")

		width, _     = strconv.Atoi(c.Query("width"))
		height, _    = strconv.Atoi(c.Query("height"))
		maxHeight, _ = strconv.Atoi(c.Query("max_height"))
		maxWidth, _  = strconv.Atoi(c.Query("max_width"))
		queryKey     = helper.BuildQueryKey(c.Query("width"), c.Query("height"), c.Query("max_width"), c.Query("max_height"))
		extraParams  = config.ExtraParams{
			Width:     width,
			Height:    height,
			MaxWidth:  maxWidth,
			MaxHeight: maxHeight,
		}
	)

	reqURI, err := helper.DecodeRequestPath(c.Path())
	if err != nil {
		return rejectInvalidPath(c)
	}

	filename := filepath.Base(reqURI)

	log.Debugf("Incoming connection from %s %s %s", c.IP(), reqHostname, c.OriginalURL())

	if !helper.CheckAllowedExtension(filename) {
		msg := "File extension not allowed! " + filename
		log.Warn(msg)
		c.Status(http.StatusBadRequest)
		_ = c.SendString(msg)
		return nil
	}

	if hostMap, hostMapFound := config.Config.ImageMap[reqHost]; hostMapFound {
		log.Debugf("Found host mapping %s -> %s", reqHostname, hostMap)
		targetHostUrl, _ := url.Parse(hostMap)
		targetHostName = targetHostUrl.Host
		targetHost = targetHostUrl.Scheme + "://" + targetHostUrl.Host
		proxyMode = true
	} else {
		httpRegexpMatcher := regexp.MustCompile(config.HttpRegexp)
		for uriMap, uriMapTarget := range config.Config.ImageMap {
			if strings.HasPrefix(reqURI, uriMap) {
				log.Debugf("Found URI mapping %s -> %s", uriMap, uriMapTarget)
				mapMode = true
				matchedURIMap = uriMap
				matchedMapBase = uriMapTarget

				if httpRegexpMatcher.Match([]byte(uriMapTarget)) {
					targetHostUrl, _ := url.Parse(uriMapTarget)
					targetHostName = targetHostUrl.Host
					targetHost = targetHostUrl.Scheme + "://" + targetHostUrl.Host
					reqURI = strings.Replace(reqURI, uriMap, targetHostUrl.Path, 1)
					proxyMode = true
				}
				break
			}
		}
	}

	var reqURIwithQuery string
	if proxyMode {
		if mapMode {
			reqURIwithQueryRaw, _ := url.QueryUnescape(c.OriginalURL())
			reqURIwithQuery = path.Clean(reqURIwithQueryRaw)
			if matchedURIMap != "" {
				target := config.Config.ImageMap[matchedURIMap]
				if matched, _ := regexp.MatchString(config.HttpRegexp, target); matched {
					targetHostUrl, _ := url.Parse(target)
					reqURIwithQuery = strings.Replace(reqURIwithQuery, matchedURIMap, targetHostUrl.Path, 1)
				}
			}
		} else {
			reqURI = c.Path()
			reqURIwithQuery = c.OriginalURL()
		}

		reqURIwithQuery = strings.TrimPrefix(reqURIwithQuery, "/")
		realRemoteAddr = targetHost + "/" + reqURIwithQuery
	}

	resolveLocalFile := func() (absPath string, relPath string, err error) {
		if mapMode && matchedMapBase != "" && !proxyMode {
			mappedAbs, absErr := filepath.Abs(matchedMapBase)
			if absErr != nil {
				return "", "", helper.ErrPathTraversal
			}
			suffix := strings.TrimPrefix(reqURI, matchedURIMap)
			return helper.ResolveUnderBase(mappedAbs, suffix)
		}
		return helper.ResolveUnderBase(config.Config.ImgPath, c.Path())
	}

	if config.AllowAllExtensions && !helper.CheckImageExtension(filename) {
		if !proxyMode {
			rawImageAbs, _, resolveErr := resolveLocalFile()
			if resolveErr != nil {
				return rejectInvalidPath(c)
			}
			return c.SendFile(rawImageAbs)
		}

		_ = fetchRemoteImg(realRemoteAddr, targetHostName)
		localFilename := path.Join(config.Config.RemoteRawPath, targetHostName, helper.HashString(realRemoteAddr)) + path.Ext(realRemoteAddr)
		return c.SendFile(localFilename)
	}

	var rawImageAbs string
	var relPath string
	var metadata = config.MetaFile{}
	if proxyMode {
		metadata = fetchRemoteImg(realRemoteAddr, targetHostName)
		rawImageAbs = path.Join(config.Config.RemoteRawPath, targetHostName, metadata.Id) + path.Ext(realRemoteAddr)
	} else {
		var resolveErr error
		rawImageAbs, relPath, resolveErr = resolveLocalFile()
		if resolveErr != nil {
			return rejectInvalidPath(c)
		}

		localTarget := helper.MetadataTarget{
			LocalRelPath:  relPath,
			LocalQueryKey: queryKey,
			LocalAbsPath:  rawImageAbs,
		}
		metadata = helper.ReadMetadataForTarget(localTarget, "", targetHostName)
		if metadata.Checksum != helper.HashFile(rawImageAbs) {
			log.Info("Source file has changed, re-encoding...")
			helper.WriteMetadataForTarget(localTarget, "", targetHostName)
			cleanProxyCache(path.Join(config.Config.ExhaustPath, targetHostName, metadata.Id))
		}
	}

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
	if supportedFormats["jpg"] == true &&
		supportedFormats["jpeg"] == true &&
		supportedFormats["png"] == true &&
		supportedFormats["gif"] == true &&
		supportedFormats["webp"] == false &&
		supportedFormats["avif"] == false &&
		supportedFormats["jxl"] == false &&
		supportedFormats["heic"] == false {
		dest := path.Join(config.Config.ExhaustPath, targetHostName, metadata.Id)
		if !helper.ImageExists(dest) {
			encoder.ResizeItself(rawImageAbs, dest, extraParams)
		}
		return c.SendFile(dest)
	}

	if !helper.ImageExists(rawImageAbs) {
		if !proxyMode {
			helper.DeleteMetadataForTarget(helper.MetadataTarget{
				LocalRelPath:  relPath,
				LocalQueryKey: queryKey,
			}, targetHostName)
		} else {
			helper.DeleteMetadataForTarget(helper.MetadataTarget{
				RemoteURL: realRemoteAddr,
			}, targetHostName)
		}
		msg := "Image not found!"
		_ = c.Send([]byte(msg))
		log.Warn(msg)
		_ = c.SendStatus(404)
		return nil
	}

	avifAbs, webpAbs, jxlAbs := helper.GenOptimizedAbsPath(metadata, targetHostName)
	encoder.ConvertFilter(rawImageAbs, jxlAbs, avifAbs, webpAbs, extraParams, supportedFormats, nil)

	var availableFiles = []string{}
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
