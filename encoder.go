package main

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/Kagami/go-avif"
	"github.com/chai2010/webp"
	"golang.org/x/image/bmp"
)

func convertFilter(raw, avifPath, webpPath string, c chan int) {
	// all absolute paths

	if !imageExists(avifPath) {
		convertImage(raw, avifPath, "avif")
	}
	if !imageExists(webpPath) {
		convertImage(raw, webpPath, "webp")
	}

	if c != nil {
		c <- 1
	}
}

func convertImage(raw, optimized, itype string) {
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
	//q, _ := strconv.ParseFloat(config.Quality, 32)

	switch itype {
	case "webp":
		webpEncoder(raw, optimized, config.Quality)
	case "avif":
		avifEncoder(raw, optimized, config.Quality)
	}

}

func readRawImage(imgPath string) (img image.Image, err error) {
	data, err := ioutil.ReadFile(imgPath)
	if err != nil {
		log.Errorln(err)
	}

	contentType := getFileContentType(data[:512])
	if strings.Contains(contentType, "jpeg") {
		img, _ = jpeg.Decode(bytes.NewReader(data))
	} else if strings.Contains(contentType, "png") {
		img, _ = png.Decode(bytes.NewReader(data))
	} else if strings.Contains(contentType, "bmp") {
		img, _ = bmp.Decode(bytes.NewReader(data))
	}

	if img == nil {
		errinfo := "image file " + path.Base(imgPath) + " is corrupted or not supported"
		log.Errorln(errinfo)
		return nil, errors.New(errinfo)
	}

	return img, nil
}

func avifEncoder(p1, p2 string, quality float32) {
	var img image.Image
	dst, err := os.Create(p2)
	if err != nil {
		log.Fatalf("Can't create destination file: %v", err)
	}
	img, err = readRawImage(p1)
	if err != nil {
		return
	}

	var avifQuality = int((100 - quality) / 100 * avif.MaxQuality)
	err = avif.Encode(dst, img, &avif.Options{Quality: avifQuality})
	if err != nil {
		log.Fatalf("Can't encode source image: %v", err)
	}

	convertLog("AVIF", p1, p2, quality)
}

func webpEncoder(p1, p2 string, quality float32) {
	// if convert fails, return error; success nil
	var buf bytes.Buffer
	var img image.Image

	img, err := readRawImage(p1)
	if err != nil {
		return
	}

	if err := webp.Encode(&buf, img, &webp.Options{Lossless: false, Quality: quality}); err != nil {
		log.Error(err)
		return
	}

	if err := ioutil.WriteFile(p2, buf.Bytes(), 0644); err != nil {
		log.Error(err)
		return
	}

	convertLog("WebP", p1, p2, quality)

}

func convertLog(itype, p1 string, p2 string, quality float32) {
	oldf, _ := os.Stat(p1)
	newf, _ := os.Stat(p2)
	log.Infof("%s@%.2f%%: %s->%s %d->%d %.2f%% deflated", itype, quality,
		p1, p2, oldf.Size(), newf.Size(), float32(newf.Size())/float32(oldf.Size())*100)
}
