package handler

import (
	"bytes"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"webp_server_go/config"
	"webp_server_go/helper"

	"github.com/gofiber/fiber/v2"
	"github.com/h2non/filetype"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

// Given /path/to/node.png
// Delete /path/to/node.png*
func cleanProxyCache(cacheImagePath string) {
	// Delete /node.png*
	files, err := filepath.Glob(cacheImagePath + "*")
	if err != nil {
		log.Infoln(err)
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			log.Info(err)
		}
	}
}

func downloadFile(filepath string, url string) {
	resp, err := http.Get(url)
	if err != nil {
		log.Errorln("Connection to remote error when downloadFile!")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		log.Errorf("remote returned %s when fetching remote image", resp.Status)
		return
	}

	// Copy bytes here
	bodyBytes := new(bytes.Buffer)
	_, err = bodyBytes.ReadFrom(resp.Body)
	if err != nil {
		return
	}

	// Check if remote content-type is image using check by filetype instead of content-type returned by origin
	kind, _ := filetype.Match(bodyBytes.Bytes())
	mime := kind.MIME.Value
	if !strings.Contains(mime, "image") {
		log.Errorf("remote file %s is not image, remote content has MIME type of %s", url, mime)
		return
	}

	_ = os.MkdirAll(path.Dir(filepath), 0755)

	// Create Cache here as a lock, so we can prevent incomplete file from being read
	// Key: filepath, Value: true
	config.WriteLock.Set(filepath, true, -1)

	err = os.WriteFile(filepath, bodyBytes.Bytes(), 0600)
	if err != nil {
		// not likely to happen
		return
	}

	// Delete lock here
	config.WriteLock.Delete(filepath)

}

func fetchRemoteImg(url string, subdir string) config.MetaFile {
	// url is https://test.webp.sh/mypic/123.jpg?someother=200&somebugs=200
	// How do we know if the remote img is changed? we're using hash(etag+length)
	var etag string

	cacheKey := subdir + ":" + helper.HashString(url)

	if val, found := config.RemoteCache.Get(cacheKey); found {
		if etagVal, ok := val.(string); ok {
			log.Infof("Using cache for remote addr: %s", url)
			etag = etagVal
		} else {
			config.RemoteCache.Delete(cacheKey)
		}
	}

	if etag == "" {
		log.Infof("Remote Addr is %s, pinging for info...", url)
		etag = pingURL(url)
		if etag != "" {
			config.RemoteCache.Set(cacheKey, etag, cache.DefaultExpiration)
		}
	}

	metadata := helper.ReadMetadata(url, etag, subdir)
	localRawImagePath := path.Join(config.Config.RemoteRawPath, subdir, metadata.Id)
	localExhaustImagePath := path.Join(config.Config.ExhaustPath, subdir, metadata.Id)

	if !helper.ImageExists(localRawImagePath) || metadata.Checksum != helper.HashString(etag) {
		cleanProxyCache(localExhaustImagePath)
		if metadata.Checksum != helper.HashString(etag) {
			// remote file has changed
			log.Info("Remote file changed, updating metadata and fetching image source...")
			helper.DeleteMetadata(url, subdir)
			helper.WriteMetadata(url, etag, subdir)
		} else {
			// local file not exists
			log.Info("Remote file not found in remote-raw, re-fetching...")
		}
		downloadFile(localRawImagePath, url)
	}
	return metadata
}

func pingURL(url string) string {
	// this function will try to return identifiable info, currently include etag, content-length as string
	// anything goes wrong, will return ""
	var etag, length string
	resp, err := http.Head(url)
	if err != nil {
		log.Errorln("Connection to remote error when pingUrl!")
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode == fiber.StatusOK {
		etag = resp.Header.Get("etag")
		length = resp.Header.Get("content-length")
	}
	if etag == "" {
		log.Info("Remote didn't return etag in header when getRemoteImageInfo, please check.")
	}
	return etag + length
}
