package ui

import (
	"image"
	"image/color"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	internalCamera "landan-desktop-fyne/internal/camera"
)

type cameraView struct {
	window fyne.Window
	video  *fyne.Container // the area the native preview overlay is laid over
	image  *canvas.Image
	status *widget.Label

	control       *internalCamera.Controller
	bOverlayShown atomic.Bool
	bPending      atomic.Bool // a frame is queued for the UI thread; drop new ones meanwhile

	// sLastStatus is only touched on the Fyne UI goroutine (every caller runs there, either
	// directly from a menu/tray callback or via fyne.Do) - see setStatus.
	sLastStatus string
}

// setStatus updates the status label, skipping the redraw when the text has not actually
// changed (e.g. repeated errors, or callers that recompute the same string).
func (v *cameraView) setStatus(sText string) {
	if sText == v.sLastStatus {
		return
	}
	v.sLastStatus = sText
	v.status.SetText(sText)
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

	oCamera.control = internalCamera.NewController(internalCamera.Handler{
		OnFrame:  oCamera.onFrame,
		OnStatus: func(sText string) { fyne.Do(func() { oCamera.setStatus(sText) }) },
		OnState:  oCamera.onState,
	})

	oCamera.video = container.New(&overlayLayout{view: oCamera}, canvas.NewRectangle(color.Black), oCamera.image)

	return container.NewBorder(nil, oCamera.status, nil, nil, oCamera.video)
}

// showPreviewOverlay lays the native, GPU-composited preview directly over the video area, in
// place of the small RGBA thumbnail (oCamera.image) - see internal/camera.ShowPreviewOverlay.
// Must run on the main thread (e.g. from inside fyne.Do).
func (v *cameraView) showPreviewOverlay() {
	oPos := v.video.Position()
	oSize := v.video.Size()
	internalCamera.ShowPreviewOverlay(oPos.X, oPos.Y, oSize.Width, oSize.Height)
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

// onFrame runs on the capture goroutine. It never blocks on the UI thread (which may be waiting for
// the camera in ShutdownCamera): if the previous frame has not been drawn yet, the new one is dropped.
// Once the native overlay is up it sits on top of the thumbnail, so frames are no longer fed to it.
func (v *cameraView) onFrame(oFrame image.Image) {

	if v.bOverlayShown.Load() || !v.bPending.CompareAndSwap(false, true) {
		return
	}
	fyne.Do(func() {
		v.image.Image = oFrame
		v.image.Refresh()
		v.bPending.Store(false)
		if !v.bOverlayShown.Load() {
			v.showPreviewOverlay()
			v.bOverlayShown.Store(true)
		}
	})
}

func (v *cameraView) onState(bOn bool) {

	v.bOverlayShown.Store(false)

	fyne.Do(func() {
		if !bOn {
			internalCamera.HidePreviewOverlay()
			v.image.Image = nil
			v.image.Refresh()
		}
		if fnOnCameraState != nil {
			fnOnCameraState(bOn)
		}
	})
}

// SetCameraStateHandler registers a callback that is told when the camera turns on or off.
func SetCameraStateHandler(fn func(bOn bool)) {
	fnOnCameraState = fn
}

// ToggleCamera starts the camera (showing it in the main window and recording it), or stops it.
func ToggleCamera() {
	if oCamera != nil {
		oCamera.control.Toggle()
	}
}

// ShutdownCamera stops the camera and waits for the recording to be finalized.
func ShutdownCamera() {
	if oCamera != nil {
		oCamera.control.Shutdown()
	}
}
