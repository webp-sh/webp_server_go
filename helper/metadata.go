package helper

import (
	"encoding/json"
	"net/url"
	"os"
	"path"
	"webp_server_go/config"

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
	// santizedPath will be /webp_server.jpg?width=200\u0026height= in local mode when requesting /webp_server.jpg?width=200
	// santizedPath will be https://docs.webp.sh/images/webp_server.jpg?width=400 in proxy mode when requesting /images/webp_server.jpg?width=400 with IMG_PATH = https://docs.webp.sh
	santizedPath := parsed.Path + "?width=" + width + "&height=" + height
	id = HashString(santizedPath)

	return id, path.Join(config.Config.ImgPath, parsed.Path), santizedPath
}

func ReadMetadata(p, etag string, subdir string) config.MetaFile {
	// try to read metadata, if we can't read, create one
	var metadata config.MetaFile
	var id, _, _ = getId(p)

	buf, err := os.ReadFile(path.Join(config.Metadata, subdir, id+".json"))
	if err != nil {
		log.Warnf("can't read metadata: %s", err)
		WriteMetadata(p, etag, subdir)
		return ReadMetadata(p, etag, subdir)
	}

	err = json.Unmarshal(buf, &metadata)
	if err != nil {
		log.Warnf("unmarshal metadata error, possible corrupt file, re-building...: %s", err)
		WriteMetadata(p, etag, subdir)
		return ReadMetadata(p, etag, subdir)
	}
	return metadata
}

func WriteMetadata(p, etag string, subdir string) config.MetaFile {
	_ = os.MkdirAll(path.Join(config.Metadata, subdir), 0755)

	var id, filepath, sant = getId(p)

	var data = config.MetaFile{
		Id: id,
	}

	if config.ProxyMode {
		data.Path = p
		data.Checksum = HashString(etag)
	} else {
		data.Path = sant
		data.Checksum = HashFile(filepath)
	}

	buf, _ := json.Marshal(data)
	_ = os.WriteFile(path.Join(config.Metadata, subdir, data.Id+".json"), buf, 0644)
	return data
}

func DeleteMetadata(p string, subdir string) {
	var id, _, _ = getId(p)
	metadataPath := path.Join(config.Metadata, subdir, id+".json")
	err := os.Remove(metadataPath)
	if err != nil {
		log.Warnln("failed to delete metadata", err)
	}
}
