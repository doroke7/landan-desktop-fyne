package container

import (
	"context"
	"fmt"

	"landan-desktop-fyne/bootstrap"
	"landan-desktop-fyne/internal/classifier"
	"landan-desktop-fyne/internal/detector"
	inputApplicationRecognitionInference "landan-desktop-fyne/internal/input/application/recognition/inference"
	outputApplicationOnnx "landan-desktop-fyne/internal/output/application/onnx"
	outputApplicationOnnxPipeline "landan-desktop-fyne/internal/output/application/onnx/pipeline"
	outputPortAnyPipeline "landan-desktop-fyne/internal/output/port/any/pipeline"
	usecaseApplicationRecognitionInference "landan-desktop-fyne/internal/usecase/application/recognition/inference"
	usecasePortRecognitionInference "landan-desktop-fyne/internal/usecase/port/recognition/inference"
)

type RecognitionContainer struct {
	RecognitionInferenceDie   *inputApplicationRecognitionInference.DieHandler
	RecognitionInferencePoker *inputApplicationRecognitionInference.PokerHandler
	RecognitionInferenceDisk  *inputApplicationRecognitionInference.DiskHandler
}

func InitRecognitionContainer(oContext context.Context) (*RecognitionContainer, error) {
	oDieUsecase, err := newDieUsecase(oContext)
	if err != nil {
		return nil, err
	}

	return &RecognitionContainer{
		RecognitionInferenceDie:   inputApplicationRecognitionInference.NewDieHandler(oDieUsecase),
		RecognitionInferencePoker: inputApplicationRecognitionInference.NewPokerHandler(),
		RecognitionInferenceDisk:  inputApplicationRecognitionInference.NewDiskHandler(),
	}, nil
}

func newDieTopDetector(oContext context.Context) (*outputApplicationOnnx.AbstractOnnx, *detector.DieTopDetector, error) {
	oAbstractOnnx, err := outputApplicationOnnx.NewAbstractOnnx(oContext, bootstrap.CONFIG.ONNX.LIBRARY)
	if err != nil {
		return nil, nil, err
	}

	oDieTopDetector, err := detector.NewDieTopDetector(oAbstractOnnx)
	if err != nil {
		return nil, nil, fmt.Errorf("init die top detector model: %w", err)
	}

	return oAbstractOnnx, oDieTopDetector, nil
}

func newDieUsecase(oContext context.Context) (usecasePortRecognitionInference.DieUsecase, error) {
	oDiePipeline, err := newDiePipeline(oContext)
	if err != nil {
		return nil, err
	}

	return usecaseApplicationRecognitionInference.NewDieUsecase(oDiePipeline), nil
}

func newDiePipeline(oContext context.Context) (outputPortAnyPipeline.DiePipeline, error) {
	oAbstractOnnx, oDieTopDetector, err := newDieTopDetector(oContext)
	if err != nil {
		return nil, err
	}

	oDieValueClassifier, err := classifier.NewDieValueClassifier(oAbstractOnnx)
	if err != nil {
		return nil, fmt.Errorf("init die value classifier model: %w", err)
	}

	return outputApplicationOnnxPipeline.NewDiePipeline(oDieTopDetector, oDieValueClassifier), nil
}
