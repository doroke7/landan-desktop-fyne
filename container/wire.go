//go:build wireinject
// +build wireinject

package container

import (
	"context"

	"github.com/google/wire"

	"landan-desktop-fyne/bootstrap"
	"landan-desktop-fyne/internal/classifier"
	"landan-desktop-fyne/internal/detector"

	inputApplicationCommandIrPredictor "landan-desktop-fyne/internal/input/application/command/ir/predictor"
	inputApplicationRecognitionIrInference "landan-desktop-fyne/internal/input/application/recognition/ir/inference"

	outputApplicationOnnx "landan-desktop-fyne/internal/output/application/onnx"
	outputApplicationOnnxPipeline "landan-desktop-fyne/internal/output/application/onnx/pipeline"

	usecaseApplicationAnyIrInference "landan-desktop-fyne/internal/usecase/application/any/ir/inference"
	usecaseApplicationAnyIrPredictor "landan-desktop-fyne/internal/usecase/application/any/ir/predictor"
)

//////////////////////////////////////////////////////////////////////////////

// RecognitionContainer 只給 `recognition` gRPC 服務使用。
// 目前只有 die 會載入模型，poker、disk 的 handler 還沒有相依。
type RecognitionContainer struct {

	// recognition
	RecognitionInferenceDie   *inputApplicationRecognitionIrInference.DieHandler
	RecognitionInferencePoker *inputApplicationRecognitionIrInference.PokerHandler
	RecognitionInferenceDisk  *inputApplicationRecognitionIrInference.DiskHandler
}

func InitRecognitionContainer(ctx context.Context, config bootstrap.Config) (*RecognitionContainer, error) {
	wire.Build(

		// onnx：onnxruntime 或 OpenVINO 由 config/openvino.yaml 的 enabled 決定
		outputApplicationOnnx.NewAbstractOnnx,

		// detector
		detector.NewDieTopDetector,

		// classifier
		classifier.NewDieValueClassifier,

		// output
		outputApplicationOnnxPipeline.NewDiePipeline,

		// usecase
		usecaseApplicationAnyIrInference.NewDieUsecase,

		// recognition
		inputApplicationRecognitionIrInference.NewDieHandler,
		inputApplicationRecognitionIrInference.NewPokerHandler,
		inputApplicationRecognitionIrInference.NewDiskHandler,

		wire.Struct(new(RecognitionContainer), "*"),
	)
	return nil, nil
}

//////////////////////////////////////////////////////////////////////////////

// DiePredictorCommandContainer 只給 `die-predictor` 指令使用，不載入 poker 的模型。
type DiePredictorCommandContainer struct {

	// command
	DiePredictor *inputApplicationCommandIrPredictor.DiePredictorHandler
}

func InitDiePredictorCommandContainer(ctx context.Context, config bootstrap.Config) (*DiePredictorCommandContainer, error) {
	wire.Build(

		// onnx
		outputApplicationOnnx.NewAbstractOnnx,

		// detector
		detector.NewDieTopDetector,

		// classifier
		classifier.NewDieValueClassifier,

		// output
		outputApplicationOnnxPipeline.NewDiePipeline,

		// usecase
		usecaseApplicationAnyIrPredictor.NewDiePredictorUsecase,

		// command
		inputApplicationCommandIrPredictor.NewDiePredictorHandler,

		wire.Struct(new(DiePredictorCommandContainer), "*"),
	)
	return nil, nil
}

//////////////////////////////////////////////////////////////////////////////

// PokerPredictorCommandContainer 只給 `poker-predictor` 指令使用，不載入 die 的模型。
type PokerPredictorCommandContainer struct {

	// command
	PokerPredictor *inputApplicationCommandIrPredictor.PokerPredictorHandler
}

func InitPokerPredictorCommandContainer(ctx context.Context, config bootstrap.Config) (*PokerPredictorCommandContainer, error) {
	wire.Build(

		// onnx
		outputApplicationOnnx.NewAbstractOnnx,

		// detector
		detector.NewPokerCardDetector,

		// classifier
		classifier.NewPokerFaceClassifier,
		classifier.NewPokerRankClassifier,
		classifier.NewPokerSuitClassifier,

		// output
		outputApplicationOnnxPipeline.NewPokerPipeline,

		// usecase
		usecaseApplicationAnyIrPredictor.NewPokerPredictorUsecase,

		// command
		inputApplicationCommandIrPredictor.NewPokerPredictorHandler,

		wire.Struct(new(PokerPredictorCommandContainer), "*"),
	)
	return nil, nil
}
