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

func (oSelf *PokerPredictorUsecase) Recognize(sWorkdir string, nLimit int, fnOnResult func(domain.PokerPredictionResult)) (*domain.PokerPredictionReport, error) {
	aImages, err := oSelf.ListImages(sWorkdir)
	if err != nil {
		return nil, err
	}

	if nLimit > 0 && len(aImages) > nLimit {
		aImages = aImages[:nLimit]
	}

	tStarted := time.Now()

	aResults := make([]domain.PokerPredictionResult, 0, len(aImages))
	for _, sImage := range aImages {
		aImage, err := os.ReadFile(sImage)
		if err != nil {
			return nil, fmt.Errorf("%s: 無法讀取圖片: %w", filepath.Base(sImage), err)
		}

		// 只量 pipeline 跑一次的時間（偵測加所有牌的分類），不含讀檔。
		tImageStarted := time.Now()
		oRecognition, err := oSelf.pokerPipeline.Recognize(aImage)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(sImage), err)
		}

		oResult := domain.PokerPredictionResult{Image: sImage, Pokers: oRecognition.Pokers, Elapsed: time.Since(tImageStarted), DetectElapsed: oRecognition.DetectElapsed}
		aResults = append(aResults, oResult)
		if fnOnResult != nil {
			fnOnResult(oResult)
		}
	}

	tElapsed := time.Since(tStarted)

	return &domain.PokerPredictionReport{Results: aResults, Elapsed: tElapsed}, nil
}
