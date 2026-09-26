//go:build darwin && cgo

package avfoundation

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework AVFoundation -framework Accelerate -framework CoreMedia -framework CoreVideo -framework Foundation
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
	"time"
	"unsafe"
)

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

// Stream reads the default webcam through AVFoundation.
// One capture session feeds three things: a small live preview (fnFrame), an H.264 mp4 file,
// and periodic full-size JPEG snapshots. Cancelling oCtx stops the camera and finalizes the mp4.
func Stream(oCtx context.Context, oOptions Options, fnFrame func(image.Image)) error {

	iBitrate, err := parseBitrate(oOptions.Bitrate)
	if err != nil {
		return err
	}

	var dSnapshotSeconds float64
	if oOptions.SnapshotDir != "" && oOptions.SnapshotEvery >= time.Second {
		dSnapshotSeconds = oOptions.SnapshotEvery.Seconds()
	}

	oStream := &cameraStream{options: oOptions, frame: fnFrame, failed: make(chan error, 1)}
	oHandle := cgo.NewHandle(oStream)
	defer oHandle.Delete()

	cDevice := C.CString("default")
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

	select {
	case <-oCtx.Done():
	case err = <-oStream.failed:
	}

	// Waits until the mp4 is finalized; no callback runs after this.
	C.avf_stream_stop(pStream)

	if oCtx.Err() != nil {
		return nil
	}
	return err
}
