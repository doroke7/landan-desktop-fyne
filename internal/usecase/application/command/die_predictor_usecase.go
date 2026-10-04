package usecaseApplicationCommand

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	domain "landan-desktop-fyne/internal/domain"
	outputPortAnyPipeline "landan-desktop-fyne/internal/output/port/any/pipeline"
	usecasePortCommand "landan-desktop-fyne/internal/usecase/port/command"
)

type DiePredictorUsecase struct {
	*AbstractUsecase
	diePipeline outputPortAnyPipeline.DiePipeline
}

func NewDiePredictorUsecase(oDiePipeline outputPortAnyPipeline.DiePipeline) usecasePortCommand.DiePredictorUsecase {
	return &DiePredictorUsecase{
		AbstractUsecase: NewAbstractUsecase(),
		diePipeline:     oDiePipeline,
	}
}

func (oSelf *DiePredictorUsecase) Recognize(sWorkdir string) (*domain.DiePredictionReport, error) {
	aImages, err := oSelf.ListImages(sWorkdir)
	if err != nil {
		return nil, err
	}

	tStarted := time.Now()

	aResults := make([]domain.DiePredictionResult, 0, len(aImages))
	for _, sImage := range aImages {
		tImageStarted := time.Now()

		aImage, err := os.ReadFile(sImage)
		if err != nil {
			return nil, fmt.Errorf("%s: 無法讀取圖片: %w", filepath.Base(sImage), err)
		}

		aDies, err := oSelf.diePipeline.Recognize(aImage)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(sImage), err)
		}

		aResults = append(aResults, domain.DiePredictionResult{Image: sImage, Dies: aDies, Elapsed: time.Since(tImageStarted)})
	}

	tElapsed := time.Since(tStarted)

	return &domain.DiePredictionReport{Results: aResults, Elapsed: tElapsed}, nil
}
