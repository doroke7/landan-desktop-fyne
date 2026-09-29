//go:build darwin && cgo

// Package reopen reports clicks on the macOS Dock icon, which Fyne does not surface itself.
package reopen

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework AppKit -framework Foundation
#include "reopen_darwin.h"
*/
import "C"

var fnOnReopen func()

// Install calls fn (on a native thread; hop to the UI thread yourself) whenever the Dock icon is clicked.
func Install(fn func()) {
	fnOnReopen = fn
	C.InstallReopenHandler()
}

//export goReopen
func goReopen() {
	if fnOnReopen != nil {
		fnOnReopen()
	}
}
