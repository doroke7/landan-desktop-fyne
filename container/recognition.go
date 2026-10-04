package container

import (
	"context"
	"fmt"

	"landan-desktop-fyne/bootstrap"
	"landan-desktop-fyne/internal/classifier"
	"landan-desktop-fyne/internal/detector"
	inputApplicationRecognitionIrInference "landan-desktop-fyne/internal/input/application/recognition/ir/inference"
	outputApplicationOnnx "landan-desktop-fyne/internal/output/application/onnx"
	outputApplicationOnnxPipeline "landan-desktop-fyne/internal/output/application/onnx/pipeline"
	outputPortAnyPipeline "landan-desktop-fyne/internal/output/port/any/pipeline"
	usecaseApplicationAnyIrInference "landan-desktop-fyne/internal/usecase/application/any/ir/inference"
	usecasePortAnyIrInference "landan-desktop-fyne/internal/usecase/port/any/ir/inference"
)

type RecognitionContainer struct {
	RecognitionInferenceDie   *inputApplicationRecognitionIrInference.DieHandler
	RecognitionInferencePoker *inputApplicationRecognitionIrInference.PokerHandler
	RecognitionInferenceDisk  *inputApplicationRecognitionIrInference.DiskHandler
}

func InitRecognitionContainer(oContext context.Context) (*RecognitionContainer, error) {
	oDieUsecase, err := newDieUsecase(oContext)
	if err != nil {
		return nil, err
	}

	return &RecognitionContainer{
		RecognitionInferenceDie:   inputApplicationRecognitionIrInference.NewDieHandler(oDieUsecase),
		RecognitionInferencePoker: inputApplicationRecognitionIrInference.NewPokerHandler(),
		RecognitionInferenceDisk:  inputApplicationRecognitionIrInference.NewDiskHandler(),
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

func newDieUsecase(oContext context.Context) (usecasePortAnyIrInference.DieUsecase, error) {
	oDiePipeline, err := newDiePipeline(oContext)
	if err != nil {
		return nil, err
	}

	return usecaseApplicationAnyIrInference.NewDieUsecase(oDiePipeline), nil
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

func newPokerPipeline(oContext context.Context) (outputPortAnyPipeline.PokerPipeline, error) {
	oAbstractOnnx, err := outputApplicationOnnx.NewAbstractOnnx(oContext, bootstrap.CONFIG.ONNX.LIBRARY)
	if err != nil {
		return nil, err
	}

	oPokerCardDetector, err := detector.NewPokerCardDetector(oAbstractOnnx)
	if err != nil {
		return nil, fmt.Errorf("init poker card detector: %w", err)
	}
	oPokerFaceClassifier, err := classifier.NewPokerFaceClassifier(oAbstractOnnx)
	if err != nil {
		return nil, fmt.Errorf("init poker face classifier: %w", err)
	}
	oPokerRankClassifier, err := classifier.NewPokerRankClassifier(oAbstractOnnx)
	if err != nil {
		return nil, fmt.Errorf("init poker rank classifier: %w", err)
	}
	oPokerSuitClassifier, err := classifier.NewPokerSuitClassifier(oAbstractOnnx)
	if err != nil {
		return nil, fmt.Errorf("init poker suit classifier: %w", err)
	}

	return outputApplicationOnnxPipeline.NewPokerPipeline(oPokerCardDetector, oPokerFaceClassifier, oPokerRankClassifier, oPokerSuitClassifier), nil
}
