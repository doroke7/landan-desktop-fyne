package usecaseApplicationAnyIrInference

import (
	"fmt"

	domain "landan-desktop-fyne/internal/domain"
	outputPortAnyPipeline "landan-desktop-fyne/internal/output/port/any/pipeline"
	usecasePortAnyIrInference "landan-desktop-fyne/internal/usecase/port/any/ir/inference"
)

type DieUsecase struct {
	diePipeline outputPortAnyPipeline.DiePipeline
}

func NewDieUsecase(oDiePipeline outputPortAnyPipeline.DiePipeline) usecasePortAnyIrInference.DieUsecase {
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
