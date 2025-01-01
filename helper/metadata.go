package helper

import (
	"encoding/json"
	"net/url"
	"os"
	"path"
	"webp_server_go/config"

	"github.com/buckket/go-blurhash"
	"github.com/davidbyttow/govips/v2/vips"
	log "github.com/sirupsen/logrus"
)

func getId(p string) (string, string, string) {
	var id string
	if config.ProxyMode {
		return HashString(p), "", ""
	}
	parsed, _ := url.Parse(p)
	width := parsed.Query().Get("width")
	height := parsed.Query().Get("height")
	max_width := parsed.Query().Get("max_width")
	max_height := parsed.Query().Get("max_height")
	// santizedPath will be /webp_server.jpg?width=200\u0026height=\u0026max_width=\u0026max_height= in local mode when requesting /webp_server.jpg?width=200
	// santizedPath will be https://docs.webp.sh/images/webp_server.jpg?width=400 in proxy mode when requesting /images/webp_server.jpg?width=400 with IMG_PATH = https://docs.webp.sh
	santizedPath := parsed.Path + "?width=" + width + "&height=" + height + "&max_width=" + max_width + "&max_height=" + max_height
	id = HashString(santizedPath)

	return id, path.Join(config.Config.ImgPath, parsed.Path), santizedPath
}

func ReadMetadata(p, etag string, subdir string) config.MetaFile {
	// try to read metadata, if we can't read, create one
	var metadata config.MetaFile
	var id, _, _ = getId(p)

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

	var id, filepath, sant = getId(p)

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

	imageMeta := getImageMeta(filepath)
	data.ImageMeta = imageMeta

	buf, _ := json.Marshal(data)
	_ = os.WriteFile(path.Join(config.Config.MetadataPath, subdir, data.Id+".json"), buf, 0644)
	return data
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
	var id, _, _ = getId(p)
	metadataPath := path.Join(config.Config.MetadataPath, subdir, id+".json")
	err := os.Remove(metadataPath)
	if err != nil {
		log.Warnln("failed to delete metadata", err)
	}
}
