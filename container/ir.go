package container

import (
	"context"
	"fmt"

	"landan-desktop-fyne/bootstrap"
	inputApplicationIrInference "landan-desktop-fyne/internal/input/application/ir/inference"
	outputApplicationOnnx "landan-desktop-fyne/internal/output/application/onnx"
	outputApplicationOnnxModel "landan-desktop-fyne/internal/output/application/onnx/model"
	usecaseApplicationIrInference "landan-desktop-fyne/internal/usecase/application/ir/inference"
)

type IRContainer struct {
	IrInferenceDie   *inputApplicationIrInference.DieHandler
	IrInferencePoker *inputApplicationIrInference.PokerHandler
	IrInferenceDisk  *inputApplicationIrInference.DiskHandler
}

func InitIRContainer(oContext context.Context) (*IRContainer, error) {
	oAbstractOnnx, err := outputApplicationOnnx.NewAbstractOnnx(oContext, bootstrap.CONFIG.ONNX.LIBRARY)
	if err != nil {
		return nil, err
	}
	oAbstractModel := outputApplicationOnnxModel.NewAbstractModel(oAbstractOnnx)

	oDieTopDetectorModel, err := outputApplicationOnnxModel.NewDieTopDetectorModel(oAbstractModel)
	if err != nil {
		return nil, fmt.Errorf("init die top detector model: %w", err)
	}
	oDieValueClassifierModel, err := outputApplicationOnnxModel.NewDieValueClassifierModel(oAbstractModel)
	if err != nil {
		return nil, fmt.Errorf("init die value classifier model: %w", err)
	}

	oDieUsecase := usecaseApplicationIrInference.NewDieUsecase(oDieTopDetectorModel, oDieValueClassifierModel)

	return &IRContainer{
		IrInferenceDie:   inputApplicationIrInference.NewDieHandler(oDieUsecase),
		IrInferencePoker: inputApplicationIrInference.NewPokerHandler(),
		IrInferenceDisk:  inputApplicationIrInference.NewDiskHandler(),
	}, nil
}
