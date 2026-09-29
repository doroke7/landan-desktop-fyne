//go:build darwin && cgo

package ui

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework AppKit -framework Foundation
#include "reopen_darwin.h"
*/
import "C"

import "fyne.io/fyne/v2"

var reopenWindow fyne.Window

// InstallReopen shows oWindow again when the Dock icon is clicked (the window is only hidden when X is pressed).
func InstallReopen(oWindow fyne.Window) {
	reopenWindow = oWindow
	C.InstallReopenHandler()
}

//export goReopen
func goReopen() {
	fyne.Do(func() {
		reopenWindow.Show()
		reopenWindow.RequestFocus()
	})
}
