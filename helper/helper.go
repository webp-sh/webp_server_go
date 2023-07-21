package helper

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"webp_server_go/config"

	"github.com/h2non/filetype"

	"github.com/cespare/xxhash"
	"github.com/valyala/fasthttp"

	svg "github.com/h2non/go-is-svg"
	log "github.com/sirupsen/logrus"
)

var _ = filetype.AddMatcher(filetype.NewType("svg", "image/svg+xml"), svgMatcher)

func svgMatcher(buf []byte) bool {
	return svg.Is(buf)
}

func GetFileContentType(filename string) string {
	if strings.HasSuffix(filename, ".webp") {
		return "image/webp"
	} else if strings.HasSuffix(filename, ".avif") {
		return "image/avif"
	} else {
		// raw image, need to use filetype to determine
		buf, _ := os.ReadFile(filename)
		kind, _ := filetype.Match(buf)
		return kind.MIME.Value
	}
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

func GenOptimizedAbsPath(metadata config.MetaFile) (string, string) {
	webpFilename := fmt.Sprintf("%s.webp", metadata.Id)
	avifFilename := fmt.Sprintf("%s.avif", metadata.Id)
	webpAbsolutePath := path.Clean(path.Join(config.Config.ExhaustPath, webpFilename))
	avifAbsolutePath := path.Clean(path.Join(config.Config.ExhaustPath, avifFilename))
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

func HashFile(filepath string) string {
	buf, _ := os.ReadFile(filepath)
	return fmt.Sprintf("%x", xxhash.Sum64(buf))
}
