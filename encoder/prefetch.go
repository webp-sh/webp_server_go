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
	// maximum ongoing prefetch is depending on your core of CPU and config
	var sTime = time.Now()
	memManager := GetMemoryManager()
	
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
			// Skip SVG files as they don't need WebP conversion and often cause dimension errors
			if strings.ToLower(filepath.Ext(picAbsPath)) == ".svg" {
				log.Infof("Skipping SVG file: %s", picAbsPath)
				_ = bar.Add(1)
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

			// 使用内存管理器进行预转换
			finishChan := make(chan int, 1)
			job := &ConversionJob{
				RawPath:       picAbsPath,
				JxlPath:       jxlAbsPath,
				AvifPath:      avifAbsPath,
				WebpPath:      webpAbsPath,
				ExtraParams:   config.ExtraParams{Width: 0, Height: 0},
				SupportedFormats: supported,
				Chan:          finishChan,
			}
			
			// Add error recovery mechanism
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("Recovered from panic while processing %s: %v", picAbsPath, r)
					_ = bar.Add(1)
				}
			}()
			
			memManager.SubmitJob(job)
			
			// Add timeout to prevent hanging
			select {
			case <-finishChan:
				_ = bar.Add(1)
			case <-time.After(30 * time.Second):
				log.Warnf("Timeout processing %s after 30 seconds, skipping...", picAbsPath)
				_ = bar.Add(1)
			}
			
			return nil
		})

	if err != nil {
		log.Errorln(err)
	}
	elapsed := time.Since(sTime)
	_, _ = fmt.Fprintf(os.Stdout, "Prefetch complete in %s\n\n", elapsed)
}
