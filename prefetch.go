package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	log "github.com/sirupsen/logrus"
)

func prefetchImages(confImgPath string, ExhaustPath string) {
	// maximum ongoing prefetch is depending on your core of CPU
	var sTime = time.Now()
	log.Infof("Prefetching using %d cores", jobs)
	var finishChan = make(chan int, jobs)
	for i := 0; i < jobs; i++ {
		finishChan <- 1
	}

	//prefetch, recursive through the dir
	all := fileCount(confImgPath)
	var bar = progressbar.Default(all, "Prefetching...")
	err := filepath.Walk(confImgPath,
		func(picAbsPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			// RawImagePath string, ImgFilename string, reqURI string
			proposedURI := strings.Replace(picAbsPath, confImgPath, "", 1)
			avif, webp := genOptimizedAbsPath(picAbsPath, ExhaustPath, info.Name(), proposedURI, ExtraParams{Width: 0, Height: 0})
			_ = os.MkdirAll(path.Dir(avif), 0755)
			log.Infof("Prefetching %s", picAbsPath)
			go convertFilter(picAbsPath, avif, webp, ExtraParams{Width: 0, Height: 0}, finishChan)
			_ = bar.Add(<-finishChan)
			return nil
		})

	if err != nil {
		log.Errorln(err)
	}
	elapsed := time.Since(sTime)
	_, _ = fmt.Fprintf(os.Stdout, "Prefetch completeY(^_^)Y in %s\n\n", elapsed)

}
