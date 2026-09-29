//go:build !(darwin && cgo)

package ui

import "fyne.io/fyne/v2"

// InstallReopen does nothing off macOS; the tray's "顯示主視窗" brings the window back.
func InstallReopen(oWindow fyne.Window) {}
