package main

import (
	"bytes"
	"crypto/sha1" //#nosec
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/h2non/filetype"

	"github.com/valyala/fasthttp"

	"strings"

	log "github.com/sirupsen/logrus"
)

func avifMatcher(buf []byte) bool {
	// 0000001c 66747970 61766966 00000000 61766966 6d696631 6d696166
	return len(buf) > 1 && bytes.Equal(buf[:28], []byte{
		0x0, 0x0, 0x0, 0x1c,
		0x66, 0x74, 0x79, 0x70,
		0x61, 0x76, 0x69, 0x66,
		0x0, 0x0, 0x0, 0x0,
		0x61, 0x76, 0x69, 0x66,
		0x6d, 0x69, 0x66, 0x31,
		0x6d, 0x69, 0x61, 0x66,
	})
}
func getFileContentType(buffer []byte) string {
	// TODO deprecated.
	var avifType = filetype.NewType("avif", "image/avif")
	filetype.AddMatcher(avifType, avifMatcher)
	kind, _ := filetype.Match(buffer)
	return kind.MIME.Value
}

func fileCount(dir string) int64 {
	var count int64 = 0
	_ = filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				count += 1
			}
			return nil
		})
	return count
}

func imageExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	if info.Size() < 100 {
		// means something wrong in exhaust file system
		return false
	}
	log.Debugf("file %s exists!", filename)
	return !info.IsDir()
}

func checkAllowedType(imgFilename string) bool {
	imgFilename = strings.ToLower(imgFilename)
	for _, allowedType := range config.AllowedTypes {
		if allowedType == "*" {
			return true
		}
		allowedType = "." + strings.ToLower(allowedType)
		if strings.HasSuffix(imgFilename, allowedType) {
			return true
		}
	}
	return false
}

// Check for remote filepath, e.g: https://test.webp.sh/node.png
// return StatusCode, etagValue and length
func getRemoteImageInfo(fileURL string) (int, string, string) {
	res, err := http.Head(fileURL)
	if err != nil {
		log.Errorln("Connection to remote error!")
		return http.StatusInternalServerError, "", ""
	}
	defer res.Body.Close()
	if res.StatusCode != 404 {
		etagValue := res.Header.Get("etag")
		if etagValue == "" {
			log.Info("Remote didn't return etag in header, please check.")
		} else {
			return res.StatusCode, etagValue, res.Header.Get("content-length")
		}
	}

	return res.StatusCode, "", res.Header.Get("content-length")
}

func fetchRemoteImage(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Copy bytes here
	bodyBytes := new(bytes.Buffer)
	_, err = bodyBytes.ReadFrom(resp.Body)
	if err != nil {
		return err
	}

	// Check if remote content-type is image using check by filetype instead of content-type returned by origin
	kind, _ := filetype.Match(bodyBytes.Bytes())
	if kind == filetype.Unknown || !strings.Contains(kind.MIME.Value, "image") {
		return fmt.Errorf("remote file %s is not image, remote returned %s", url, resp.Header.Get("content-type"))
	}

	_ = os.MkdirAll(path.Dir(filepath), 0755)
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, bodyBytes)
	return err
}

// Given /path/to/node.png
// Delete /path/to/node.png*
func cleanProxyCache(cacheImagePath string) {
	// Delete /node.png*
	files, err := filepath.Glob(cacheImagePath + "*")
	if err != nil {
		log.Infoln(err)
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			log.Info(err)
		}
	}
}

func genOptimizedAbsPath(rawImagePath string, exhaustPath string, imageName string, reqURI string, extraParams ExtraParams) (string, string) {
	// get file mod time
	STAT, err := os.Stat(rawImagePath)
	if err != nil {
		log.Error(err.Error())
		return "", ""
	}
	ModifiedTime := STAT.ModTime().Unix()
	// webpFilename: abc.jpg.png -> abc.jpg.png.1582558990.webp
	webpFilename := fmt.Sprintf("%s.%d.webp", imageName, ModifiedTime)
	// avifFilename: abc.jpg.png -> abc.jpg.png.1582558990.avif
	avifFilename := fmt.Sprintf("%s.%d.avif", imageName, ModifiedTime)

	// If extraParams not enabled, exhaust path will be /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp
	// If extraParams enabled, and given request at tsuki.jpg?width=200, exhaust path will be /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp_width=200&height=0
	// If extraParams enabled, and given request at tsuki.jpg, exhaust path will be /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp_width=0&height=0
	if config.EnableExtraParams {
		webpFilename = webpFilename + extraParams.String()
		avifFilename = avifFilename + extraParams.String()
	}

	// /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp
	// Custom Exhaust: /path/to/exhaust/web_path/web_to/tsuki.jpg.1582558990.webp
	webpAbsolutePath := path.Clean(path.Join(exhaustPath, path.Dir(reqURI), webpFilename))
	avifAbsolutePath := path.Clean(path.Join(exhaustPath, path.Dir(reqURI), avifFilename))
	return avifAbsolutePath, webpAbsolutePath
}

func genEtag(ImgAbsPath string) string {
	data, err := os.ReadFile(ImgAbsPath)
	if err != nil {
		log.Warn(err)
	}
	crc := crc32.ChecksumIEEE(data)
	return fmt.Sprintf(`W/"%d-%08X"`, len(data), crc)
}

func getCompressionRate(RawImagePath string, optimizedImg string) string {
	originFileInfo, err := os.Stat(RawImagePath)
	if err != nil {
		log.Warnf("Failed to get raw image %v", err)
		return ""
	}
	optimizedFileInfo, err := os.Stat(optimizedImg)
	if err != nil {
		log.Warnf("Failed to get optimized image %v", err)
		return ""
	}
	compressionRate := float64(optimizedFileInfo.Size()) / float64(originFileInfo.Size())
	log.Debugf("The compression rate is %d/%d=%.2f", originFileInfo.Size(), optimizedFileInfo.Size(), compressionRate)
	return fmt.Sprintf(`%.2f`, compressionRate)
}

func guessSupportedFormat(header *fasthttp.RequestHeader) []string {
	var supported = map[string]bool{
		"raw":  true,
		"webp": false,
		"avif": false}

	var ua = string(header.Peek("user-agent"))
	var accept = strings.ToLower(string(header.Peek("accept")))
	log.Debugf("%s\t%s\n", ua, accept)

	if strings.Contains(accept, "image/webp") {
		supported["webp"] = true
	}
	if strings.Contains(accept, "image/avif") {
		supported["avif"] = true
	}

	// chrome on iOS will not send valid image accept header
	if strings.Contains(ua, "iPhone OS 14") || strings.Contains(ua, "CPU OS 14") ||
		strings.Contains(ua, "iPhone OS 15") || strings.Contains(ua, "CPU OS 15") {
		supported["webp"] = true
	} else if strings.Contains(ua, "Android") || strings.Contains(ua, "Linux") {
		supported["webp"] = true
	}

	var accepted []string
	for k, v := range supported {
		if v {
			accepted = append(accepted, k)
		}
	}
	return accepted
}

func findSmallestFiles(files []string) string {
	// walk files
	var small int64
	var final string
	for _, f := range files {
		stat, err := os.Stat(f)
		if err != nil {
			log.Warnf("%s not found on filesystem", f)
			continue
		}
		if stat.Size() < small || small == 0 {
			small = stat.Size()
			final = f
		}
	}
	return final
}

func Sha1Path(uri string) string {
	/* #nosec */
	h := sha1.New()
	h.Write([]byte(uri))
	return hex.EncodeToString(h.Sum(nil))
}
