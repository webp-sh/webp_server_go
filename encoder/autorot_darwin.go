package encoder

import "webp_server_go/vips"

func autorot(img *vips.Image) error {
	return img.Autorot()
}
