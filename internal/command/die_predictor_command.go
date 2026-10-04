package internalCommand

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	domain "landan-desktop-fyne/internal/domain"
	usecasePortRecognitionInference "landan-desktop-fyne/internal/usecase/port/recognition/inference"
)

// DiePredictorCommand 辨識目錄下所有圖片中的骰子，並印出報告。
type DiePredictorCommand struct {
	*AbstractCommand
	dieUsecase usecasePortRecognitionInference.DieUsecase
}

func NewDiePredictorCommand(oDieUsecase usecasePortRecognitionInference.DieUsecase) *DiePredictorCommand {
	return &DiePredictorCommand{
		AbstractCommand: NewAbstractCommand(),
		dieUsecase:      oDieUsecase,
	}
}

type diePrediction struct {
	dies    []*domain.Die
	elapsed time.Duration
	err     error
}

// Handle 辨識 sWorkdir 目錄下（只讀第一層）所有圖片中的骰子，同時最多跑 iThreads 張。
func (oSelf *DiePredictorCommand) Handle(sWorkdir string, iThreads int, oWriter io.Writer) error {
	aImages, err := oSelf.ListImages(sWorkdir)
	if err != nil {
		return err
	}

	tStarted := time.Now()

	// 結果依輸入順序放，所以輸出順序跟檔名排序一致。
	aPredictions := make([]diePrediction, len(aImages))
	oSelf.RunParallel(len(aImages), iThreads, func(iIndex int) {
		aPredictions[iIndex] = oSelf.predict(aImages[iIndex])
	})

	tElapsed := time.Since(tStarted)

	for iIndex, sImage := range aImages {
		if aPredictions[iIndex].err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(sImage), aPredictions[iIndex].err)
		}
	}

	for iIndex, sImage := range aImages {
		fmt.Fprintf(oWriter, "%s 執行時間 %.1f ms\n", filepath.Base(sImage), float64(aPredictions[iIndex].elapsed.Microseconds())/1000)
	}
	for iIndex, sImage := range aImages {
		aDies := aPredictions[iIndex].dies
		fmt.Fprintf(oWriter, "%s：共偵測到 %d 顆骰子\n", filepath.Base(sImage), len(aDies))
		for iDie, oDie := range aDies {
			fmt.Fprintf(oWriter, "  #%d die (%.2f) box=(%d,%d,%d,%d)\n", iDie+1, oDie.Confidence, oDie.X, oDie.Y, oDie.X+oDie.Width, oDie.Y+oDie.Height)
		}
	}

	fmt.Fprintln(oWriter, "========== 報告 ==========")
	fmt.Fprintf(oWriter, "線程數量: %d\n", iThreads)
	fmt.Fprintf(oWriter, "照片數量: %d\n", len(aImages))
	fmt.Fprintf(oWriter, "總共時間: %.1f ms\n", float64(tElapsed.Microseconds())/1000)

	return nil
}

func (oSelf *DiePredictorCommand) predict(sImage string) diePrediction {
	tStarted := time.Now()

	aImage, err := os.ReadFile(sImage)
	if err != nil {
		return diePrediction{err: fmt.Errorf("無法讀取圖片: %w", err)}
	}

	aDies, err := oSelf.dieUsecase.Recognize(aImage)
	if err != nil {
		return diePrediction{err: err}
	}

	return diePrediction{dies: aDies, elapsed: time.Since(tStarted)}
}
