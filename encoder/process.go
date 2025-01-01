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
	_ = resizeImage(img, extraParams)
	if config.Config.StripMetadata {
		img.RemoveMetadata()
	}
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
