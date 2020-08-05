package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/gofiber/fiber"
)

func Convert(ImgPath string, ExhaustPath string, AllowedTypes []string, QUALITY string, proxyMode bool) func(c *fiber.Ctx) {
	return func(c *fiber.Ctx) {
		//basic vars
		var reqURI = c.Path()                        // /mypic/123.jpg
		var RawImageAbs = path.Join(ImgPath, reqURI) // /home/xxx/mypic/123.jpg
		var ImgFilename = path.Base(reqURI)          // pure filename, 123.jpg
		var finalFile string                         // We'll only need one c.sendFile()
		var UA = c.Get("User-Agent")
		done := goOrigin(UA)
		if done {
			log.Infof("A Safari/IE/whatever user has arrived...%s", UA)
			// Check for Safari users. If they're Safari, just simply ignore everything.

			etag := GenEtag(RawImageAbs)
			c.Set("ETag", etag)
			c.SendFile(RawImageAbs)
			return
		}
		log.Debugf("Incoming connection from %s@%s with %s", UA, c.IP(), ImgFilename)

		// check ext
		// TODO: may remove this function. Check in Nginx.
		var allowed = false
		for _, ext := range AllowedTypes {
			haystack := strings.ToLower(ImgFilename)
			needle := strings.ToLower("." + ext)
			if strings.HasSuffix(haystack, needle) {
				allowed = true
				break
			} else {
				allowed = false
			}
		}
		if !allowed {
			msg := "File extension not allowed! " + ImgFilename
			log.Warn(msg)
			c.Send(msg)
			if ImageExists(RawImageAbs) {
				etag := GenEtag(RawImageAbs)
				c.Set("ETag", etag)
				c.SendFile(RawImageAbs)
			}
			return
		}

		// Start Proxy Mode
		if proxyMode {
			// https://test.webp.sh/node.png
			realRemoteAddr := ImgPath + reqURI
			// Ping Remote for status code and etag info

			// If status code is 200
			//     Check for local /node.png-etag-<etagValue>
			//         if exist
			//             Send local cache
			//         else
			//             Delete local /node.png*
			//             Fetch and convert to /node.png-etag-<etagValue>
			//             Send local cache
			// else status code is 404
			//      Delete /node.png*
			//      Send 404
			fmt.Println("Remote Addr is " + realRemoteAddr + ", fetching..")
			statusCode, etagValue := GetRemoteImageInfo(realRemoteAddr)
			if statusCode == 200 {
				// Check local path: /node.png-etag-<etagValue>
				localEtagImagePath := ExhaustPath + reqURI + "-etag-" + etagValue
				if ImageExists(localEtagImagePath) {
					c.SendFile(localEtagImagePath)
				} else {
					// Temporary store of remote file.
					// ./remote-raw/node.png
					CleanProxyCache(ExhaustPath + reqURI + "*")
					localRemoteTmpPath := "./remote-raw" + reqURI
					FetchRemoteImage(localRemoteTmpPath, realRemoteAddr)
					q, _ := strconv.ParseFloat(QUALITY, 32)
					_ = os.MkdirAll(path.Dir(localEtagImagePath), 0755)
					err := WebpEncoder(localRemoteTmpPath, localEtagImagePath, float32(q), true, nil)
					if err != nil {
						fmt.Println(err)
					}
					c.SendFile(localEtagImagePath)
				}
			} else {
				msg := "Remote returned + " + strconv.Itoa(statusCode) + " status code!"
				c.Send(msg)
				log.Warn(msg)
				c.SendStatus(statusCode)
				CleanProxyCache(ExhaustPath + reqURI + "*")
				return
			}
			// End Proxy Mode
		} else {
			// Check the original image for existence,
			if !ImageExists(RawImageAbs) {
				msg := "Image not found!"
				c.Send(msg)
				log.Warn(msg)
				c.SendStatus(404)
				return
			}

			_, WebpAbsPath := GenWebpAbs(RawImageAbs, ExhaustPath, ImgFilename, reqURI)

			if ImageExists(WebpAbsPath) {
				finalFile = WebpAbsPath
			} else {
				// we don't have abc.jpg.png1582558990.webp
				// delete the old pic and convert a new one.
				// /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp
				destHalfFile := path.Clean(path.Join(WebpAbsPath, path.Dir(reqURI), ImgFilename))
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
				_ = os.MkdirAll(path.Dir(WebpAbsPath), 0755)
				q, _ := strconv.ParseFloat(QUALITY, 32)
				err = WebpEncoder(RawImageAbs, WebpAbsPath, float32(q), true, nil)

				if err != nil {
					log.Error(err)
					c.SendStatus(400)
					c.Send("Bad file!")
					return
				}
				finalFile = WebpAbsPath
			}
			etag := GenEtag(finalFile)
			c.Set("ETag", etag)
			c.SendFile(finalFile)
		}

	}
}
