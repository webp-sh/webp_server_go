package main

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

func convert(c *fiber.Ctx) error {
	//basic vars
	var reqURI, _ = url.QueryUnescape(c.Path()) // /mypic/123.jpg
	var rawImageAbs string
	if proxyMode {
		rawImageAbs = config.ImgPath + reqURI
	} else {
		rawImageAbs = path.Join(config.ImgPath, reqURI) // /home/xxx/mypic/123.jpg
	}
	var imgFilename = path.Base(reqURI) // pure filename, 123.jpg
	var finalFile string                // We'll only need one c.sendFile()
	var ua = c.Get("User-Agent")
	var accept = c.Get("accept")
	log.Debugf("Incoming connection from %s@%s with %s", ua, c.IP(), imgFilename)

	needOrigin := goOrigin(accept, ua)
	if needOrigin {
		log.Debugf("A Safari/IE/whatever user has arrived...%s", ua)
		c.Set("ETag", genEtag(rawImageAbs))
		c.Set("X-Compression-Rate", NotCompressed)
		if proxyMode {
			localRemoteTmpPath := remoteRaw + reqURI
			_ = fetchRemoteImage(localRemoteTmpPath, rawImageAbs)
			return c.SendFile(localRemoteTmpPath)
		} else {
			return c.SendFile(rawImageAbs)
		}
	}

	// check ext
	var allowed = false
	for _, ext := range config.AllowedTypes {
		haystack := strings.ToLower(imgFilename)
		needle := strings.ToLower("." + ext)
		if strings.HasSuffix(haystack, needle) {
			allowed = true
			break
		} else {
			allowed = false
		}
	}

	if !allowed {
		msg := "File extension not allowed! " + imgFilename
		log.Warn(msg)
		if imageExists(rawImageAbs) {
			c.Set("ETag", genEtag(rawImageAbs))
			return c.SendFile(rawImageAbs)
		} else {
			c.Status(http.StatusBadRequest)
			_ = c.Send([]byte(msg))
			return nil
		}
	}

	// Start Proxy Mode
	if proxyMode {
		// https://test.webp.sh/node.png
		realRemoteAddr := config.ImgPath + reqURI
		// Ping Remote for status code and etag info
		log.Infof("Remote Addr is %s fetching", realRemoteAddr)
		statusCode, etagValue := getRemoteImageInfo(realRemoteAddr)
		if statusCode == 200 {
			// Check local path: /node.png-etag-<etagValue>
			localEtagImagePath := config.ExhaustPath + reqURI + "-etag-" + etagValue
			if imageExists(localEtagImagePath) {
				return c.SendFile(localEtagImagePath)
			} else {
				// Temporary store of remote file.
				cleanProxyCache(config.ExhaustPath + reqURI + "*")
				localRemoteTmpPath := remoteRaw + reqURI
				_ = fetchRemoteImage(localRemoteTmpPath, realRemoteAddr)
				q, _ := strconv.ParseFloat(config.Quality, 32)
				_ = os.MkdirAll(path.Dir(localEtagImagePath), 0755)
				err := webpEncoder(localRemoteTmpPath, localEtagImagePath, float32(q), true, nil)
				if err != nil {
					log.Warning(err)
				}
				return c.SendFile(localEtagImagePath)
			}
		} else {
			msg := fmt.Sprintf("Remote returned %d status code!", statusCode)
			_ = c.Send([]byte(msg))
			log.Warn(msg)
			_ = c.SendStatus(statusCode)
			cleanProxyCache(config.ExhaustPath + reqURI + "*")
			return errors.New(msg)
		}
		// End Proxy Mode
	} else {
		// Check the original image for existence,
		if !imageExists(rawImageAbs) {
			msg := "image not found"
			_ = c.Send([]byte(msg))
			log.Warn(msg)
			_ = c.SendStatus(404)
			return errors.New(msg)
		}

		_, webpAbsPath := genWebpAbs(rawImageAbs, config.ExhaustPath, imgFilename, reqURI)

		if imageExists(webpAbsPath) {
			finalFile = webpAbsPath
		} else {
			// we don't have abc.jpg.png1582558990.webp
			// delete the old pic and convert a new one.
			// /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp
			destHalfFile := path.Clean(path.Join(webpAbsPath, path.Dir(reqURI), imgFilename))
			matches, err := filepath.Glob(destHalfFile + "*")
			if err != nil {
				log.Error(err.Error())
			} else {
				// /home/webp_server/exhaust/path/to/tsuki.jpg.1582558100.webp <- older ones will be removed
				// /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp <- keep the latest one
				for _, p := range matches {
					if strings.Compare(destHalfFile, p) != 0 {
						_ = os.Remove(p)
					}
				}
			}

			//for webp, we need to create dir first
			err = os.MkdirAll(path.Dir(webpAbsPath), 0755)
			q, _ := strconv.ParseFloat(config.Quality, 32)
			err = webpEncoder(rawImageAbs, webpAbsPath, float32(q), true, nil)

			if err != nil {
				log.Error(err)
				_ = c.SendStatus(400)
				_ = c.Send([]byte("Bad file. " + err.Error()))
				return err
			}
			finalFile = webpAbsPath
		}
		etag := genEtag(finalFile)
		c.Set("ETag", etag)
		c.Set("X-Compression-Rate", getCompressionRate(rawImageAbs, webpAbsPath))
		setWebpType(c, rawImageAbs, webpAbsPath)
		return c.SendFile(finalFile)
	}
}

func setWebpType(c *fiber.Ctx, rawImageAbs, webpAbsPath string) {
	raw, _ := ioutil.ReadFile(rawImageAbs)
	webp, _ := ioutil.ReadFile(webpAbsPath)
	if len(webp) > len(raw) {
		c.Set("X-WebP-Type", WebpBigger)
	} else {
		c.Set("X-WebP-Type", WebpSmaller)
	}

}
