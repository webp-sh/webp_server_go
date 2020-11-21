package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func prefetchImages(confImgPath string, ExhaustPath string, QUALITY string) {
	var sTime = time.Now()
	// maximum ongoing prefetch is depending on your core of CPU
	log.Infof("Prefetching using %d cores", jobs)
	var finishChan = make(chan int, jobs)
	for i := 0; i < jobs; i++ {
		finishChan <- 0
	}

	//prefetch, recursive through the dir
	all := fileCount(confImgPath)
	count := 0
	err := filepath.Walk(confImgPath,
		func(picAbsPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// RawImagePath string, ImgFilename string, reqURI string
			proposedURI := strings.Replace(picAbsPath, confImgPath, "", 1)
			_, p2 := genWebpAbs(picAbsPath, ExhaustPath, info.Name(), proposedURI)
			q, _ := strconv.ParseFloat(QUALITY, 32)
			_ = os.MkdirAll(path.Dir(p2), 0755)
			go webpEncoder(picAbsPath, p2, float32(q), false, finishChan)
			count += <-finishChan
			//progress bar
			_, _ = fmt.Fprintf(os.Stdout, "[Webp Server started] - convert in progress: %d/%d\r", count, all)
			return nil
		})
	if err != nil {
		log.Debug(err)
	}
	elapsed := time.Since(sTime)
	_, _ = fmt.Fprintf(os.Stdout, "Prefetch completeY(^_^)Y\n\n")
	_, _ = fmt.Fprintf(os.Stdout, "convert %d file in %s (^_^)Y\n\n", count, elapsed)

}
