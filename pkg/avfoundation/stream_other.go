//go:build !(darwin && cgo)

package avfoundation

import (
	"context"
	"errors"
	"image"
)

// Stream is only implemented on macOS with cgo enabled.
func Stream(oCtx context.Context, oOptions Options, fnFrame func(image.Image)) error {
	return errors.New("這個版本不支援攝影機(只有 macOS 且啟用 cgo 才有)")
}
