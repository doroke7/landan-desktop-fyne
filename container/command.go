package container

import (
	"context"

	inputApplicationCommandIrPredictor "landan-desktop-fyne/internal/input/application/command/ir/predictor"
	usecaseApplicationAnyIrPredictor "landan-desktop-fyne/internal/usecase/application/any/ir/predictor"
)

type CommandContainer struct {
	DiePredictor   *inputApplicationCommandIrPredictor.DiePredictorHandler
	PokerPredictor *inputApplicationCommandIrPredictor.PokerPredictorHandler
}

// InitDiePredictorCommandContainer 只建 die-predictor 要用的東西，不載入 poker 的模型。
func InitDiePredictorCommandContainer(oContext context.Context) (*CommandContainer, error) {
	oDiePipeline, err := newDiePipeline(oContext)
	if err != nil {
		return nil, err
	}

	return &CommandContainer{
		DiePredictor: inputApplicationCommandIrPredictor.NewDiePredictorHandler(usecaseApplicationAnyIrPredictor.NewDiePredictorUsecase(oDiePipeline)),
	}, nil
}

// InitPokerPredictorCommandContainer 只建 poker-predictor 要用的東西，不載入 die 的模型。
func InitPokerPredictorCommandContainer(oContext context.Context) (*CommandContainer, error) {
	oPokerPipeline, err := newPokerPipeline(oContext)
	if err != nil {
		return nil, err
	}

	return &CommandContainer{
		PokerPredictor: inputApplicationCommandIrPredictor.NewPokerPredictorHandler(usecaseApplicationAnyIrPredictor.NewPokerPredictorUsecase(oPokerPipeline)),
	}, nil
}
