package helper

import (
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"webp_server_go/config"

	_ "golang.org/x/image/webp"

	"slices"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/h2non/filetype"

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
	_ = filepath.WalkDir(dir,
		func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
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

	for range maxRetries {
		if _, found := config.WriteLock.Get(filename); found {
			log.Infof("file %s is locked, retrying in %s", filename, retryDelay)
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff
			continue
		}
		f, err := os.Open(filename)
		if err != nil {
			return false
		}
		head := make([]byte, 512)
		n, err := f.Read(head)
		_ = f.Close()
		if err != nil && err != io.EOF {
			return false
		}

		kind, _ := filetype.Match(head[:n])

		if kind != filetype.Unknown && strings.HasPrefix(kind.MIME.Value, "image/") {
			return true
		}

		return false
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
		accept    = string(header.Peek("accept"))
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

	supported["webp"] = explicitlyAcceptsMediaType(accept, "image/webp")
	supported["avif"] = explicitlyAcceptsMediaType(accept, "image/avif")
	supported["jxl"] = explicitlyAcceptsMediaType(accept, "image/jxl")

	return supported
}

// explicitlyAcceptsMediaType reports whether Accept explicitly lists mediaType
// with a positive quality value. Wildcards are intentionally ignored so that
// clients which do not advertise a modern image format receive a safe fallback.

// image/* -> fallback to raw format
// image/jxl,q=0 -> return false on JXL
func explicitlyAcceptsMediaType(accept, mediaType string) bool {
	for _, value := range strings.Split(accept, ",") {
		parsedType, params, err := mime.ParseMediaType(strings.TrimSpace(value))
		if err != nil || !strings.EqualFold(parsedType, mediaType) {
			continue
		}

		quality := 1.0
		if rawQuality, ok := params["q"]; ok {
			quality, err = strconv.ParseFloat(rawQuality, 64)
			if err != nil || quality < 0 || quality > 1 {
				continue
			}
		}
		if quality > 0 {
			return true
		}
	}

	return false
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
