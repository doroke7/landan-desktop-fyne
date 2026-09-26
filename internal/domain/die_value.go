package domain

// DieValue is the number facing up on a die.
type DieValue struct {
	// Value is 1 to 6.
	Value int
	// Confidence in [0, 1].
	Confidence float32
}
