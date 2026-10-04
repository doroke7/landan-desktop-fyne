package outputPortAnyPipeline

import (
	domain "landan-desktop-fyne/internal/domain"
)

type DiePipeline interface {
	// Recognize finds the dice in an encoded (JPEG or PNG) image and reads the value of each one.
	Recognize(aImage []byte) ([]*domain.Die, error)
}
