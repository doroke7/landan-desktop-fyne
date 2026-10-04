package usecaseApplicationRecognitionInference

import (
	"fmt"

	domain "landan-desktop-fyne/internal/domain"
	outputPortAnyClassifier "landan-desktop-fyne/internal/output/port/any/classifier"
	outputPortAnyDetector "landan-desktop-fyne/internal/output/port/any/detector"
	usecasePortRecognitionInference "landan-desktop-fyne/internal/usecase/port/recognition/inference"
)

type DieUsecase struct {
	dieTopDetectorModel     outputPortAnyDetector.DieTopDetectorModel
	dieValueClassifierModel outputPortAnyClassifier.DieValueClassifierModel
}

func NewDieUsecase(
	oDieTopDetectorModel outputPortAnyDetector.DieTopDetectorModel,
	oDieValueClassifierModel outputPortAnyClassifier.DieValueClassifierModel,
) usecasePortRecognitionInference.DieUsecase {
	return &DieUsecase{
		dieTopDetectorModel:     oDieTopDetectorModel,
		dieValueClassifierModel: oDieValueClassifierModel,
	}
}

func (oSelf *DieUsecase) Recognize(aImage []byte) ([]*domain.Die, error) {
	aDies, err := oSelf.dieTopDetectorModel.Recognize(aImage)
	if err != nil {
		return nil, fmt.Errorf("recognize dice: %w", err)
	}

	return aDies, nil
}
