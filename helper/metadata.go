package helper

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/url"
	"os"
	"path"
	"webp_server_go/config"
)

func getId(p string) (string, string, string) {
	var id string
	if config.ProxyMode {
		return HashString(p), "", ""
	}
	parsed, _ := url.Parse(p)
	width := parsed.Query().Get("width")
	height := parsed.Query().Get("height")
	santizedPath := parsed.Path + "?width=" + width + "&height=" + height
	id = HashString(santizedPath)

	return id, path.Join(config.Config.ImgPath, parsed.Path), santizedPath
}

func ReadMetadata(p, etag string) config.MetaFile {
	// try to read metadata, if we can't read, create one
	var metadata config.MetaFile
	var id, _, _ = getId(p)

	buf, err := os.ReadFile(path.Join(config.Metadata, id+".json"))
	if err != nil {
		log.Errorf("can't read metadata: %s", err)
		WriteMetadata(p, etag)
		return ReadMetadata(p, etag)
	}

	err = json.Unmarshal(buf, &metadata)
	if err != nil {
		log.Errorf("unmarshal metadata error: %s", err)
		WriteMetadata(p, etag)
		return ReadMetadata(p, etag)
	}
	return metadata
}

func WriteMetadata(p, etag string) config.MetaFile {
	_ = os.Mkdir(config.Metadata, 0755)

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
	_ = os.WriteFile(path.Join(config.Metadata, data.Id+".json"), buf, 0644)
	return data
}
