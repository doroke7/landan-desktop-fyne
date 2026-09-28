//go:build darwin && cgo

package camera

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework Accelerate -framework AppKit -framework CoreMedia -framework CoreVideo -framework Foundation
#include <stdlib.h>
#include "avf_stream_darwin.h"
*/
import "C"

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"path/filepath"
	"runtime/cgo"
	"sync"
	"time"
	"unsafe"
)

// currentStream is the native handle of the running stream, if any, so ShowPreviewOverlay and
// HidePreviewOverlay can reach it. Only one stream runs at a time in this app.
var (
	currentStreamMutex sync.Mutex
	currentStream       unsafe.Pointer
)

// ShowPreviewOverlay lays the camera feed directly over the app's window content, at (fX, fY,
// fWidth, fHeight) in the window's own top-left-origin point space (i.e. the same units and
// coordinate system as fyne.CanvasObject.Position()/Size()). It is composited by the GPU/window
// server straight off the capture session - no per-frame CPU pixel copy. Safe to call again (e.g.
// on resize) to reposition it. It is a no-op if no stream is running. Must be called on the main
// thread (e.g. from inside fyne.Do).
func ShowPreviewOverlay(fX, fY, fWidth, fHeight float32) {
	currentStreamMutex.Lock()
	pStream := currentStream
	currentStreamMutex.Unlock()

	if pStream != nil {
		C.avf_preview_overlay_show(pStream, C.double(fX), C.double(fY), C.double(fWidth), C.double(fHeight))
	}
}

// HidePreviewOverlay removes the overlay shown by ShowPreviewOverlay, if any.
// Must be called on the main thread.
func HidePreviewOverlay() {
	currentStreamMutex.Lock()
	pStream := currentStream
	currentStreamMutex.Unlock()

	if pStream != nil {
		C.avf_preview_overlay_hide(pStream)
	}
}

// cameraStream is what the AVFoundation callbacks reach through a cgo.Handle.
type cameraStream struct {
	options Options
	frame   func(image.Image)
	failed  chan error
}

// rgbaToImage copies RGBA pixels into an image.RGBA (the buffer is only valid during the callback).
func rgbaToImage(pData *C.uchar, iWidth, iHeight, iStride int) *image.RGBA {

	aRGBA := unsafe.Slice((*byte)(unsafe.Pointer(pData)), iStride*iHeight)

	oImage := image.NewRGBA(image.Rect(0, 0, iWidth, iHeight))
	for y := range iHeight {
		copy(oImage.Pix[y*oImage.Stride:y*oImage.Stride+iWidth*4], aRGBA[y*iStride:y*iStride+iWidth*4])
	}
	return oImage
}

// saveSnapshot writes oImage into sDirectory, named by time.
func saveSnapshot(sDirectory string, oImage image.Image) {

	var oBuffer bytes.Buffer
	if err := jpeg.Encode(&oBuffer, oImage, &jpeg.Options{Quality: 92}); err != nil {
		log.Printf("截圖編碼失敗: %v", err)
		return
	}

	sPath := filepath.Join(sDirectory, "snapshot-"+time.Now().Format("20060102-150405")+".jpg")
	if err := os.WriteFile(sPath, oBuffer.Bytes(), 0o644); err != nil {
		log.Printf("截圖存檔失敗: %v", err)
	}
}

//export goCameraFrame
func goCameraFrame(iHandle C.uintptr_t, iKind C.int, pData *C.uchar, iWidth, iHeight, iStride C.int) {

	oStream := cgo.Handle(iHandle).Value().(*cameraStream)
	oImage := rgbaToImage(pData, int(iWidth), int(iHeight), int(iStride))

	switch iKind {
	case C.AVF_FRAME_PREVIEW:
		oStream.frame(oImage)
	case C.AVF_FRAME_SNAPSHOT:
		// Encoding a full-size JPEG is slow; do not hold up the camera thread.
		go saveSnapshot(oStream.options.SnapshotDir, oImage)
	}
}

//export goCameraError
func goCameraError(iHandle C.uintptr_t, cMessage *C.char) {

	oStream := cgo.Handle(iHandle).Value().(*cameraStream)
	select {
	case oStream.failed <- errors.New(C.GoString(cMessage)):
	default:
	}
}

// Stream reads the webcam through AVFoundation.
// One capture session feeds three things: a small live preview (fnFrame), an H.264 mp4 file,
// and periodic full-size JPEG snapshots. Cancelling oCtx stops the camera and finalizes the mp4.
func Stream(oCtx context.Context, oOptions Options, fnFrame func(image.Image)) error {

	var iBitrate int
	if oOptions.RecordPath != "" {
		var err error
		if iBitrate, err = parseBitrate(oOptions.Bitrate); err != nil {
			return err
		}
	}

	var dSnapshotSeconds float64
	if oOptions.SnapshotDir != "" && oOptions.SnapshotEvery >= time.Second {
		dSnapshotSeconds = oOptions.SnapshotEvery.Seconds()
	}

	oStream := &cameraStream{options: oOptions, frame: fnFrame, failed: make(chan error, 1)}
	oHandle := cgo.NewHandle(oStream)
	defer oHandle.Delete()

	sDevice := oOptions.Device
	if sDevice == "" {
		sDevice = "default"
	}
	cDevice := C.CString(sDevice)
	defer C.free(unsafe.Pointer(cDevice))
	cRecordPath := C.CString(oOptions.RecordPath)
	defer C.free(unsafe.Pointer(cRecordPath))

	var pError *C.char
	pStream := C.avf_stream_start(cDevice,
		C.int(oOptions.Width), C.int(oOptions.Height), C.int(oOptions.Framerate), C.int(iBitrate),
		C.int(oOptions.PreviewWidth), C.int(oOptions.PreviewHeight), C.int(oOptions.PreviewFramerate),
		cRecordPath, C.double(dSnapshotSeconds),
		C.uintptr_t(oHandle), 60000, &pError)
	if pStream == nil {
		sMessage := C.GoString(pError)
		C.free(unsafe.Pointer(pError))
		return fmt.Errorf("無法讀取攝影機: %s", sMessage)
	}

	currentStreamMutex.Lock()
	currentStream = pStream
	currentStreamMutex.Unlock()

	var errCamera error
	select {
	case <-oCtx.Done():
	case errCamera = <-oStream.failed:
	}

	currentStreamMutex.Lock()
	currentStream = nil
	currentStreamMutex.Unlock()

	// Waits until the mp4 is finalized; no callback runs after this.
	C.avf_stream_stop(pStream)

	if oCtx.Err() != nil {
		return nil
	}
	return errCamera
}
