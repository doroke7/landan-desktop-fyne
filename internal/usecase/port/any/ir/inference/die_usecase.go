package usecasePortAnyIrInference

import (
	domain "landan-desktop-fyne/internal/domain"
)

type DieUsecase interface {
	// Recognize finds the dice in an encoded (JPEG or PNG) image.
	Recognize(aImage []byte) ([]*domain.Die, error)
}
