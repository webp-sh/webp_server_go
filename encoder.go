package main

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/davidbyttow/govips/v2/vips"
	log "github.com/sirupsen/logrus"
)

func convertFilter(raw, avifPath, webpPath string, c chan int) {
	// all absolute paths

	var wg sync.WaitGroup
	wg.Add(2)
	if !imageExists(avifPath) && config.EnableAVIF {
		go func() {
			err := convertImage(raw, avifPath, "avif")
			if err != nil {
				log.Errorln(err)
			}
			defer wg.Done()
		}()
	} else {
		wg.Done()
	}

	if !imageExists(webpPath) {
		go func() {
			err := convertImage(raw, webpPath, "webp")
			if err != nil {
				log.Errorln(err)
			}
			defer wg.Done()
		}()
	} else {
		wg.Done()
	}
	wg.Wait()

	if c != nil {
		c <- 1
	}
}

func convertImage(raw, optimized, itype string) error {
	// we don't have abc.jpg.png1582558990.webp
	// delete the old pic and convert a new one.
	// optimized: /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp
	// we'll delete file starts with /home/webp_server/exhaust/path/to/tsuki.jpg.ts.itype

	s := strings.Split(path.Base(optimized), ".")
	pattern := path.Join(path.Dir(optimized), s[0]+"."+s[1]+".*."+s[len(s)-1])

	matches, err := filepath.Glob(pattern)
	if err != nil {
		log.Error(err.Error())
	} else {
		for _, p := range matches {
			_ = os.Remove(p)
		}
	}

	// we need to create dir first
	err = os.MkdirAll(path.Dir(optimized), 0755)
	if err != nil {
		log.Error(err.Error())
	}

	switch itype {
	case "webp":
		err = webpEncoder(raw, optimized, config.Quality)
	case "avif":
		err = avifEncoder(raw, optimized, config.Quality)
	}
	return err
}

func avifEncoder(p1, p2 string, quality int) error {
	// if convert fails, return error; success nil
	var buf []byte
	// AVIF has a maximum resolution of 65536 x 65536 pixels.
	img, err := vips.NewImageFromFile(p1)
	if err != nil {
		return err
	}

	// If quality >= 100, we use lossless mode
	if quality >= 100 {
		buf, _, err = img.ExportAvif(&vips.AvifExportParams{
			Lossless:      true,
			StripMetadata: true,
		})
	} else {
		buf, _, err = img.ExportAvif(&vips.AvifExportParams{
			Quality:       quality,
			Lossless:      false,
			StripMetadata: true,
		})
	}

	if err != nil {
		log.Warnf("Can't encode source image: %v to AVIF", err)
	}

	if err := os.WriteFile(p2, buf, 0600); err != nil {
		log.Error(err)
		return err
	}

	convertLog("AVIF", p1, p2, quality)
	return nil
}

func webpEncoder(p1, p2 string, quality int) error {
	// if convert fails, return error; success nil
	var buf []byte
	// The maximum pixel dimensions of a WebP image is 16383 x 16383.
	img, err := vips.NewImageFromFile(p1)
	if err != nil {
		return err
	}

	// If quality >= 100, we use lossless mode
	if quality >= 100 {
		buf, _, err = img.ExportWebp(&vips.WebpExportParams{
			Lossless:      true,
			StripMetadata: true,
		})
	} else {
		buf, _, err = img.ExportWebp(&vips.WebpExportParams{
			Quality:       quality,
			Lossless:      false,
			StripMetadata: true,
		})
	}

	if err != nil {
		log.Warnf("Can't encode source image: %v to WebP", err)
	}

	if err := os.WriteFile(p2, buf, 0600); err != nil {
		log.Error(err)
		return err
	}

	convertLog("WebP", p1, p2, quality)
	return nil
}

func convertLog(itype, p1 string, p2 string, quality int) {
	oldf, _ := os.Stat(p1)
	newf, _ := os.Stat(p2)
	log.Infof("%s@%d%%: %s->%s %d->%d %.2f%% deflated", itype, quality,
		p1, p2, oldf.Size(), newf.Size(), float32(newf.Size())/float32(oldf.Size())*100)
}
