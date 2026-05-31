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

type MetadataTarget struct {
	RemoteURL     string
	LocalRelPath  string
	LocalQueryKey string
	LocalAbsPath  string
}

func ReadMetadataForTarget(target MetadataTarget, etag, subdir string) config.MetaFile {
	if target.RemoteURL != "" {
		return ReadMetadata(target.RemoteURL, etag, subdir)
	}
	return ReadLocalMetadata(target.LocalRelPath, target.LocalQueryKey, target.LocalAbsPath, etag, subdir)
}

func WriteMetadataForTarget(target MetadataTarget, etag, subdir string) config.MetaFile {
	if target.RemoteURL != "" {
		return WriteMetadata(target.RemoteURL, etag, subdir)
	}
	return WriteLocalMetadata(target.LocalRelPath, target.LocalQueryKey, target.LocalAbsPath, etag, subdir)
}

func DeleteMetadataForTarget(target MetadataTarget, subdir string) {
	if target.RemoteURL != "" {
		DeleteMetadata(target.RemoteURL, subdir)
		return
	}
	DeleteLocalMetadata(target.LocalRelPath, target.LocalQueryKey, subdir)
}

func getLocalId(relPath, queryKey string) (id string, sanitizedPath string) {
	sanitizedPath = "/" + relPath + "?" + queryKey
	id = HashString(sanitizedPath)
	return id, sanitizedPath
}

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
	santizedPath = parsed.Path + "?width=" + width + "&height=" + height + "&max_width=" + max_width + "&max_height=" + max_height
	id = HashString(santizedPath)
	filePath = path.Join(config.Config.ImgPath, parsed.Path)

	return id, filePath, santizedPath
}

func ReadLocalMetadata(relPath, queryKey, absPath, etag, subdir string) config.MetaFile {
	var metadata config.MetaFile
	id, _ := getLocalId(relPath, queryKey)

	if buf, err := os.ReadFile(path.Join(config.Config.MetadataPath, subdir, id+".json")); err != nil {
		WriteLocalMetadata(relPath, queryKey, absPath, etag, subdir)
		return ReadLocalMetadata(relPath, queryKey, absPath, etag, subdir)
	} else {
		err = json.Unmarshal(buf, &metadata)
		if err != nil {
			log.Warnf("unmarshal metadata error, possible corrupt file, re-building...: %s", err)
			WriteLocalMetadata(relPath, queryKey, absPath, etag, subdir)
			return ReadLocalMetadata(relPath, queryKey, absPath, etag, subdir)
		}
		return metadata
	}
}

func WriteLocalMetadata(relPath, queryKey, absPath, etag, subdir string) config.MetaFile {
	_ = os.MkdirAll(path.Join(config.Config.MetadataPath, subdir), 0755)

	id, sanitizedPath := getLocalId(relPath, queryKey)

	var data = config.MetaFile{
		Id: id,
	}

	if etag != "" {
		data.Path = "/" + relPath + "?" + queryKey
		data.Checksum = HashString(etag)
	} else {
		data.Path = sanitizedPath
		data.Checksum = HashFile(absPath)
	}

	if CheckImageExtension(absPath) {
		imageMeta := getImageMeta(absPath)
		data.ImageMeta = imageMeta
	}

	buf, _ := json.Marshal(data)
	_ = os.WriteFile(path.Join(config.Config.MetadataPath, subdir, data.Id+".json"), buf, 0644)
	return data
}

func DeleteLocalMetadata(relPath, queryKey, subdir string) {
	id, _ := getLocalId(relPath, queryKey)
	metadataPath := path.Join(config.Config.MetadataPath, subdir, id+".json")
	err := os.Remove(metadataPath)
	if err != nil {
		log.Warnln("failed to delete metadata", err)
	}
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
