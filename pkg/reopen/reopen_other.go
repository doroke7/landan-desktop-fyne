//go:build !(darwin && cgo)

// Package reopen reports clicks on the macOS Dock icon, which Fyne does not surface itself.
package reopen

// Install does nothing off macOS.
func Install(fn func()) {}
