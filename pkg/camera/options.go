// Package camera captures a webcam, previewing it frame by frame while recording it to an mp4 file
// and saving a still snapshot at a fixed interval.
//
// It has four backends behind the same Stream function:
//   - macOS with cgo: AVFoundation, called directly from Objective-C (stream_darwin.go, avf_stream_darwin.m).
//   - Windows with cgo: Media Foundation, called directly via COM (stream_windows.go, mf_stream_windows.c).
//   - Windows without cgo: the same Media Foundation COM APIs, called through raw vtable pointers via
//     syscall.SyscallN instead of C (stream_windows_syscall.go). No C compiler is needed to build this
//     one, so it's what a CGO_ENABLED=0 cross-compile from another OS produces.
//   - everything else (Linux, macOS without cgo): the ffmpeg program (stream_ffmpeg.go).
package camera

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Options controls what Stream shows and writes to disk.
type Options struct {
	// Device picks the camera. Empty means the system default.
	// AVFoundation and Media Foundation both accept a zero-based index or part of the device's name;
	// the ffmpeg backend takes the device name on Windows and the device path (e.g. /dev/video0) on Linux.
	Device string

	Width     int // capture size
	Height    int
	Framerate int    // capture frames per second
	Bitrate   string // recording bitrate, e.g. "4M"

	PreviewWidth     int // live preview size, kept small to save CPU
	PreviewHeight    int
	PreviewFramerate int

	RecordPath    string        // mp4 file to record into; empty means do not record
	SnapshotDir   string        // folder for snapshots; empty disables snapshots
	SnapshotEvery time.Duration // interval between snapshots; the first one is taken right away
}

// parseBitrate turns "4M", "500k" or "4000000" into bits per second.
func parseBitrate(sBitrate string) (int, error) {

	sText := strings.ToLower(strings.TrimSpace(sBitrate))
	nUnit := 1
	switch {
	case strings.HasSuffix(sText, "k"):
		nUnit, sText = 1000, strings.TrimSuffix(sText, "k")
	case strings.HasSuffix(sText, "m"):
		nUnit, sText = 1000000, strings.TrimSuffix(sText, "m")
	}

	fValue, err := strconv.ParseFloat(sText, 64)
	if err != nil || fValue <= 0 {
		return 0, fmt.Errorf("錄影位元率格式不正確: %q(例: 4M)", sBitrate)
	}
	return int(fValue * float64(nUnit)), nil
}
