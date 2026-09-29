package camera

import (
	"context"
	"fmt"
	"image"
	"sync"
	"time"

	"landan-desktop-fyne/bootstrap"
	pkgCamera "landan-desktop-fyne/pkg/camera"
)

// Handler is how a Controller reports to its display. Every callback may run on any goroutine, so a
// Fyne caller must hop to the UI thread itself (fyne.Do); callbacks are never invoked concurrently
// with themselves for the same event, and status/state arrive in order.
type Handler struct {
	OnFrame  func(oFrame image.Image) // a new preview frame; drop it if the display is still busy
	OnStatus func(sText string)       // the text to show under the video
	OnState  func(bOn bool)           // the camera turned on (true) or fully stopped (false)
}

// Controller starts and stops the camera and its recording.
type Controller struct {
	handler Handler

	mutex  sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func NewController(oHandler Handler) *Controller {
	return &Controller{handler: oHandler}
}

// Toggle starts the camera (previewing and recording it), or stops it if it is running.
func (c *Controller) Toggle() {

	c.mutex.Lock()
	bRunning := c.cancel != nil
	c.mutex.Unlock()

	if bRunning {
		c.Stop()
		return
	}
	c.start()
}

// Stop asks the camera to stop; OnState(false) follows once the recording is finalized.
func (c *Controller) Stop() {

	c.mutex.Lock()
	fnCancel := c.cancel
	c.mutex.Unlock()

	if fnCancel != nil {
		fnCancel()
	}
}

// Shutdown stops the camera and waits for the recording to be finalized.
func (c *Controller) Shutdown() {

	c.mutex.Lock()
	fnCancel, chDone := c.cancel, c.done
	c.mutex.Unlock()

	if fnCancel == nil {
		return
	}
	fnCancel()
	select {
	case <-chDone:
	case <-time.After(8 * time.Second):
	}
}

// ShowPreviewOverlay lays the native, GPU-composited preview over the given area (macOS only; a no-op
// elsewhere). Must run on the main thread.
func ShowPreviewOverlay(fX, fY, fWidth, fHeight float32) {
	pkgCamera.ShowPreviewOverlay(fX, fY, fWidth, fHeight)
}

// HidePreviewOverlay removes the overlay shown by ShowPreviewOverlay. Must run on the main thread.
func HidePreviewOverlay() {
	pkgCamera.HidePreviewOverlay()
}

func (c *Controller) status(sText string) {
	if c.handler.OnStatus != nil {
		c.handler.OnStatus(sText)
	}
}

func (c *Controller) state(bOn bool) {
	if c.handler.OnState != nil {
		c.handler.OnState(bOn)
	}
}

func (c *Controller) start() {

	oConfig := bootstrap.CONFIG.CAMERA

	sPath, err := NewRecordingPath(oConfig.RECORD_DIRECTORY)
	if err != nil {
		c.status("錯誤: " + err.Error())
		return
	}

	var sSnapshotDir string
	if oConfig.SNAPSHOT_INTERVAL > 0 {
		sSnapshotDir, err = NewSnapshotDir(oConfig.SNAPSHOT_DIRECTORY)
		if err != nil {
			c.status("錯誤: " + err.Error())
			return
		}
	}

	oCtx, fnCancel := context.WithCancel(context.Background())
	chDone := make(chan struct{})

	c.mutex.Lock()
	c.cancel = fnCancel
	c.done = chDone
	c.mutex.Unlock()

	c.status("● 錄影中 00:00  " + sPath)
	c.state(true)

	go c.tickStatus(oCtx, sPath)

	go func() {
		defer close(chDone)

		oOptions := pkgCamera.Options{
			Device:           oConfig.DEVICE,
			Width:            oConfig.WIDTH,
			Height:           oConfig.HEIGHT,
			Framerate:        oConfig.FRAMERATE,
			Bitrate:          oConfig.RECORD_BITRATE,
			PreviewWidth:     oConfig.PREVIEW_WIDTH,
			PreviewHeight:    oConfig.PREVIEW_HEIGHT,
			PreviewFramerate: oConfig.PREVIEW_FRAMERATE,
			RecordPath:       sPath,
			SnapshotDir:      sSnapshotDir,
			SnapshotEvery:    time.Duration(oConfig.SNAPSHOT_INTERVAL) * time.Second,
		}

		err := pkgCamera.Stream(oCtx, oOptions, func(oFrame image.Image) {
			if oCtx.Err() != nil || c.handler.OnFrame == nil {
				return
			}
			c.handler.OnFrame(oFrame)
		})

		c.mutex.Lock()
		c.cancel = nil
		c.mutex.Unlock()

		if err != nil {
			c.status("錯誤: " + err.Error())
		} else {
			c.status("已儲存: " + sPath)
		}
		c.state(false)
	}()
}

func (c *Controller) tickStatus(oCtx context.Context, sPath string) {

	oStart := time.Now()
	oTicker := time.NewTicker(time.Second)
	defer oTicker.Stop()

	for {
		select {
		case <-oCtx.Done():
			return
		case <-oTicker.C:
			nSec := int(time.Since(oStart).Seconds())
			c.status(fmt.Sprintf("● 錄影中 %02d:%02d  %s", nSec/60, nSec%60, sPath))
		}
	}
}
