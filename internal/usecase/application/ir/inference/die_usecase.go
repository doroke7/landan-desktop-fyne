package usecaseApplicationIrInference

import (
	"fmt"

	domain "landan-desktop-fyne/internal/domain"
	outputPortAnyModel "landan-desktop-fyne/internal/output/port/any/model"
	usecasePortIrInference "landan-desktop-fyne/internal/usecase/port/ir/inference"
)

type DieUsecase struct {
	dieTopDetectorModel     outputPortAnyModel.DieTopDetectorModel
	dieValueClassifierModel outputPortAnyModel.DieValueClassifierModel
}

func NewDieUsecase(
	oDieTopDetectorModel outputPortAnyModel.DieTopDetectorModel,
	oDieValueClassifierModel outputPortAnyModel.DieValueClassifierModel,
) usecasePortIrInference.DieUsecase {
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
