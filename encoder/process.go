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
	imageHeight := img.Height()
	imageWidth := img.Width()

	imgHeightWidthRatio := float32(imageHeight) / float32(imageWidth)

	// Here we have width, height and max_width, max_height
	// Both pairs cannot be used at the same time

	// max_height and max_width are used to make sure bigger images are resized to max_height and max_width
	// e.g, 500x500px image with max_width=200,max_height=100 will be resized to 100x100
	// while smaller images are untouched

	// If both are used, we will use width and height

	if extraParams.MaxHeight > 0 && extraParams.MaxWidth > 0 {
		// If any of it exceeds
		if imageHeight > extraParams.MaxHeight || imageWidth > extraParams.MaxWidth {
			// Check which dimension exceeds most
			heightExceedRatio := float32(imageHeight) / float32(extraParams.MaxHeight)
			widthExceedRatio := float32(imageWidth) / float32(extraParams.MaxWidth)
			// If height exceeds more, like 500x500 -> 200x100 (2.5 < 5)
			// Take max_height as new height ,resize and retain ratio
			if heightExceedRatio > widthExceedRatio {
				err := img.Thumbnail(int(float32(extraParams.MaxHeight)/imgHeightWidthRatio), extraParams.MaxHeight, 0)
				if err != nil {
					return err
				}
			} else {
				err := img.Thumbnail(extraParams.MaxWidth, int(float32(extraParams.MaxWidth)*imgHeightWidthRatio), 0)
				if err != nil {
					return err
				}
			}
		}
	}

	if extraParams.MaxHeight > 0 && imageHeight > extraParams.MaxHeight && extraParams.MaxWidth == 0 {
		err := img.Thumbnail(int(float32(extraParams.MaxHeight)/imgHeightWidthRatio), extraParams.MaxHeight, 0)
		if err != nil {
			return err
		}
	}

	if extraParams.MaxWidth > 0 && imageWidth > extraParams.MaxWidth && extraParams.MaxHeight == 0 {
		err := img.Thumbnail(extraParams.MaxWidth, int(float32(extraParams.MaxWidth)*imgHeightWidthRatio), 0)
		if err != nil {
			return err
		}
	}

	if extraParams.Width > 0 && extraParams.Height > 0 {
		var cropInteresting vips.Interesting
		switch config.Config.ExtraParamsCropInteresting {
		case "InterestingNone":
			cropInteresting = vips.InterestingNone
		case "InterestingCentre":
			cropInteresting = vips.InterestingCentre
		case "InterestingEntropy":
			cropInteresting = vips.InterestingEntropy
		case "InterestingAttention":
			cropInteresting = vips.InterestingAttention
		case "InterestingLow":
			cropInteresting = vips.InterestingLow
		case "InterestingHigh":
			cropInteresting = vips.InterestingHigh
		case "InterestingAll":
			cropInteresting = vips.InterestingAll
		default:
			cropInteresting = vips.InterestingAttention
		}

		err := img.Thumbnail(extraParams.Width, extraParams.Height, cropInteresting)
		if err != nil {
			return err
		}
	}
	if extraParams.Width > 0 && extraParams.Height == 0 {
		err := img.Thumbnail(extraParams.Width, int(float32(extraParams.Width)*imgHeightWidthRatio), 0)
		if err != nil {
			return err
		}
	}
	if extraParams.Height > 0 && extraParams.Width == 0 {
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
		NumPages:    intMinusOne,
	})

	if err != nil {
		log.Warnf("Could not load %s: %s", raw, err)
		return
	}

	defer img.Close()

	if hasResizeExtraParams(extraParams) && !config.Config.EnableExtraParams {
		log.Warnf("Extra params disabled, skip resizing for %s", raw)
	}

	applyResize := isResizeApplicable(extraParams, img, raw)

	if applyResize {
		if err := resizeImage(img, extraParams); err != nil {
			log.Warnf("Failed to resize %s: %v", raw, err)
		}
	}

	if config.Config.StripMetadata {
		img.RemoveMetadata()
	}
	buf, _, _ := img.ExportNative()
	_ = os.WriteFile(dest, buf, 0600)
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

	if config.Config.EnableExtraParams {
		err := resizeImage(img, extraParams)
		if err != nil {
			return err
		}
	}
	// Skip auto rotate for GIF/WebP
	if img.Format() == vips.ImageTypeGIF || img.Format() == vips.ImageTypeWEBP {
		return nil
	} else {
		// Auto rotate
		err := img.AutoRotate()
		if err != nil {
			return err
		}
	}

	return nil
}

func hasResizeExtraParams(extraParams config.ExtraParams) bool {
	return extraParams.Width > 0 || extraParams.Height > 0 || extraParams.MaxWidth > 0 || extraParams.MaxHeight > 0
}

func isResizeApplicable(extraParams config.ExtraParams, img *vips.ImageRef, raw string) bool {
	if !hasResizeExtraParams(extraParams) || !config.Config.EnableExtraParams {
		return false
	}

	// Params out of bounds
	if extraParams.Width > 0 && extraParams.Width > config.RawImageMax {
		log.Warnf("Requested width %d exceeds limit %d for %s", extraParams.Width, config.RawImageMax, raw)
		return false
	}
	if extraParams.Height > 0 && extraParams.Height > config.RawImageMax {
		log.Warnf("Requested height %d exceeds limit %d for %s", extraParams.Height, config.RawImageMax, raw)
		return false
	}
	if extraParams.MaxWidth > 0 && extraParams.MaxWidth > config.RawImageMax {
		log.Warnf("Requested max width %d exceeds limit %d for %s", extraParams.MaxWidth, config.RawImageMax, raw)
		return false
	}
	if extraParams.MaxHeight > 0 && extraParams.MaxHeight > config.RawImageMax {
		log.Warnf("Requested max height %d exceeds limit %d for %s", extraParams.MaxHeight, config.RawImageMax, raw)
		return false
	}

	// RawImage size out of bounds
	meta := img.Metadata()
	if meta.Width > config.RawImageMax || meta.Height > config.RawImageMax {
		log.Warnf("Source image %s is %dx%d, exceeds resize limit %d", raw, meta.Width, meta.Height, config.RawImageMax)
		return false
	}

	// Zoom out of bounds
	if extraParams.Width > 0 && extraParams.Height == 0 && meta.Width > 0 {
		targetHeight := int(float64(extraParams.Width) * float64(meta.Height) / float64(meta.Width))
		if targetHeight > config.RawImageMax {
			log.Warnf("Computed height %d exceeds limit %d for %s (requested width %d)", targetHeight, config.RawImageMax, raw, extraParams.Width)
			return false
		}
	}

	if extraParams.Height > 0 && extraParams.Width == 0 && meta.Height > 0 {
		targetWidth := int(float64(extraParams.Height) * float64(meta.Width) / float64(meta.Height))
		if targetWidth > config.RawImageMax {
			log.Warnf("Computed width %d exceeds limit %d for %s (requested height %d)", targetWidth, config.RawImageMax, raw, extraParams.Height)
			return false
		}
	}

	return true
}
