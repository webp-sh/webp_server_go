package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/Kagami/go-avif"
	"github.com/chai2010/webp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/image/bmp"
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

	//we need to create dir first
	err = os.MkdirAll(path.Dir(optimized), 0755)
	if err != nil {
		log.Error(err.Error())
	}
	//q, _ := strconv.ParseFloat(config.Quality, 32)

	switch itype {
	case "webp":
		err = webpEncoder(raw, optimized, config.Quality)
	case "avif":
		err = avifEncoder(raw, optimized, config.Quality)
	}
	return err
}

func readRawImage(imgPath string, maxPixel int) (img image.Image, err error) {
	data, err := os.ReadFile(imgPath)
	if err != nil {
		log.Errorln(err)
	}

	imgExtension := strings.ToLower(path.Ext(imgPath))
	if strings.Contains(imgExtension, "jpeg") || strings.Contains(imgExtension, "jpg") {
		img, err = jpeg.Decode(bytes.NewReader(data))
	} else if strings.Contains(imgExtension, "png") {
		img, err = png.Decode(bytes.NewReader(data))
	} else if strings.Contains(imgExtension, "bmp") {
		img, err = bmp.Decode(bytes.NewReader(data))
	}
	if err != nil || img == nil {
		errinfo := fmt.Sprintf("image file %s is corrupted: %v", imgPath, err)
		log.Errorln(errinfo)
		return nil, errors.New(errinfo)
	}

	x, y := img.Bounds().Max.X, img.Bounds().Max.Y
	if x > maxPixel || y > maxPixel {
		errinfo := fmt.Sprintf("Read image error: %s(%dx%d) is too large", imgPath, x, y)
		log.Warnf(errinfo)
		return nil, errors.New(errinfo)
	}

	return img, nil
}

func avifEncoder(p1, p2 string, quality float32) error {
	var img image.Image
	dst, err := os.Create(p2)
	if err != nil {
		log.Fatalf("Can't create destination file: %v", err)
	}
	// AVIF has a maximum resolution of 65536 x 65536 pixels.
	img, err = readRawImage(p1, avifMax)
	if err != nil {
		return err
	}

	err = avif.Encode(dst, img, &avif.Options{
		Threads:        runtime.NumCPU(),
		Speed:          avif.MaxSpeed,
		Quality:        int((100 - quality) / 100 * avif.MaxQuality),
		SubsampleRatio: nil,
	})

	if err != nil {
		log.Warnf("Can't encode source image: %v to AVIF", err)
	}

	convertLog("AVIF", p1, p2, quality)
	return nil
}

func webpEncoder(p1, p2 string, quality float32) error {
	// if convert fails, return error; success nil
	var buf bytes.Buffer
	var img image.Image
	// The maximum pixel dimensions of a WebP image is 16383 x 16383.
	img, err := readRawImage(p1, webpMax)
	if err != nil {
		return err
	}

	err = webp.Encode(&buf, img, &webp.Options{Lossless: false, Quality: quality})
	if err != nil {
		log.Warnf("Can't encode source image: %v to WebP", err)
	}

	if err := os.WriteFile(p2, buf.Bytes(), 0600); err != nil {
		log.Error(err)
		return err
	}

	convertLog("WebP", p1, p2, quality)
	return nil
}

func convertLog(itype, p1 string, p2 string, quality float32) {
	oldf, _ := os.Stat(p1)
	newf, _ := os.Stat(p2)
	log.Infof("%s@%.2f%%: %s->%s %d->%d %.2f%% deflated", itype, quality,
		p1, p2, oldf.Size(), newf.Size(), float32(newf.Size())/float32(oldf.Size())*100)
}
