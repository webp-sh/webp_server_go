package encoder

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
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
	for i := 0; i < config.Jobs; i++ {
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
			proposedURI := strings.Replace(picAbsPath, config.Config.ImgPath, "", 1)
			avif, webp := helper.GenOptimizedAbsPath(picAbsPath, proposedURI, config.ExtraParams{Width: 0, Height: 0})
			_ = os.MkdirAll(path.Dir(avif), 0755)
			log.Infof("Prefetching %s", picAbsPath)
			go ConvertFilter(picAbsPath, avif, webp, config.ExtraParams{Width: 0, Height: 0}, finishChan)
			_ = bar.Add(<-finishChan)
			return nil
		})

	if err != nil {
		log.Errorln(err)
	}
	elapsed := time.Since(sTime)
	_, _ = fmt.Fprintf(os.Stdout, "Prefetch complete in %s\n\n", elapsed)

}
