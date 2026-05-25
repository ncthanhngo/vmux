package browsersession

import (
	"encoding/base64"
	"image"
	"image/color"
)

// downscale returns a nearest-neighbor-scaled copy of img whose longest edge is
// at most maxDim (no upscaling). Pure stdlib — avoids an image-resize dep; the
// output is only a UI thumbnail, so nearest-neighbor quality is fine.
func downscale(img image.Image, maxDim int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxDim && h <= maxDim {
		return img
	}
	scale := float64(maxDim) / float64(max(w, h))
	nw := int(float64(w) * scale)
	nh := int(float64(h) * scale)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		sy := b.Min.Y + int(float64(y)/scale)
		for x := 0; x < nw; x++ {
			sx := b.Min.X + int(float64(x)/scale)
			dst.Set(x, y, color.RGBAModel.Convert(img.At(sx, sy)))
		}
	}
	return dst
}

func base64PNG(pngBytes []byte) string {
	return base64.StdEncoding.EncodeToString(pngBytes)
}
