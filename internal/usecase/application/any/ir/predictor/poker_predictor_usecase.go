package usecaseApplicationAnyIrPredictor

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	domain "landan-desktop-fyne/internal/domain"
	outputPortAnyPipeline "landan-desktop-fyne/internal/output/port/any/pipeline"
	usecasePortAnyIrPredictor "landan-desktop-fyne/internal/usecase/port/any/ir/predictor"
)

type PokerPredictorUsecase struct {
	*AbstractUsecase
	pokerPipeline outputPortAnyPipeline.PokerPipeline
}

func NewPokerPredictorUsecase(oPokerPipeline outputPortAnyPipeline.PokerPipeline) usecasePortAnyIrPredictor.PokerPredictorUsecase {
	return &PokerPredictorUsecase{
		AbstractUsecase: NewAbstractUsecase(),
		pokerPipeline:   oPokerPipeline,
	}
}

func (oSelf *PokerPredictorUsecase) Recognize(sWorkdir string) (*domain.PokerPredictionReport, error) {
	aImages, err := oSelf.ListImages(sWorkdir)
	if err != nil {
		return nil, err
	}

	tStarted := time.Now()

	aResults := make([]domain.PokerPredictionResult, 0, len(aImages))
	for _, sImage := range aImages {
		tImageStarted := time.Now()

		aImage, err := os.ReadFile(sImage)
		if err != nil {
			return nil, fmt.Errorf("%s: 無法讀取圖片: %w", filepath.Base(sImage), err)
		}

		aPokers, err := oSelf.pokerPipeline.Recognize(aImage)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(sImage), err)
		}

		aResults = append(aResults, domain.PokerPredictionResult{Image: sImage, Pokers: aPokers, Elapsed: time.Since(tImageStarted)})
	}

	tElapsed := time.Since(tStarted)

	return &domain.PokerPredictionReport{Results: aResults, Elapsed: tElapsed}, nil
}
