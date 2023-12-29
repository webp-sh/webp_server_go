package encoder

import (
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
	"webp_server_go/config"
	"webp_server_go/helper"

	"github.com/davidbyttow/govips/v2/vips"
	log "github.com/sirupsen/logrus"
)

var (
	boolFalse   vips.BoolParameter
	intMinusOne vips.IntParameter
	// Source image encoder ignore list for WebP and AVIF
	// We shouldn't convert Unknown and AVIF to WebP
	webpIgnore = []vips.ImageType{vips.ImageTypeUnknown, vips.ImageTypeAVIF}
	// We shouldn't convert Unknown,AVIF and GIF to AVIF
	avifIgnore = append(webpIgnore, vips.ImageTypeGIF)
)

func init() {
	vips.LoggingSettings(nil, vips.LogLevelError)
	vips.Startup(&vips.Config{
		ConcurrencyLevel: runtime.NumCPU(),
	})
	boolFalse.Set(false)
	intMinusOne.Set(-1)
}

func ConvertFilter(rawPath, avifPath, webpPath string, extraParams config.ExtraParams, c chan int) {
	// Wait for the conversion to complete and return the converted image
	retryDelay := 100 * time.Millisecond // Initial retry delay

	for {
		if _, found := config.ConvertLock.Get(rawPath); found {
			log.Debugf("file %s is locked under conversion, retrying in %s", rawPath, retryDelay)
			time.Sleep(retryDelay)
		} else {
			// The lock is released, indicating that the conversion is complete
			break
		}
	}

	// If there is a lock here, it means that another thread is converting the same image
	// Lock rawPath to prevent concurrent conversion
	config.ConvertLock.Set(rawPath, true, -1)
	defer config.ConvertLock.Delete(rawPath)

	var wg sync.WaitGroup
	wg.Add(2)
	if !helper.ImageExists(avifPath) && config.Config.EnableAVIF {
		go func() {
			err := convertImage(rawPath, avifPath, "avif", extraParams)
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
			err := convertImage(rawPath, webpPath, "webp", extraParams)
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

func convertImage(rawPath, optimizedPath, imageType string, extraParams config.ExtraParams) error {
	// we need to create dir first
	var err = os.MkdirAll(path.Dir(optimizedPath), 0755)
	if err != nil {
		log.Error(err.Error())
	}
	// If original image is NEF, convert NEF image to JPG first
	if strings.HasSuffix(strings.ToLower(rawPath), ".nef") {
		var convertedRaw, converted = ConvertRawToJPG(rawPath, optimizedPath)
		// If converted, use converted file as raw
		if converted {
			// Use converted file(JPG) as raw input for further conversion
			rawPath = convertedRaw
			// Remove converted file after conversion
			defer func() {
				log.Infoln("Removing intermediate conversion file:", convertedRaw)
				err := os.Remove(convertedRaw)
				if err != nil {
					log.Warnln("failed to delete converted file", err)
				}
			}()
		}
	}

	// Image is only opened here
	img, err := vips.LoadImageFromFile(rawPath, &vips.ImportParams{
		FailOnError: boolFalse,
	})
	defer img.Close()

	// Pre-process image(auto rotate, resize, etc.)
	preProcessImage(img, imageType, extraParams)

	switch imageType {
	case "webp":
		err = webpEncoder(img, rawPath, optimizedPath, extraParams)
	case "avif":
		err = avifEncoder(img, rawPath, optimizedPath, extraParams)
	}

	return err
}

func avifEncoder(img *vips.ImageRef, rawPath string, optimizedPath string, extraParams config.ExtraParams) error {
	var (
		buf     []byte
		quality = config.Config.Quality
		err     error
	)

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

	if err := os.WriteFile(optimizedPath, buf, 0600); err != nil {
		log.Error(err)
		return err
	}

	convertLog("AVIF", rawPath, optimizedPath, quality)
	return nil
}

func webpEncoder(img *vips.ImageRef, rawPath string, optimizedPath string, extraParams config.ExtraParams) error {
	var (
		buf     []byte
		quality = config.Config.Quality
		err     error
	)

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
		// If some special images cannot encode with default ReductionEffort(0), then retry from 0 to 6
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

	if err := os.WriteFile(optimizedPath, buf, 0600); err != nil {
		log.Error(err)
		return err
	}

	convertLog("WebP", rawPath, optimizedPath, quality)
	return nil
}

func convertLog(itype, rawPath string, optimizedPath string, quality int) {
	oldf, err := os.Stat(rawPath)
	if err != nil {
		log.Error(err)
		return
	}

	newf, err := os.Stat(optimizedPath)
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("%s@%d%%: %s->%s %d->%d %.2f%% deflated", itype, quality,
		rawPath, optimizedPath, oldf.Size(), newf.Size(), float32(newf.Size())/float32(oldf.Size())*100)
}
