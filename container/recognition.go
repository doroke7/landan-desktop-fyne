package container

import (
	"context"
	"fmt"

	"landan-desktop-fyne/bootstrap"
	inputApplicationRecognitionInference "landan-desktop-fyne/internal/input/application/recognition/inference"
	outputApplicationOnnx "landan-desktop-fyne/internal/output/application/onnx"
	outputApplicationOnnxClassifier "landan-desktop-fyne/internal/output/application/onnx/classifier"
	outputApplicationOnnxDetector "landan-desktop-fyne/internal/output/application/onnx/detector"
	usecaseApplicationRecognitionInference "landan-desktop-fyne/internal/usecase/application/recognition/inference"
)

type RecognitionContainer struct {
	RecognitionInferenceDie   *inputApplicationRecognitionInference.DieHandler
	RecognitionInferencePoker *inputApplicationRecognitionInference.PokerHandler
	RecognitionInferenceDisk  *inputApplicationRecognitionInference.DiskHandler
}

func InitRecognitionContainer(oContext context.Context) (*RecognitionContainer, error) {
	oAbstractOnnx, err := outputApplicationOnnx.NewAbstractOnnx(oContext, bootstrap.CONFIG.ONNX.LIBRARY)
	if err != nil {
		return nil, err
	}

	oDieTopDetectorModel, err := outputApplicationOnnxDetector.NewDieTopDetectorModel(oAbstractOnnx)
	if err != nil {
		return nil, fmt.Errorf("init die top detector model: %w", err)
	}
	oDieValueClassifierModel, err := outputApplicationOnnxClassifier.NewDieValueClassifierModel(oAbstractOnnx)
	if err != nil {
		return nil, fmt.Errorf("init die value classifier model: %w", err)
	}

	oDieUsecase := usecaseApplicationRecognitionInference.NewDieUsecase(oDieTopDetectorModel, oDieValueClassifierModel)

	return &RecognitionContainer{
		RecognitionInferenceDie:   inputApplicationRecognitionInference.NewDieHandler(oDieUsecase),
		RecognitionInferencePoker: inputApplicationRecognitionInference.NewPokerHandler(),
		RecognitionInferenceDisk:  inputApplicationRecognitionInference.NewDiskHandler(),
	}, nil
}
