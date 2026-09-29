// Package camera is the logic of the desktop camera: it decides where recordings go, drives pkg/camera
// and reports progress to whoever shows it (ui). It knows nothing about Fyne.
package camera

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// expandHome turns a leading "~" into the user's home directory.
func expandHome(sPath string) (string, error) {

	if sPath != "~" && !strings.HasPrefix(sPath, "~/") {
		return sPath, nil
	}

	sHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(sHome, strings.TrimPrefix(sPath, "~")), nil
}

// NewRecordingPath returns a new timestamped mp4 path inside sDirectory, creating the folder if needed.
func NewRecordingPath(sDirectory string) (string, error) {

	sDir, err := expandHome(sDirectory)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(sDir, 0o755); err != nil {
		return "", err
	}

	return filepath.Join(sDir, "landan-"+time.Now().Format("20060102-150405")+".mp4"), nil
}

// NewSnapshotDir returns sDirectory for periodic snapshots, creating it if needed.
func NewSnapshotDir(sDirectory string) (string, error) {

	sDir, err := expandHome(sDirectory)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(sDir, 0o755); err != nil {
		return "", err
	}

	return sDir, nil
}
