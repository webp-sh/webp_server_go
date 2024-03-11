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
			// RawImagePath string, ImgFilename string, reqURI string
			metadata := helper.ReadMetadata(picAbsPath, "", config.LocalHostAlias)
			avifAbsPath, webpAbsPath, jxlAbsPath := helper.GenOptimizedAbsPath(metadata, config.LocalHostAlias)
			_ = os.MkdirAll(path.Dir(avifAbsPath), 0755)
			log.Infof("Prefetching %s", picAbsPath)
			go ConvertFilter(picAbsPath, jxlAbsPath, avifAbsPath, webpAbsPath, config.ExtraParams{Width: 0, Height: 0}, finishChan)
			_ = bar.Add(<-finishChan)
			return nil
		})

	if err != nil {
		log.Errorln(err)
	}
	elapsed := time.Since(sTime)
	_, _ = fmt.Fprintf(os.Stdout, "Prefetch complete in %s\n\n", elapsed)

}
