package encoder

import (
	"errors"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"webp_server_go/config"
	"webp_server_go/helper"

	"github.com/davidbyttow/govips/v2/vips"
	log "github.com/sirupsen/logrus"
)

var (
	boolFalse   vips.BoolParameter
	intMinusOne vips.IntParameter
)

func init() {
	vips.Startup(&vips.Config{
		ConcurrencyLevel: runtime.NumCPU(),
	})
	boolFalse.Set(false)
	intMinusOne.Set(-1)

}

func resizeImage(img *vips.ImageRef, extraParams config.ExtraParams) error {
	imgHeightWidthRatio := float32(img.Metadata().Height) / float32(img.Metadata().Width)
	if extraParams.Width > 0 && extraParams.Height > 0 {
		err := img.Thumbnail(extraParams.Width, extraParams.Height, vips.InterestingAttention)
		if err != nil {
			return err
		}
	} else if extraParams.Width > 0 && extraParams.Height == 0 {
		err := img.Thumbnail(extraParams.Width, int(float32(extraParams.Width)*imgHeightWidthRatio), 0)
		if err != nil {
			return err
		}
	} else if extraParams.Height > 0 && extraParams.Width == 0 {
		err := img.Thumbnail(int(float32(extraParams.Height)/imgHeightWidthRatio), extraParams.Height, 0)
		if err != nil {
			return err
		}
	}
	return nil
}

func ConvertFilter(raw, avifPath, webpPath string, extraParams config.ExtraParams, c chan int) {
	// all absolute paths

	var wg sync.WaitGroup
	wg.Add(2)
	if !helper.ImageExists(avifPath) && config.Config.EnableAVIF {
		go func() {
			err := convertImage(raw, avifPath, "avif", extraParams)
			if err != nil {
				log.Errorln(err)
			}
			defer wg.Done()
		}()
	} else {
		wg.Done()
	}

	if !helper.ImageExists(webpPath) {
		go func() {
			err := convertImage(raw, webpPath, "webp", extraParams)
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

func ResizeItself(raw, dest string, extraParams config.ExtraParams) {
	log.Infof("Resize %s itself to %s", raw, dest)

	// we need to create dir first
	var err = os.MkdirAll(path.Dir(dest), 0755)
	if err != nil {
		log.Error(err.Error())
	}

	img, err := vips.LoadImageFromFile(raw, &vips.ImportParams{
		FailOnError: boolFalse,
	})
	if err != nil {
		log.Warnf("Could not load %s: %s", raw, err)
		return
	}
	_ = resizeImage(img, extraParams)
	buf, _, _ := img.ExportNative()
	_ = os.WriteFile(dest, buf, 0600)
	img.Close()
}

func convertImage(raw, optimized, imageType string, extraParams config.ExtraParams) error {
	// we need to create dir first
	var err = os.MkdirAll(path.Dir(optimized), 0755)
	if err != nil {
		log.Error(err.Error())
	}

	switch imageType {
	case "webp":
		err = webpEncoder(raw, optimized, extraParams)
	case "avif":
		err = avifEncoder(raw, optimized, extraParams)
	}
	return err
}

func imageIgnore(imageFormat vips.ImageType) bool {
	// Ignore Unknown, WebP, AVIF
	ignoreList := []vips.ImageType{vips.ImageTypeUnknown, vips.ImageTypeWEBP, vips.ImageTypeAVIF}
	for _, ignore := range ignoreList {
		if imageFormat == ignore {
			// Return err to render original image
			return true
		}
	}
	return false
}

func avifEncoder(p1, p2 string, extraParams config.ExtraParams) error {
	// if convert fails, return error; success nil
	var (
		buf     []byte
		quality = config.Config.Quality
	)
	img, err := vips.LoadImageFromFile(p1, &vips.ImportParams{
		FailOnError: boolFalse,
	})
	if err != nil {
		return err
	}

	if imageIgnore(img.Format()) {
		return errors.New("encoder: ignore image type")
	}

	if config.Config.EnableExtraParams {
		err = resizeImage(img, extraParams)
		if err != nil {
			return err
		}
	}

	// AVIF has a maximum resolution of 65536 x 65536 pixels.
	if img.Metadata().Width > config.AvifMax || img.Metadata().Height > config.AvifMax {
		return errors.New("AVIF: image too large")
	}

	err = img.AutoRotate()
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
		return err
	}

	if err := os.WriteFile(p2, buf, 0600); err != nil {
		log.Error(err)
		return err
	}
	img.Close()

	convertLog("AVIF", p1, p2, quality)
	return nil
}

func webpEncoder(p1, p2 string, extraParams config.ExtraParams) error {
	// if convert fails, return error; success nil
	var (
		buf     []byte
		quality = config.Config.Quality
	)

	img, err := vips.LoadImageFromFile(p1, &vips.ImportParams{
		FailOnError: boolFalse,
		NumPages:    intMinusOne,
	})
	if err != nil {
		return err
	}

	if imageIgnore(img.Format()) {
		return errors.New("encoder: ignore image type")
	}

	// If the image is smaller than the minimum file size set in config, we just copy it without any compressions.
	if config.Config.MinFileSizeBytes > 0 {
		originalSize := fileSizeBytes(p1)
		if originalSize < int64(config.Config.MinFileSizeBytes) {
			copyImage(p1, p2)
			return nil
		}
	}

	if config.Config.EnableExtraParams {
		err = resizeImage(img, extraParams)
		if err != nil {
			return err
		}
	}

	// The maximum pixel dimensions of a WebP image is 16383 x 16383.
	if (img.Metadata().Width > config.WebpMax || img.Metadata().Height > config.WebpMax) && img.Format() != vips.ImageTypeGIF {
		return errors.New("WebP: image too large")
	}

	err = img.AutoRotate()
	if err != nil {
		return err
	}

	// If quality >= 100, we use lossless mode
	if quality >= 100 {
		// Lossless mode will not encounter problems as below, because in libvips as code below
		// 	config.method = ExUtilGetInt(argv[++c], 0, &parse_error);
		//   use_lossless_preset = 0;   // disable -z option
		buf, _, err = img.ExportWebp(&vips.WebpExportParams{
			Lossless:      true,
			StripMetadata: true,
		})
	} else {
		// If some special images cannot encode with default ReductionEffort(0), then try with 4
		// Example: https://github.com/webp-sh/webp_server_go/issues/234
		ep := vips.WebpExportParams{
			Quality:       quality,
			Lossless:      false,
			StripMetadata: true,
		}
		for i := 0; i <= 6; i++ {
			ep.ReductionEffort = i
			buf, _, err = img.ExportWebp(&ep)
			if err != nil && strings.Contains(err.Error(), "unable to encode") {
				log.Warnf("Can't encode image to WebP with ReductionEffort %d, trying higher value...", i)
			} else if err != nil {
				log.Warnf("Can't encode source image to WebP:%v", err)
			} else {
				break
			}
		}
		buf, _, err = img.ExportWebp(&ep)

	}

	if err != nil {
		log.Warnf("Can't encode source image: %v to WebP", err)
		return err
	}

	if err := os.WriteFile(p2, buf, 0600); err != nil {
		log.Error(err)
		return err
	}
	img.Close()

	convertLog("WebP", p1, p2, quality)
	return nil
}

func convertLog(itype, p1 string, p2 string, quality int) {
	oldf, err := os.Stat(p1)
	if err != nil {
		log.Error(err)
		return
	}

	newf, err := os.Stat(p2)
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("%s@%d%%: %s->%s %d->%d %.2f%% deflated", itype, quality,
		p1, p2, oldf.Size(), newf.Size(), float32(newf.Size())/float32(oldf.Size())*100)
}

func fileSizeBytes(file string) int64 {
	fileInfo, err := os.Stat(file)
	if err != nil {
		log.Error(err)
		return 0
	}
	return fileInfo.Size()
}

func copyImage(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	err = os.WriteFile(dst, data, 0600)
	if err != nil {
		return err
	}
	return nil
}
