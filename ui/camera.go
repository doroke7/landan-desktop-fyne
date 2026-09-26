package ui

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	bootstrap "landan-desktop-fyne/bootstrap"
	helper "landan-desktop-fyne/internal/helper"
	camera "landan-desktop-fyne/pkg/camera"
)

type cameraView struct {
	image  *canvas.Image
	status *widget.Label

	mutex  sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

var (
	oCamera         *cameraView
	fnOnCameraState func(bOn bool)
)

// NewCameraView builds the video area shown at the bottom of the main window.
func NewCameraView() *fyne.Container {

	oCamera = &cameraView{
		image:  canvas.NewImageFromImage(nil),
		status: widget.NewLabel(""),
	}
	oCamera.image.FillMode = canvas.ImageFillContain
	oCamera.image.ScaleMode = canvas.ImageScaleFastest
	oCamera.image.SetMinSize(fyne.NewSize(320, 180))

	oVideo := container.NewStack(canvas.NewRectangle(color.Black), oCamera.image)

	return container.NewBorder(nil, oCamera.status, nil, nil, oVideo)
}

// SetCameraStateHandler registers a callback that is told when the camera turns on or off.
func SetCameraStateHandler(fn func(bOn bool)) {
	fnOnCameraState = fn
}

// ToggleCamera starts the camera (showing it in the main window and recording it), or stops it.
func ToggleCamera() {

	if oCamera == nil {
		return
	}

	oCamera.mutex.Lock()
	bRunning := oCamera.cancel != nil
	oCamera.mutex.Unlock()

	if bRunning {
		oCamera.stop()
		return
	}
	oCamera.start()
}

// ShutdownCamera stops the camera and waits for the recording to be finalized.
func ShutdownCamera() {

	if oCamera == nil {
		return
	}

	oCamera.mutex.Lock()
	fnCancel, chDone := oCamera.cancel, oCamera.done
	oCamera.mutex.Unlock()

	if fnCancel == nil {
		return
	}
	fnCancel()
	select {
	case <-chDone:
	case <-time.After(8 * time.Second):
	}
}

func (v *cameraView) start() {

	sPath, err := helper.NewRecordingPath(bootstrap.CONFIG.CAMERA.RECORD_DIRECTORY)
	if err != nil {
		v.status.SetText("錯誤: " + err.Error())
		return
	}

	var sSnapshotDir string
	if bootstrap.CONFIG.CAMERA.SNAPSHOT_INTERVAL > 0 {
		sSnapshotDir, err = helper.NewSnapshotDir(bootstrap.CONFIG.CAMERA.SNAPSHOT_DIRECTORY)
		if err != nil {
			v.status.SetText("錯誤: " + err.Error())
			return
		}
	}

	oCtx, fnCancel := context.WithCancel(context.Background())
	chDone := make(chan struct{})

	v.mutex.Lock()
	v.cancel = fnCancel
	v.done = chDone
	v.mutex.Unlock()

	v.status.SetText("● 錄影中 00:00  " + sPath)
	if fnOnCameraState != nil {
		fnOnCameraState(true)
	}

	go v.tickStatus(oCtx, sPath)

	go func() {
		defer close(chDone)

		// Never block on the UI thread (it may be waiting for us in ShutdownCamera):
		// if the previous frame has not been drawn yet, drop this one.
		var bPending atomic.Bool

		oOptions := camera.Options{
			Device:           bootstrap.CONFIG.CAMERA.DEVICE,
			Width:            bootstrap.CONFIG.CAMERA.WIDTH,
			Height:           bootstrap.CONFIG.CAMERA.HEIGHT,
			Framerate:        bootstrap.CONFIG.CAMERA.FRAMERATE,
			Bitrate:          bootstrap.CONFIG.CAMERA.RECORD_BITRATE,
			PreviewWidth:     bootstrap.CONFIG.CAMERA.PREVIEW_WIDTH,
			PreviewHeight:    bootstrap.CONFIG.CAMERA.PREVIEW_HEIGHT,
			PreviewFramerate: bootstrap.CONFIG.CAMERA.PREVIEW_FRAMERATE,
			RecordPath:       sPath,
			SnapshotDir:      sSnapshotDir,
			SnapshotEvery:    time.Duration(bootstrap.CONFIG.CAMERA.SNAPSHOT_INTERVAL) * time.Second,
		}

		err := camera.Stream(oCtx, oOptions, func(oFrame image.Image) {
			if oCtx.Err() != nil || !bPending.CompareAndSwap(false, true) {
				return
			}
			fyne.Do(func() {
				v.image.Image = oFrame
				v.image.Refresh()
				bPending.Store(false)
			})
		})

		v.mutex.Lock()
		v.cancel = nil
		v.mutex.Unlock()

		fyne.Do(func() {
			v.image.Image = nil
			v.image.Refresh()
			if err != nil {
				v.status.SetText("錯誤: " + err.Error())
			} else {
				v.status.SetText("已儲存: " + sPath)
			}
			if fnOnCameraState != nil {
				fnOnCameraState(false)
			}
		})
	}()
}

func (v *cameraView) stop() {

	v.mutex.Lock()
	fnCancel := v.cancel
	v.mutex.Unlock()

	if fnCancel != nil {
		fnCancel()
	}
}

func (v *cameraView) tickStatus(oCtx context.Context, sPath string) {

	oStart := time.Now()
	oTicker := time.NewTicker(time.Second)
	defer oTicker.Stop()

	for {
		select {
		case <-oCtx.Done():
			return
		case <-oTicker.C:
			nSec := int(time.Since(oStart).Seconds())
			sText := fmt.Sprintf("● 錄影中 %02d:%02d  %s", nSec/60, nSec%60, sPath)
			fyne.Do(func() {
				if oCtx.Err() == nil {
					v.status.SetText(sText)
				}
			})
		}
	}
}
