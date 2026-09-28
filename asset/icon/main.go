// Package icon embeds the desktop app's icon image.
package icon

import _ "embed"

//go:embed main.png
var aByteImage []byte

// Bytes returns the app icon image data (JPEG).
func Bytes() []byte {
	return aByteImage
}
