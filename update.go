package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"

	log "github.com/sirupsen/logrus"
	"github.com/staktrace/go-update"
)

func autoUpdate() {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Download error: %s", err)
		}
	}()

	var api = "https://api.github.com/repos/webp-sh/webp_server_go/releases/latest"
	type Result struct {
		TagName string `json:"tag_name"`
	}
	var res Result
	log.Debugf("Requesting to %s", api)
	resp1, _ := http.Get(api)
	data1, _ := io.ReadAll(resp1.Body)
	_ = json.Unmarshal(data1, &res)
	var gitVersion = res.TagName

	if gitVersion > version {
		log.Infof("Time to update! New version %s found", gitVersion)
	} else {
		log.Debug("No new version found.")
		return
	}

	var filename = fmt.Sprintf("webp-server-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		filename += ".exe"
	}
	log.Info("Downloading binary to update...")
	resp, _ := http.Get(releaseUrl + filename)
	if resp.StatusCode != 200 {
		log.Debugf("%s-%s not found on release.", runtime.GOOS, runtime.GOARCH)
		return
	}

	err := update.Apply(resp.Body, update.Options{})
	if err != nil {
		// error handling
		log.Errorf("Update error. %v", err)
	} else {
		log.Info("Update complete. Please restart to apply changes.")
	}
	_ = resp.Body.Close()
}
