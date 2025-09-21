package helper

import (
	"bytes"
	"encoding/json"
	"image"
	"net/url"
	"os"
	"path"
	"webp_server_go/config"

	"webp_server_go/vips"

	"github.com/buckket/go-blurhash"
	log "github.com/sirupsen/logrus"
)

// Get ID and filepath
// For ProxyMode, pass in p the remote-raw path
func getId(p string, subdir string) (id string, filePath string, santizedPath string) {
	if config.ProxyMode {
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

func ReadMetadata(p, etag string, subdir string) config.MetaFile {
	// try to read metadata, if we can't read, create one
	var metadata config.MetaFile
	var id, _, _ = getId(p, subdir)

	if buf, err := os.ReadFile(path.Join(config.Config.MetadataPath, subdir, id+".json")); err != nil {
		// First time reading metadata, create one
		WriteMetadata(p, etag, subdir)
		return ReadMetadata(p, etag, subdir)
	} else {
		err = json.Unmarshal(buf, &metadata)
		if err != nil {
			log.Warnf("unmarshal metadata error, possible corrupt file, re-building...: %s", err)
			WriteMetadata(p, etag, subdir)
			return ReadMetadata(p, etag, subdir)
		}
		return metadata
	}
}

func WriteMetadata(p, etag string, subdir string) config.MetaFile {
	_ = os.MkdirAll(path.Join(config.Config.MetadataPath, subdir), 0755)

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

	buf, _ := json.Marshal(data)
	_ = os.WriteFile(path.Join(config.Config.MetadataPath, subdir, data.Id+".json"), buf, 0644)
	return data
}

func getImageMeta(filePath string) (metadata config.ImageMeta) {
	img := LoadImage(filePath)
	defer img.Close()
	var colorspace string
	switch img.Interpretation() {
	case vips.InterpretationSrgb:
		colorspace = "sRGB"
	case vips.InterpretationYxy:
		colorspace = "YXY"
	case vips.InterpretationFourier:
		colorspace = "Fourier"
	case vips.InterpretationGrey16:
		colorspace = "Grey16"
	case vips.InterpretationMatrix:
		colorspace = "Matrix"
	case vips.InterpretationScrgb:
		colorspace = "scRGB"
	case vips.InterpretationHsv:
		colorspace = "HSV"
	default:
		colorspace = "Unknown"
	}
	// Get image size
	height := img.Height()
	width := img.Width()
	numPages := img.Pages()
	if numPages > 1 {
		height = height / numPages
	}
	var (
		imgFormat string
		imgBytes  []byte
	)
	switch img.Format() {
	case vips.ImageTypeJpeg:
		imgFormat = "jpeg"
		imgBytes, _ = img.JpegsaveBuffer(nil)
	case vips.ImageTypePng:
		imgFormat = "png"
		imgBytes, _ = img.PngsaveBuffer(nil)

	case vips.ImageTypeWebp:
		imgFormat = "webp"
		imgBytes, _ = img.WebpsaveBuffer(nil)

	case vips.ImageTypeAvif:
		imgFormat = "avif"
		imgBytes, _ = img.HeifsaveBuffer(&vips.HeifsaveBufferOptions{
			Encoder: vips.HeifEncoderSvt,
		})

	case vips.ImageTypeGif:
		imgFormat = "gif"
		imgBytes, _ = img.GifsaveBuffer(nil)

	case vips.ImageTypeBmp:
		imgFormat = "bmp"
		imgBytes, _ = img.MagicksaveBuffer(&vips.MagicksaveBufferOptions{Format: "bmp"})

	default:
		imgFormat = "unknown"
	}

	metadata = config.ImageMeta{
		Width:      width,
		Height:     height,
		Format:     imgFormat,
		Colorspace: colorspace,
		NumPages:   numPages,
		Size:       len(imgBytes), //TODO old algorithm: wrong way to calculate size?
	}

	// Get blurhash
	_ = img.ThumbnailImage(32, &vips.ThumbnailImageOptions{
		Height: 32,
		Crop:   vips.InterestingAttention,
	})

	reader := bytes.NewReader(imgBytes)
	imageImage, _, err := image.Decode(reader)
	if err != nil {
		log.Error("Error in img.ToImage", err)
		return
	}

	// imageImage: image.Image
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
