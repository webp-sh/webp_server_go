package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber"
)

func Convert(ImgPath string, ExhaustPath string, AllowedTypes []string, QUALITY string) func(c *fiber.Ctx) {
	return func(c *fiber.Ctx) {
		//basic vars
		var reqURI = c.Path()                        // mypic/123.jpg
		var RawImageAbs = path.Join(ImgPath, reqURI) // /home/xxx/mypic/123.jpg
		var ImgFilename = path.Base(reqURI)          // pure filename, 123.jpg
		var finalFile string                         // We'll only need one c.sendFile()
		// Check for Safari users. If they're Safari, just simply ignore everything.
		UA := c.Get("User-Agent")
		if strings.Contains(UA, "Safari") && !strings.Contains(UA, "Chrome") &&
			!strings.Contains(UA, "Firefox") {
			c.SendFile(RawImageAbs)
			return
		}

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
			c.Send("File extension not allowed!")
			c.SendStatus(403)
			return
		}

		// Check the original image for existence,
		if !ImageExists(RawImageAbs) {
			c.Send("Image not found!")
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
				fmt.Println(err.Error())
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
			err = WebpEncoder(RawImageAbs, WebpAbsPath, float32(q), verboseMode, nil)

			if err != nil {
				fmt.Println(err)
				c.SendStatus(400)
				c.Send("Bad file!")
				return
			}
			finalFile = WebpAbsPath
		}
		c.SendFile(finalFile)
	}
}
