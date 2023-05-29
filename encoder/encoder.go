package encoder

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"webp_server_go/config"
	"webp_server_go/helper"

	"github.com/davidbyttow/govips/v2/vips"
	log "github.com/sirupsen/logrus"
)

func init() {
	vips.Startup(&vips.Config{
		ConcurrencyLevel: runtime.NumCPU(),
	})
	// TODO can't shutdown vips in init
	//defer vips.Shutdown()
}

func resizeImage(img *vips.ImageRef, extraParams config.ExtraParams) error {
	imgHeightWidthRatio := float32(img.Metadata().Height) / float32(img.Metadata().Width)
	if extraParams.Width > 0 && extraParams.Height > 0 {
		err := img.Thumbnail(extraParams.Width, extraParams.Height, 0)
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

func convertImage(raw, optimized, itype string, extraParams config.ExtraParams) error {
	// we don't have /path/to/tsuki.jpg.1582558990.webp, maybe we have /path/to/tsuki.jpg.1082008000.webp
	// delete the old converted pic and convert a new one.
	// optimized: /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp
	// we'll delete file starts with /home/webp_server/exhaust/path/to/tsuki.jpg.ts.itype
	// If contain extraParams like tsuki.jpg?width=200, exhaust path will be /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp_width=200

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
		err = webpEncoder(raw, optimized, config.Config.Quality, extraParams)
	case "avif":
		err = avifEncoder(raw, optimized, config.Config.Quality, extraParams)
	}
	return err
}

func avifEncoder(p1, p2 string, quality int, extraParams config.ExtraParams) error {
	// if convert fails, return error; success nil
	var buf []byte
	var boolFalse vips.BoolParameter
	boolFalse.Set(false)
	img, err := vips.LoadImageFromFile(p1, &vips.ImportParams{
		FailOnError: boolFalse,
	})
	if err != nil {
		return err
	}
	// Ignore Unknown, WebP, AVIF
	ignoreList := []vips.ImageType{vips.ImageTypeUnknown, vips.ImageTypeWEBP, vips.ImageTypeAVIF}

	imageFormat := img.Format()
	for _, ignore := range ignoreList {
		if imageFormat == ignore {
			// Return err to render original image
			return errors.New("encoder: ignore image type")
		}
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

func webpEncoder(p1, p2 string, quality int, extraParams config.ExtraParams) error {

	// if convert fails, return error; success nil
	var buf []byte
	var boolFalse vips.BoolParameter
	boolFalse.Set(false)
	var intMinusOne vips.IntParameter
	intMinusOne.Set(-1)
	img, err := vips.LoadImageFromFile(p1, &vips.ImportParams{
		FailOnError: boolFalse,
		NumPages:    intMinusOne,
	})
	if err != nil {
		return err
	}
	// Ignore Unknown, WebP, AVIF
	ignoreList := []vips.ImageType{vips.ImageTypeUnknown, vips.ImageTypeWEBP, vips.ImageTypeAVIF}

	imageFormat := img.Format()
	for _, ignore := range ignoreList {
		if imageFormat == ignore {
			// Return err to render original image
			return errors.New("encoder: ignore image type")
		}
	}
	if config.Config.EnableExtraParams {
		err = resizeImage(img, extraParams)
		if err != nil {
			return err
		}
	}

	// The maximum pixel dimensions of a WebP image is 16383 x 16383.
	if (img.Metadata().Width > config.WebpMax || img.Metadata().Height > config.WebpMax) && imageFormat != vips.ImageTypeGIF {
		return errors.New("WebP: image too large")
	}

	err = img.AutoRotate()
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
