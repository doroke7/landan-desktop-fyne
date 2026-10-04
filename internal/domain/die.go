package domain

// Die is one die found in an image.
type Die struct {
	// Bounding box in pixels, origin at the top-left corner of the image.
	X      int
	Y      int
	Width  int
	Height int
	// Confidence in [0, 1].
	Confidence float32
	// Value is the number facing up; nil until a classifier has read it.
	Value *DieValue
}
