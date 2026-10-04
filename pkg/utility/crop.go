package pkgUtility

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
)

// Crop 把 (iX, iY, iWidth, iHeight) 的框（超出圖片的部分會裁掉）從 oSource 切出來，編成 PNG。
func Crop(oSource image.Image, iX int, iY int, iWidth int, iHeight int) ([]byte, error) {
	oBox := image.Rect(iX, iY, iX+iWidth, iY+iHeight).Intersect(oSource.Bounds())
	if oBox.Empty() {
		return nil, fmt.Errorf("box (%d,%d,%d,%d) is outside the image", iX, iY, iX+iWidth, iY+iHeight)
	}

	oSubImager, ok := oSource.(interface {
		SubImage(image.Rectangle) image.Image
	})
	if !ok {
		return nil, fmt.Errorf("image type %T cannot be cropped", oSource)
	}

	var oBuffer bytes.Buffer
	if err := png.Encode(&oBuffer, oSubImager.SubImage(oBox)); err != nil {
		return nil, err
	}

	return oBuffer.Bytes(), nil
}
