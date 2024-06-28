package helper

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"webp_server_go/config"

	"slices"

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
	// raw image, need to use filetype to determine
	buf, _ := os.ReadFile(filename)
	return GetContentType(buf)
}

func GetContentType(buf []byte) string {
	// raw image, need to use filetype to determine
	kind, _ := filetype.Match(buf)
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
	if os.IsNotExist(err) || err != nil {
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
	if config.Config.AllowedTypes[0] == "*" {
		return true
	}
	imgFilenameExtension := strings.ToLower(path.Ext(imgFilename))
	imgFilenameExtension = strings.TrimPrefix(imgFilenameExtension, ".") // .jpg -> jpg
	return slices.Contains(config.Config.AllowedTypes, imgFilenameExtension)
}

func GenOptimizedAbsPath(metadata config.MetaFile, subdir string) (string, string, string) {
	webpFilename := fmt.Sprintf("%s.webp", metadata.Id)
	avifFilename := fmt.Sprintf("%s.avif", metadata.Id)
	jxlFilename := fmt.Sprintf("%s.jxl", metadata.Id)
	webpAbsolutePath := path.Clean(path.Join(config.Config.ExhaustPath, subdir, webpFilename))
	avifAbsolutePath := path.Clean(path.Join(config.Config.ExhaustPath, subdir, avifFilename))
	jxlAbsolutePath := path.Clean(path.Join(config.Config.ExhaustPath, subdir, jxlFilename))
	return avifAbsolutePath, webpAbsolutePath, jxlAbsolutePath
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

func GuessSupportedFormat(header *fasthttp.RequestHeader) map[string]bool {
	var (
		supported = map[string]bool{
			"raw":  true,
			"webp": false,
			"avif": false,
			"jxl":  false,
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
	if strings.Contains(accept, "image/jxl") {
		supported["jxl"] = true
	}

	supportedWebPs := []string{"iPhone OS 14", "CPU OS 14", "iPhone OS 15", "CPU OS 15", "iPhone OS 16", "CPU OS 16", "iPhone OS 17", "CPU OS 17"}
	for _, version := range supportedWebPs {
		if strings.Contains(ua, version) {
			supported["webp"] = true
			break
		}
	}

	supportedAVIFs := []string{"iPhone OS 16", "CPU OS 16", "iPhone OS 17", "CPU OS 17"}
	for _, version := range supportedAVIFs {
		if strings.Contains(ua, version) {
			supported["avif"] = true
			break
		}
	}

	// Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15 <- iPad
	// Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3.1 Safari/605.1.15 <- Mac
	// Mozilla/5.0 (iPhone; CPU iPhone OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Mobile/15E148 Safari/604.1 <- iPhone @ Safari
	supportedJXLs := []string{"iPhone OS 17", "CPU OS 17", "Version/17"}
	if strings.Contains(ua, "iPhone") || strings.Contains(ua, "Macintosh") {
		for _, version := range supportedJXLs {
			if strings.Contains(ua, version) {
				supported["jxl"] = true
				break
			}
		}
	}

	return supported
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
