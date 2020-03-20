package main

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
	giftowebp "github.com/sizeofint/gif-to-webp"

	"github.com/chai2010/webp"
	"golang.org/x/image/bmp"
)

func WebpEncoder(p1, p2 string, quality float32, Log bool, c chan int) (err error) {
	// if convert fails, return error; success nil

	log.Debugf("target: %s with quality of %f", path.Base(p1), quality)
	var buf bytes.Buffer
	var img image.Image

	data, err := ioutil.ReadFile(p1)
	if err != nil {
		ChanErr(c)
		return
	}

	contentType := GetFileContentType(data[:512])
	if strings.Contains(contentType, "jpeg") {
		img, _ = jpeg.Decode(bytes.NewReader(data))
	} else if strings.Contains(contentType, "png") {
		img, _ = png.Decode(bytes.NewReader(data))
	} else if strings.Contains(contentType, "bmp") {
		img, _ = bmp.Decode(bytes.NewReader(data))
	} else if strings.Contains(contentType, "gif") {
		// TODO: need to support animated webp
		//log.Warn("Gif support is not perfect!")
		//img, _ = gif.Decode(bytes.NewReader(data))
		gitBin, err := gitToWebP(data, quality)
		if err != nil {
			return err
		}
		buf.Write(gitBin)
		goto savefile
	}

	if img == nil {
		msg := "image file " + path.Base(p1) + " is corrupted or not supported"
		log.Debug(msg)
		err = errors.New(msg)
		ChanErr(c)
		return
	}

	if err = webp.Encode(&buf, img, &webp.Options{Lossless: false, Quality: quality}); err != nil {
		log.Error(err)
		ChanErr(c)
		return
	}
savefile:
	if err = ioutil.WriteFile(p2, buf.Bytes(), 0644); err != nil {
		log.Error(err)
		ChanErr(c)
		return
	}

	if Log {
		log.Info("Save to " + p2 + " ok!\n")
	}

	ChanErr(c)

	return nil
}

func gitToWebP(gifBin []byte, quality float32) (webPBin []byte, err error) {
	converter := giftowebp.NewConverter()
	converter.LoopCompatibility = false
	//0 有损压缩  1无损压缩
	converter.WebPConfig.SetLossless(0)
	//压缩速度  0-6  0最快 6质量最好
	converter.WebPConfig.SetMethod(0)
	converter.WebPConfig.SetQuality(quality)
	//搞不懂什么意思,例子是这样用的
	converter.WebPAnimEncoderOptions.SetKmin(9)
	converter.WebPAnimEncoderOptions.SetKmax(17)

	return converter.Convert(gifBin)
}
