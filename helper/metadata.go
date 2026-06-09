package helper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"webp_server_go/config"

	"github.com/buckket/go-blurhash"
	"github.com/davidbyttow/govips/v2/vips"
	log "github.com/sirupsen/logrus"
)

// Get ID and filepath
// For remote URLs, metadata id is generated from full URL.
func getId(p string, subdir string) (id string, filePath string, santizedPath string) {
	httpRegexpMatcher := regexp.MustCompile(config.HttpRegexp)
	if httpRegexpMatcher.MatchString(p) {
		fileID := HashString(p)
		return fileID, path.Join(config.Config.RemoteRawPath, subdir, fileID) + path.Ext(p), ""
	}
	parsed, _ := url.Parse(p)
	width := parsed.Query().Get("width")
	height := parsed.Query().Get("height")
	max_width := parsed.Query().Get("max_width")
	max_height := parsed.Query().Get("max_height")
	// santizedPath will be /webp_server.jpg?width=200\u0026height=\u0026max_width=\u0026max_height= in local mode when requesting /webp_server.jpg?width=200
	// santizedPath will be https://docs.webp.sh/images/webp_server.jpg?width=400 in proxy mode when requesting /images/webp_server.jpg?width=400 with IMG_PATH = https://docs.webp.sh
	santizedPath = parsed.Path + "?width=" + width + "&height=" + height + "&max_width=" + max_width + "&max_height=" + max_height
	id = HashString(santizedPath)
	filePath = path.Join(config.Config.ImgPath, parsed.Path)

	return id, filePath, santizedPath
}

func ReadMetadata(p, etag string, subdir string) (config.MetaFile, error) {
	// Try to read metadata. If missing/corrupt, rebuild once.
	var metadata config.MetaFile
	var id, _, _ = getId(p, subdir)
	metadataPath := path.Join(config.Config.MetadataPath, subdir, id+".json")

	readAndUnmarshal := func() (config.MetaFile, error) {
		buf, err := os.ReadFile(metadataPath)
		if err != nil {
			return config.MetaFile{}, err
		}
		if err := json.Unmarshal(buf, &metadata); err != nil {
			return config.MetaFile{}, err
		}
		return metadata, nil
	}

	if data, err := readAndUnmarshal(); err == nil {
		return data, nil
	} else {
		log.Warnf("read metadata failed, rebuilding: %s", err)
	}

	// Rebuild metadata once, then try reading again.
	rebuilt, err := WriteMetadata(p, etag, subdir)
	if err != nil {
		return rebuilt, fmt.Errorf("failed to rebuild metadata at %s: %w", metadataPath, err)
	}
	data, err := readAndUnmarshal()
	if err != nil {
		return config.MetaFile{}, fmt.Errorf("failed to read metadata at %s after rebuild: %w", metadataPath, err)
	}
	return data, nil
}

func WriteMetadata(p, etag string, subdir string) (config.MetaFile, error) {
	metadataDir := path.Join(config.Config.MetadataPath, subdir)

	var id, filepath, sant = getId(p, subdir)

	var data = config.MetaFile{
		Id: id,
	}

	if etag != "" {
		data.Path = p
		data.Checksum = HashString(etag)
	} else {
		data.Path = sant
		data.Checksum = HashFile(filepath)
	}

	// Only get image metadata if the file has image extension
	if CheckImageExtension(filepath) {
		imageMeta := getImageMeta(filepath)
		data.ImageMeta = imageMeta
	}

	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return data, fmt.Errorf("create metadata dir %s: %w", metadataDir, err)
	}

	buf, err := json.Marshal(data)
	if err != nil {
		return data, fmt.Errorf("marshal metadata %s: %w", data.Id, err)
	}

	metadataPath := path.Join(config.Config.MetadataPath, subdir, data.Id+".json")
	if err := os.WriteFile(metadataPath, buf, 0644); err != nil {
		return data, fmt.Errorf("write metadata file %s: %w", metadataPath, err)
	}
	return data, nil
}

func getImageMeta(filePath string) (metadata config.ImageMeta) {
	boolFalse.Set(false)
	intMinusOne.Set(-1)
	img, err := vips.LoadImageFromFile(filePath, &vips.ImportParams{
		FailOnError: boolFalse,
		NumPages:    intMinusOne,
	})
	if err != nil {
		log.Warnf("Could not load %s: %s", filePath, err)
		return metadata
	}
	defer img.Close()
	var colorspace string
	switch img.Interpretation() {
	case vips.InterpretationSRGB:
		colorspace = "sRGB"
	case vips.InterpretationYXY:
		colorspace = "YXY"
	case vips.InterpretationFourier:
		colorspace = "Fourier"
	case vips.InterpretationGrey16:
		colorspace = "Grey16"
	case vips.InterpretationMatrix:
		colorspace = "Matrix"
	case vips.InterpretationScRGB:
		colorspace = "scRGB"
	case vips.InterpretationHSV:
		colorspace = "HSV"
	default:
		colorspace = "Unknown"
	}
	// Get image size
	height := img.Metadata().Height
	width := img.Metadata().Width
	numPages := img.Metadata().Pages
	if numPages > 1 {
		height = height / numPages
	}
	var imgFormat string
	switch img.Format() {
	case vips.ImageTypeJPEG:
		imgFormat = "jpeg"
	case vips.ImageTypePNG:
		imgFormat = "png"
	case vips.ImageTypeWEBP:
		imgFormat = "webp"
	case vips.ImageTypeAVIF:
		imgFormat = "avif"
	case vips.ImageTypeGIF:
		imgFormat = "gif"
	case vips.ImageTypeBMP:
		imgFormat = "bmp"
	default:
		imgFormat = "unknown"
	}

	imgBytes, err := img.ToBytes()
	if err != nil {
		log.Error("Error in img.ToBytes", err)
		return
	}

	metadata = config.ImageMeta{
		Width:      width,
		Height:     height,
		Format:     imgFormat,
		Colorspace: colorspace,
		NumPages:   numPages,
		Size:       len(imgBytes),
	}

	// Get blurhash
	_ = img.Thumbnail(32, 32, vips.InterestingAttention)
	imageImage, err := img.ToImage(vips.NewDefaultExportParams())
	if err != nil {
		log.Error("Error in img.ToImage", err)
		return
	}

	blurHash, err := blurhash.Encode(4, 3, imageImage)
	if err != nil {
		log.Error("Error in blurhash", err)
		return
	}

	metadata.Blurhash = blurHash

	return metadata
}

func DeleteMetadata(p string, subdir string) {
	var id, _, _ = getId(p, subdir)
	metadataPath := path.Join(config.Config.MetadataPath, subdir, id+".json")
	err := os.Remove(metadataPath)
	if err != nil {
		log.Warnln("failed to delete metadata", err)
	}
}
