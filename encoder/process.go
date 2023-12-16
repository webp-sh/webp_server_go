package encoder

import (
	"errors"
	"os"
	"path"
	"slices"
	"webp_server_go/config"

	"github.com/davidbyttow/govips/v2/vips"
	log "github.com/sirupsen/logrus"
)

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

// Pre-process image(auto rotate, resize, etc.)
func preProcessImage(img *vips.ImageRef, imageType string, extraParams config.ExtraParams) error {
	// Check Width/Height and ignore image formats
	switch imageType {
	case "webp":
		if img.Metadata().Width > config.WebpMax || img.Metadata().Height > config.WebpMax {
			return errors.New("WebP: image too large")
		}
		imageFormat := img.Format()
		if slices.Contains(webpIgnore, imageFormat) {
			// Return err to render original image
			return errors.New("WebP encoder: ignore image type")
		}
	case "avif":
		if img.Metadata().Width > config.AvifMax || img.Metadata().Height > config.AvifMax {
			return errors.New("AVIF: image too large")
		}
		imageFormat := img.Format()
		if slices.Contains(avifIgnore, imageFormat) {
			// Return err to render original image
			return errors.New("AVIF encoder: ignore image type")
		}
	}

	// Auto rotate
	err := img.AutoRotate()
	if err != nil {
		return err
	}
	if config.Config.EnableExtraParams {
		err = resizeImage(img, extraParams)
		if err != nil {
			return err
		}
	}

	return nil
}
