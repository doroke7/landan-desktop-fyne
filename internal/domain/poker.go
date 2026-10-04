package domain

// Poker is one playing card found in an image.
type Poker struct {
	// Bounding box in pixels, origin at the top-left corner of the image.
	X      int
	Y      int
	Width  int
	Height int
	// Confidence in [0, 1].
	Confidence float32
	// Face is which side of the card faces up; nil until a classifier has read it.
	Face *PokerFace
	// Rank and Suit are only read when Face is "Front"; nil otherwise.
	Rank *PokerRank
	Suit *PokerSuit
}

// PokerFace is which side of a card faces up.
type PokerFace struct {
	// Name is "Front", "Flow" or "Back".
	Name string
	// Confidence in [0, 1].
	Confidence float32
}

// PokerRank is the rank printed on a card.
type PokerRank struct {
	// Name is "A", "2" to "10", "J", "Q" or "K".
	Name string
	// Confidence in [0, 1].
	Confidence float32
}

// PokerSuit is the suit printed on a card.
type PokerSuit struct {
	// Name is "Spade", "Heart", "Diamond" or "Club".
	Name string
	// Confidence in [0, 1].
	Confidence float32
}
