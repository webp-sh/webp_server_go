package helper

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"webp_server_go/config"

	"github.com/cespare/xxhash"
	"github.com/h2non/filetype"
	"github.com/valyala/fasthttp"

	log "github.com/sirupsen/logrus"
)

var _ = filetype.AddMatcher(filetype.NewType("avif", "image/avif"), avifMatcher)

func avifMatcher(buf []byte) bool {
	// use hexdump on macOS to see the magic number
	// 0000001c 66747970 61766966 00000000 61766966 6d696631 6d696166
	magicHeader := []byte{
		0x0, 0x0, 0x0, 0x1c,
		0x66, 0x74, 0x79, 0x70,
		0x61, 0x76, 0x69, 0x66,
		0x0, 0x0, 0x0, 0x0,
		0x61, 0x76, 0x69, 0x66,
		0x6d, 0x69, 0x66, 0x31,
		0x6d, 0x69, 0x61, 0x66,
	}

	return len(buf) > 1 && bytes.Equal(buf[:28], magicHeader) || strings.Contains(string(buf), "ftypavif")
}

func GetFileContentType(buffer []byte) string {
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
	//  if file size is less than 100 bytes, we assume it's invalid file
	// png starts with an 8-byte signature, follow by 4 chunks 58 bytes.
	// JPG is 134 bytes.
	// webp is 33 bytes.
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
	for _, allowedType := range config.Config.AllowedTypes {
		if allowedType == "*" {
			return true
		}
		allowedType = "." + strings.ToLower(allowedType)
		if strings.HasSuffix(strings.ToLower(imgFilename), allowedType) {
			return true
		}
	}
	return false
}

func GenOptimizedAbsPath(rawImagePath, reqURI string, extraParams config.ExtraParams) (string, string) {
	// imageName is not needed, we can use reqURI
	// get file mod time
	var (
		imageName   = path.Base(reqURI)
		exhaustPath = config.Config.ExhaustPath
	)
	STAT, err := os.Stat(rawImagePath)
	if err != nil {
		log.Error(err.Error())
		return "", ""
	}
	ModifiedTime := STAT.ModTime().Unix()
	// TODO: just hash it?
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
	var (
		supported = map[string]bool{
			"raw":  true,
			"webp": false,
			"avif": false,
		}

		ua     = string(header.Peek("user-agent"))
		accept = strings.ToLower(string(header.Peek("accept")))
	)

	if strings.Contains(accept, "image/webp") {
		supported["webp"] = true
	}
	if strings.Contains(accept, "image/avif") {
		supported["avif"] = true
	}

	// chrome on iOS will not send valid image accept header
	if strings.Contains(ua, "iPhone OS 14") || strings.Contains(ua, "CPU OS 14") ||
		strings.Contains(ua, "iPhone OS 15") || strings.Contains(ua, "CPU OS 15") ||
		strings.Contains(ua, "iPhone OS 16") || strings.Contains(ua, "CPU OS 16") ||
		strings.Contains(ua, "iPhone OS 17") || strings.Contains(ua, "CPU OS 17") ||
		strings.Contains(ua, "Android") || strings.Contains(ua, "Linux") {
		supported["webp"] = true
	}

	// iOS 16 supports AVIF
	if strings.Contains(ua, "iPhone OS 16") || strings.Contains(ua, "CPU OS 16") ||
		strings.Contains(ua, "iPhone OS 17") || strings.Contains(ua, "CPU OS 17") {
		supported["avif"] = true
	}

	// save true value's key to slice
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

func HashString(uri string) string {
	// xxhash supports cross compile
	return fmt.Sprintf("%x", xxhash.Sum64String(uri))
}
