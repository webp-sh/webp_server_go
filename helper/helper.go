package helper

import (
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"webp_server_go/config"

	_ "golang.org/x/image/webp"

	"slices"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/h2non/filetype"
	"github.com/mileusna/useragent"

	"github.com/cespare/xxhash"
	"github.com/valyala/fasthttp"

	svg "github.com/h2non/go-is-svg"
	log "github.com/sirupsen/logrus"
)

var (
	boolFalse   vips.BoolParameter
	intMinusOne vips.IntParameter
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
	if info.Size() == 0 {
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
			f, err := os.Open(filename)
			if err != nil {
				return false
			}
			defer f.Close()
			head := make([]byte, 512)
			n, err := f.Read(head)
			if err != nil && err != io.EOF {
				return false
			}

			kind, _ := filetype.Match(head[:n])

			if kind != filetype.Unknown && strings.HasPrefix(kind.MIME.Value, "image/") {
				return true
			}

			return false
		}
	}
	return false
}

func GetImageExtension(filename string) string {
	return strings.TrimPrefix(strings.ToLower(path.Ext(filename)), ".")
}

// CheckAllowedExtension checks if the image extension is in the user's allowed types
func CheckAllowedExtension(imgFilename string) bool {
	if config.Config.AllowedTypes[0] == "*" {
		return true
	}
	return slices.Contains(config.Config.AllowedTypes, GetImageExtension(imgFilename))
}

// CheckImageExtension checks if the image extension is in the WebP Server Go's default types
func CheckImageExtension(imgFilename string) bool {
	return slices.Contains(config.DefaultAllowedTypes, GetImageExtension(imgFilename))
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
		ua        = string(header.Peek("user-agent"))
		accept    = strings.ToLower(string(header.Peek("accept")))
		supported = map[string]bool{}
	)
	// Initialize all supported formats to false
	for _, item := range config.DefaultAllowedTypes {
		supported[item] = false
	}
	// raw format(jpg,jpeg,png,gif) is always supported
	supported["jpg"] = true
	supported["jpeg"] = true
	supported["png"] = true
	supported["gif"] = true
	supported["svg"] = true
	supported["bmp"] = true

	if strings.Contains(accept, "image/webp") {
		supported["webp"] = true
	}
	if strings.Contains(accept, "image/avif") {
		supported["avif"] = true
	}
	if strings.Contains(accept, "image/jxl") {
		supported["jxl"] = true
	}
	parsedUA := useragent.Parse(ua)

	if parsedUA.IsIOS() && parsedUA.VersionNo.Major >= 14 {
		supported["webp"] = true
	}

	if parsedUA.IsIOS() && parsedUA.VersionNo.Major >= 16 {
		supported["avif"] = true
	}

	// Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15 <- iPad
	// Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3.1 Safari/605.1.15 <- Mac
	// Mozilla/5.0 (iPhone; CPU iPhone OS 17_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Mobile/15E148 Safari/604.1 <- iPhone @ Safari
	if parsedUA.IsIOS() && parsedUA.VersionNo.Major >= 17 {
		supported["jxl"] = true
	}

	if parsedUA.IsSafari() && parsedUA.VersionNo.Major >= 17 {
		supported["heic"] = true
	}

	// Firefox will not send correct accept header on url without image extension, we need to check user agent to see if `Firefox/133` version is supported
	// https://caniuse.com/webp
	if parsedUA.IsFirefox() && parsedUA.VersionNo.Major >= 133 {
		supported["webp"] = true
	}

	// https://caniuse.com/avif
	if parsedUA.IsFirefox() && parsedUA.VersionNo.Major >= 93 {
		supported["avif"] = true
	}

	return supported
}

func CopyFile(src, dst string) error {
	// Read all content of src to data
	data, _ := os.ReadFile(src)
	// Write data to dst
	return os.WriteFile(dst, data, 0644)
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
