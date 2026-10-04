package container

import (
	"context"

	inputApplicationCommand "landan-desktop-fyne/internal/input/application/command"
	usecaseApplicationCommand "landan-desktop-fyne/internal/usecase/application/command"
)

type CommandContainer struct {
	DiePredictor *inputApplicationCommand.DiePredictorHandler
}

func InitCommandContainer(oContext context.Context) (*CommandContainer, error) {
	oDiePipeline, err := newDiePipeline(oContext)
	if err != nil {
		return nil, err
	}

	return &CommandContainer{
		DiePredictor: inputApplicationCommand.NewDiePredictorHandler(usecaseApplicationCommand.NewDiePredictorUsecase(oDiePipeline)),
	}, nil
}
