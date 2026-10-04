package usecaseApplicationRecognitionInference

import (
	"fmt"

	domain "landan-desktop-fyne/internal/domain"
	outputPortAnyPipeline "landan-desktop-fyne/internal/output/port/any/pipeline"
	usecasePortRecognitionInference "landan-desktop-fyne/internal/usecase/port/recognition/inference"
)

type DieUsecase struct {
	diePipeline outputPortAnyPipeline.DiePipeline
}

func NewDieUsecase(oDiePipeline outputPortAnyPipeline.DiePipeline) usecasePortRecognitionInference.DieUsecase {
	return &DieUsecase{
		diePipeline: oDiePipeline,
	}
}

func (oSelf *DieUsecase) Recognize(aImage []byte) ([]*domain.Die, error) {
	aDies, err := oSelf.diePipeline.Recognize(aImage)
	if err != nil {
		return nil, fmt.Errorf("recognize dice: %w", err)
	}

	return aDies, nil
}
