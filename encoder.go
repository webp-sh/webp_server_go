package main

import (
	"bytes"
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/chai2010/webp"
	"golang.org/x/image/bmp"
)

func webpEncoder(p1, p2 string, quality float32) (err error) {
	// if convert fails, return error; success nil
	log.Debugf("target: %s with quality of %f", path.Base(p1), quality)
	var buf bytes.Buffer
	var img image.Image

	data, err := ioutil.ReadFile(p1)
	if err != nil {
		return
	}

	contentType := getFileContentType(data[:512])
	if strings.Contains(contentType, "jpeg") {
		img, _ = jpeg.Decode(bytes.NewReader(data))
	} else if strings.Contains(contentType, "png") {
		img, _ = png.Decode(bytes.NewReader(data))
	} else if strings.Contains(contentType, "bmp") {
		img, _ = bmp.Decode(bytes.NewReader(data))
	} else if strings.Contains(contentType, "gif") {
		// TODO: need to support animated webp
		log.Warn("Gif support is not perfect!")
		img, _ = gif.Decode(bytes.NewReader(data))
	}

	if img == nil {
		msg := "image file " + path.Base(p1) + " is corrupted or not supported"
		log.Debug(msg)
		return errors.New(msg)
	}

	if err = webp.Encode(&buf, img, &webp.Options{Lossless: false, Quality: quality}); err != nil {
		log.Error(err)
		return
	}
	if err = ioutil.WriteFile(p2, buf.Bytes(), 0644); err != nil {
		log.Error(err)
		return
	}

	log.Info("Save to " + p2 + " ok!\n")

	return nil
}
