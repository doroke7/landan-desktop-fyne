package outputPortAnyClassifier

import (
	domain "landan-desktop-fyne/internal/domain"
)

type DieValueClassifierModel interface {
	// Classify reads the value of one die from a cropped, encoded (JPEG or PNG) image.
	Classify(aImage []byte) (*domain.DieValue, error)
}
