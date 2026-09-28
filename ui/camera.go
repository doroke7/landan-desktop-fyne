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
	pkgCamera "landan-desktop-fyne/pkg/camera"
)

type cameraView struct {
	window fyne.Window
	video  *fyne.Container // the area the native preview overlay is laid over
	image  *canvas.Image
	status *widget.Label

	mutex         sync.Mutex
	cancel        context.CancelFunc
	done          chan struct{}
	bOverlayShown atomic.Bool
}

var (
	oCamera         *cameraView
	fnOnCameraState func(bOn bool)
)

// NewCameraView builds the video area shown at the bottom of the main window.
func NewCameraView(oWindow fyne.Window) *fyne.Container {

	oCamera = &cameraView{
		window: oWindow,
		image:  canvas.NewImageFromImage(nil),
		status: widget.NewLabel(""),
	}
	oCamera.image.FillMode = canvas.ImageFillContain
	oCamera.image.ScaleMode = canvas.ImageScaleFastest
	oCamera.image.SetMinSize(fyne.NewSize(320, 180))

	oCamera.video = container.New(&overlayLayout{view: oCamera}, canvas.NewRectangle(color.Black), oCamera.image)

	return container.NewBorder(nil, oCamera.status, nil, nil, oCamera.video)
}

// showPreviewOverlay lays the native, GPU-composited preview directly over the video area, in
// place of the small RGBA thumbnail (oCamera.image) - see pkg/camera.ShowPreviewOverlay.
// Must run on the main thread (e.g. from inside fyne.Do).
func (v *cameraView) showPreviewOverlay() {
	oPos := v.video.Position()
	oSize := v.video.Size()
	pkgCamera.ShowPreviewOverlay(oPos.X, oPos.Y, oSize.Width, oSize.Height)
}

// overlayLayout stacks its children to fill the available space, exactly like
// layout.NewStackLayout(), but additionally keeps the native camera preview overlay (if currently
// shown) aligned with that area - Fyne calls Layout whenever the window (and so this container) is
// resized, which native subviews outside Fyne's own tree do not otherwise learn about.
type overlayLayout struct {
	view *cameraView
}

func (l *overlayLayout) Layout(aObjects []fyne.CanvasObject, oSize fyne.Size) {
	oTopLeft := fyne.NewPos(0, 0)
	for _, oChild := range aObjects {
		oChild.Resize(oSize)
		oChild.Move(oTopLeft)
	}
	if l.view.bOverlayShown.Load() {
		l.view.showPreviewOverlay()
	}
}

func (l *overlayLayout) MinSize(aObjects []fyne.CanvasObject) fyne.Size {
	var fWidth, fHeight float32
	for _, oChild := range aObjects {
		if oChild.Visible() {
			fWidth = max(fWidth, oChild.MinSize().Width)
			fHeight = max(fHeight, oChild.MinSize().Height)
		}
	}
	return fyne.NewSize(fWidth, fHeight)
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
	v.bOverlayShown.Store(false)

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

		// Shows the native, GPU-composited preview overlay (see cameraView.showPreviewOverlay) the
		// first time a frame arrives, i.e. once the session is actually running, then stops feeding
		// the small RGBA thumbnail below - the overlay sits directly on top of it and does not need
		// a per-frame CPU pixel copy the way the thumbnail does.
		var oShowOverlayOnce sync.Once

		oOptions := pkgCamera.Options{
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

		err := pkgCamera.Stream(oCtx, oOptions, func(oFrame image.Image) {
			if oCtx.Err() != nil || v.bOverlayShown.Load() || !bPending.CompareAndSwap(false, true) {
				return
			}
			fyne.Do(func() {
				v.image.Image = oFrame
				v.image.Refresh()
				bPending.Store(false)
				oShowOverlayOnce.Do(func() {
					v.showPreviewOverlay()
					v.bOverlayShown.Store(true)
				})
			})
		})

		v.bOverlayShown.Store(false)
		fyne.Do(pkgCamera.HidePreviewOverlay)

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
