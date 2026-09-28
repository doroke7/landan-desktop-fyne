//go:build !(darwin && cgo)

package camera

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ShowPreviewOverlay is only available on macOS (AVCaptureVideoPreviewLayer is an AVFoundation
// type). On this backend it is a no-op.
func ShowPreviewOverlay(fX, fY, fWidth, fHeight float32) {}

// HidePreviewOverlay is only available on macOS. On this backend it is a no-op.
func HidePreviewOverlay() {}

// readJPEG returns the next JPEG from a concatenated MJPEG stream.
// Inside JPEG entropy data every 0xFF is stuffed as FF 00, so the bytes FF D9 only ever mean "end of image".
func readJPEG(oReader *bufio.Reader) ([]byte, error) {

	var aData []byte
	for {
		aChunk, err := oReader.ReadBytes(0xD9)
		aData = append(aData, aChunk...)
		if err != nil {
			return nil, err
		}
		if n := len(aData); n >= 2 && aData[n-2] == 0xFF {
			return aData, nil
		}
	}
}

// tailWriter keeps the last few KB written to it, so an error can show what ffmpeg complained about.
type tailWriter struct {
	mutex sync.Mutex
	data  []byte
}

func (w *tailWriter) Write(aData []byte) (int, error) {

	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.data = append(w.data, aData...)
	if len(w.data) > 4096 {
		w.data = w.data[len(w.data)-4096:]
	}
	return len(aData), nil
}

func (w *tailWriter) String() string {

	w.mutex.Lock()
	defer w.mutex.Unlock()

	return strings.TrimSpace(string(w.data))
}

// dshowTagPattern matches a device line of `ffmpeg -list_devices true -f dshow -i dummy`, new format:
// "Name" (video)   or   "Name" (audio, video)
var dshowTagPattern = regexp.MustCompile(`"([^"]+)"\s*\(([a-z, ]+)\)`)

// parseDshowCameras returns the camera names in the output of `ffmpeg -list_devices true -f dshow -i dummy`.
// Newer ffmpeg tags every device with its kind; older ones list the names under section headers instead.
func parseDshowCameras(sOutput string) []string {

	var aNames []string
	bTagged := false
	for _, sLine := range strings.Split(sOutput, "\n") {
		if aMatch := dshowTagPattern.FindStringSubmatch(sLine); aMatch != nil {
			bTagged = true
			if strings.Contains(aMatch[2], "video") {
				aNames = append(aNames, aMatch[1])
			}
		}
	}
	if bTagged {
		return aNames
	}

	// old format
	bVideoSection := false
	for _, sLine := range strings.Split(sOutput, "\n") {
		switch {
		case strings.Contains(sLine, "DirectShow video devices"):
			bVideoSection = true
		case strings.Contains(sLine, "DirectShow audio devices"):
			bVideoSection = false
		case bVideoSection && !strings.Contains(sLine, "Alternative name"):
			if iStart := strings.Index(sLine, `"`); iStart >= 0 {
				if iEnd := strings.LastIndex(sLine, `"`); iEnd > iStart {
					aNames = append(aNames, sLine[iStart+1:iEnd])
				}
			}
		}
	}
	return aNames
}

// firstDshowCamera asks ffmpeg for the first camera on Windows.
func firstDshowCamera(oCtx context.Context) (string, error) {

	oCmd := exec.CommandContext(oCtx, "ffmpeg", "-hide_banner", "-list_devices", "true", "-f", "dshow", "-i", "dummy")
	hideWindow(oCmd)
	// ffmpeg exits with an error here on purpose ("dummy" is not a device); the list is on stderr.
	aOutput, _ := oCmd.CombinedOutput()

	aNames := parseDshowCameras(string(aOutput))
	if len(aNames) == 0 {
		return "", fmt.Errorf("找不到攝影機,請確認已接上並允許攝影機權限")
	}
	return aNames[0], nil
}

// inputArgs is how ffmpeg opens the camera on this operating system.
func inputArgs(oCtx context.Context, oOptions Options) ([]string, error) {

	sSize := fmt.Sprintf("%dx%d", oOptions.Width, oOptions.Height)
	sRate := strconv.Itoa(oOptions.Framerate)

	switch runtime.GOOS {
	case "windows":
		sDevice := oOptions.Device
		if sDevice == "" {
			var err error
			if sDevice, err = firstDshowCamera(oCtx); err != nil {
				return nil, err
			}
		}
		return []string{"-f", "dshow", "-framerate", sRate, "-video_size", sSize, "-i", "video=" + sDevice}, nil

	case "darwin": // macOS built without cgo
		sDevice := oOptions.Device
		if sDevice == "" {
			sDevice = "default"
		}
		return []string{"-f", "avfoundation", "-framerate", sRate, "-video_size", sSize,
			"-pixel_format", "nv12", // what the hardware encoder takes directly, avoids a conversion
			"-i", sDevice + ":none"}, nil

	default:
		sDevice := oOptions.Device
		if sDevice == "" {
			sDevice = "/dev/video0"
		}
		return []string{"-f", "v4l2", "-framerate", sRate, "-video_size", sSize, "-i", sDevice}, nil
	}
}

// encoderCandidates lists the H.264 encoders to try, hardware ones first.
func encoderCandidates() []string {

	switch runtime.GOOS {
	case "darwin":
		return []string{"h264_videotoolbox", "libx264"}
	case "windows":
		return []string{"h264_nvenc", "h264_qsv", "h264_amf", "h264_mf", "libx264"}
	default:
		return []string{"h264_nvenc", "libx264"}
	}
}

// encoderWorks really encodes a few frames: ffmpeg lists an encoder even when the machine has no such hardware.
func encoderWorks(sName string) bool {

	oCtx, fnCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer fnCancel()

	oCmd := exec.CommandContext(oCtx, "ffmpeg", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "color=c=black:s=640x360:r=5", "-frames:v", "5",
		"-c:v", sName, "-f", "null", "-")
	hideWindow(oCmd)
	return oCmd.Run() == nil
}

var (
	encoderOnce sync.Once
	encoderName string
)

// pickEncoder finds the first encoder that works, once per run (probing takes a moment).
// libx264 is the software fallback and is used without a probe.
func pickEncoder() string {

	encoderOnce.Do(func() {
		encoderName = "libx264"
		for _, sName := range encoderCandidates() {
			if sName == "libx264" || encoderWorks(sName) {
				encoderName = sName
				break
			}
		}
		log.Printf("攝影機錄影編碼器: %s", encoderName)
	})
	return encoderName
}

// encoderArgs are the ffmpeg options that record with the encoder sName.
func encoderArgs(sName, sBitrate string) []string {

	switch sName {
	case "h264_videotoolbox":
		return []string{"-c:v", sName, "-b:v", sBitrate}
	case "libx264":
		return []string{"-c:v", sName, "-preset", "veryfast", "-b:v", sBitrate, "-pix_fmt", "yuv420p"}
	default:
		return []string{"-c:v", sName, "-b:v", sBitrate, "-pix_fmt", "yuv420p"}
	}
}

// Stream reads the webcam with the ffmpeg program.
// One ffmpeg process feeds up to three outputs: an MJPEG stream on stdout for the live preview,
// an H.264 mp4 file, and periodic full-size JPEG snapshots.
// Cancelling oCtx asks ffmpeg to quit so the mp4 is finalized.
func Stream(oCtx context.Context, oOptions Options, fnFrame func(image.Image)) error {

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("找不到 ffmpeg,請先安裝並加入 PATH(Windows: winget install ffmpeg; macOS: brew install ffmpeg)")
	}

	aArgs := []string{"-hide_banner", "-loglevel", "error"}

	aInput, err := inputArgs(oCtx, oOptions)
	if err != nil {
		return err
	}
	aArgs = append(aArgs, aInput...)

	// output 1: live preview, kept small and slow to save CPU
	aArgs = append(aArgs,
		"-vf", fmt.Sprintf("scale=%d:%d", oOptions.PreviewWidth, oOptions.PreviewHeight),
		"-r", strconv.Itoa(oOptions.PreviewFramerate),
		"-f", "image2pipe", "-vcodec", "mjpeg", "-q:v", "5", "-")

	if oOptions.RecordPath != "" {
		// output 2: recording at full size (fragmented mp4 stays playable even if the process dies)
		aArgs = append(aArgs, encoderArgs(pickEncoder(), oOptions.Bitrate)...)
		aArgs = append(aArgs, "-movflags", "+frag_keyframe+empty_moov", "-y", oOptions.RecordPath)
	}

	if oOptions.SnapshotDir != "" && oOptions.SnapshotEvery >= time.Second {
		// output 3: one full-size still every SnapshotEvery, named by time
		aArgs = append(aArgs,
			"-vf", fmt.Sprintf("fps=1/%d", int(oOptions.SnapshotEvery.Seconds())),
			"-q:v", "2", "-f", "image2", "-strftime", "1",
			filepath.Join(oOptions.SnapshotDir, "snapshot-%Y%m%d-%H%M%S.jpg"),
		)
	}

	oCmd := exec.CommandContext(oCtx, "ffmpeg", aArgs...)
	hideWindow(oCmd)

	oStderr := &tailWriter{}
	oCmd.Stderr = oStderr

	oStdin, err := oCmd.StdinPipe()
	if err != nil {
		return err
	}
	oStdout, err := oCmd.StdoutPipe()
	if err != nil {
		return err
	}

	// Instead of killing ffmpeg, send "q" so it flushes and closes the mp4 properly.
	oCmd.Cancel = func() error {
		_, err := io.WriteString(oStdin, "q")
		return err
	}
	oCmd.WaitDelay = 5 * time.Second

	if err := oCmd.Start(); err != nil {
		return fmt.Errorf("無法啟動 ffmpeg: %w", err)
	}

	oReader := bufio.NewReaderSize(oStdout, 1<<20)
	for {
		aData, err := readJPEG(oReader)
		if err != nil {
			break
		}
		oFrame, err := jpeg.Decode(bytes.NewReader(aData))
		if err != nil {
			continue
		}
		fnFrame(oFrame)
	}

	oCmd.Wait()
	if oCtx.Err() != nil {
		return nil
	}
	if sMessage := oStderr.String(); sMessage != "" {
		return fmt.Errorf("無法讀取攝影機(請確認已允許攝影機權限,且沒有被其他程式佔用): %s", sMessage)
	}
	return fmt.Errorf("無法讀取攝影機,請確認已允許攝影機權限,且沒有被其他程式佔用")
}
