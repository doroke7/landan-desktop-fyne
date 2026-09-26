package camera

import (
	"context"
	"image"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// MODE=pipe: preview only.  MODE=record: preview + mp4 + snapshot every 2s.
func TestTmpPipe(t *testing.T) {
	oOptions := Options{Width: 1280, Height: 720, Framerate: 30, Bitrate: "4M",
		PreviewWidth: 640, PreviewHeight: 360, PreviewFramerate: 15}
	sLabel := "GO_PIPE"
	if os.Getenv("MODE") == "record" {
		sLabel = "GO_REC"
		sDir := os.Getenv("OUT")
		oOptions.RecordPath = filepath.Join(sDir, "rec.mp4")
		oOptions.SnapshotDir = sDir
		oOptions.SnapshotEvery = 2 * time.Second
	}

	oCtx, fnCancel := context.WithCancel(context.Background())
	var n int
	go func() {
		time.Sleep(6 * time.Second)
		var r0 syscall.Rusage
		syscall.Getrusage(0, &r0)
		t0, n0 := time.Now(), n
		time.Sleep(12 * time.Second)
		var r1 syscall.Rusage
		syscall.Getrusage(0, &r1)
		dt := time.Since(t0).Seconds()
		c := float64(r1.Utime.Nano()+r1.Stime.Nano()-r0.Utime.Nano()-r0.Stime.Nano()) / 1e9
		t.Logf("%s cpu=%.1f%% preview=%.1f/s", sLabel, c/dt*100, float64(n-n0)/dt)
		fnCancel()
	}()
	if err := Stream(oCtx, oOptions, func(image.Image) { n++ }); err != nil {
		t.Fatal(err)
	}
}
