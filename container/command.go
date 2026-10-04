package container

import (
	"context"

	internalCommand "landan-desktop-fyne/internal/command"
)

type CommandContainer struct {
	DiePredictor *internalCommand.DiePredictorCommand
}

func InitCommandContainer(oContext context.Context) (*CommandContainer, error) {
	oDieUsecase, err := newDieUsecase(oContext)
	if err != nil {
		return nil, err
	}

	return &CommandContainer{
		DiePredictor: internalCommand.NewDiePredictorCommand(oDieUsecase),
	}, nil
}
