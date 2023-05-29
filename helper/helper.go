package helper

import (
	"bytes"
	"crypto/sha1" //#nosec
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
	"webp_server_go/config"

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

func FileCount(dir string) int64 {
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

func ImageExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	if info.Size() < 100 {
		// means something wrong in exhaust file system
		return false
	}

	// Check if there is lock in cache, retry after 1 second
	maxRetries := 3
	retryDelay := 100 * time.Millisecond // Initial retry delay

	for retry := 0; retry < maxRetries; retry++ {
		if _, found := config.WriteLock.Get(filename); found {
			log.Infof("file %s is locked, retrying in %s", filename, retryDelay)
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
		} else {
			return !info.IsDir()
		}
	}

	return !info.IsDir()
}

func CheckAllowedType(imgFilename string) bool {
	imgFilename = strings.ToLower(imgFilename)
	for _, allowedType := range config.Config.AllowedTypes {
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
func GetRemoteImageInfo(fileURL string) (int, string, string) {
	resp, err := http.Head(fileURL)
	if err != nil {
		log.Errorln("Connection to remote error when getRemoteImageInfo!")
		return http.StatusInternalServerError, "", ""
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		etagValue := resp.Header.Get("etag")
		if etagValue == "" {
			log.Info("Remote didn't return etag in header when getRemoteImageInfo, please check.")
		} else {
			return resp.StatusCode, etagValue, resp.Header.Get("content-length")
		}
	}

	return resp.StatusCode, "", resp.Header.Get("content-length")
}

func FetchRemoteImage(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		log.Errorln("Connection to remote error when fetchRemoteImage!")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("remote returned %s when fetching remote image", resp.Status)
	}

	// Copy bytes here
	bodyBytes := new(bytes.Buffer)
	_, err = bodyBytes.ReadFrom(resp.Body)
	if err != nil {
		return err
	}

	// Check if remote content-type is image using check by filetype instead of content-type returned by origin
	kind, _ := filetype.Match(bodyBytes.Bytes())
	if kind == filetype.Unknown || !strings.Contains(kind.MIME.Value, "image") {
		return fmt.Errorf("remote file %s is not image, remote content has MIME type of %s", url, kind.MIME.Value)
	}

	_ = os.MkdirAll(path.Dir(filepath), 0755)

	// Create Cache here as a lock
	// Key: filepath, Value: true
	config.WriteLock.Set(filepath, true, -1)

	err = os.WriteFile(filepath, bodyBytes.Bytes(), 0600)
	if err != nil {
		return err
	}

	// Delete lock here
	config.WriteLock.Delete(filepath)

	return nil
}

// Given /path/to/node.png
// Delete /path/to/node.png*
func CleanProxyCache(cacheImagePath string) {
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

func GenOptimizedAbsPath(rawImagePath string, exhaustPath string, imageName string, reqURI string, extraParams config.ExtraParams) (string, string) {
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
	if config.Config.EnableExtraParams {
		webpFilename = webpFilename + extraParams.String()
		avifFilename = avifFilename + extraParams.String()
	}

	// /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp
	// Custom Exhaust: /path/to/exhaust/web_path/web_to/tsuki.jpg.1582558990.webp
	webpAbsolutePath := path.Clean(path.Join(exhaustPath, path.Dir(reqURI), webpFilename))
	avifAbsolutePath := path.Clean(path.Join(exhaustPath, path.Dir(reqURI), avifFilename))
	return avifAbsolutePath, webpAbsolutePath
}

func GetCompressionRate(RawImagePath string, optimizedImg string) string {
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
	return fmt.Sprintf(`%.2f`, compressionRate)
}

func GuessSupportedFormat(header *fasthttp.RequestHeader) []string {
	var supported = map[string]bool{
		"raw":  true,
		"webp": false,
		"avif": false}

	var ua = string(header.Peek("user-agent"))
	var accept = strings.ToLower(string(header.Peek("accept")))

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

func FindSmallestFiles(files []string) string {
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
