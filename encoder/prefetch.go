package encoder

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"
	"webp_server_go/config"
	"webp_server_go/helper"

	"github.com/schollz/progressbar/v3"
	log "github.com/sirupsen/logrus"
)

func PrefetchImages() {
	// maximum ongoing prefetch is depending on your core of CPU
	var sTime = time.Now()
	log.Infof("Prefetching using %d cores", config.Jobs)
	var finishChan = make(chan int, config.Jobs)
	for range config.Jobs {
		finishChan <- 1
	}

	//prefetch, recursive through the dir
	all := helper.FileCount(config.Config.ImgPath)
	var bar = progressbar.Default(all, "Prefetching...")
	err := filepath.Walk(config.Config.ImgPath,
		func(picAbsPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			// Only convert files with image extensions, use smaller of config.DefaultAllowedTypes and config.Config.AllowedTypes
			if helper.CheckAllowedExtension(picAbsPath) {
				// File type is allowed by user, check if it is an image
				if helper.CheckImageExtension(picAbsPath) {
					// File is an image, continue
				} else {
					return nil
				}
			} else {
				return nil
			}

			// RawImagePath string, ImgFilename string, reqURI string
			metadata := helper.ReadMetadata(picAbsPath, "", config.LocalHostAlias)
			avifAbsPath, webpAbsPath, jxlAbsPath := helper.GenOptimizedAbsPath(metadata, config.LocalHostAlias)

			// Using avifAbsPath here is the same as using webpAbsPath/jxlAbsPath
			_ = os.MkdirAll(path.Dir(avifAbsPath), 0755)

			log.Infof("Prefetching %s", picAbsPath)

			// Allow all supported formats
			supported := map[string]bool{
				"raw":  true,
				"webp": true,
				"avif": true,
				"jxl":  true,
			}

			go ConvertFilter(picAbsPath, jxlAbsPath, avifAbsPath, webpAbsPath, config.ExtraParams{Width: 0, Height: 0}, supported, finishChan)
			_ = bar.Add(<-finishChan)
			return nil
		})

	if err != nil {
		log.Errorln(err)
	}
	elapsed := time.Since(sTime)
	_, _ = fmt.Fprintf(os.Stdout, "Prefetch complete in %s\n\n", elapsed)

}
