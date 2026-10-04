package outputPortAnyPipeline

import (
	domain "landan-desktop-fyne/internal/domain"
)

type PokerPipeline interface {
	// Recognize finds the cards in an encoded (JPEG or PNG) image, reads which side faces up,
	// and for the cards facing up also reads the rank and suit. The cards are sorted left to right.
	Recognize(aImage []byte) (*domain.PokerRecognition, error)
}
