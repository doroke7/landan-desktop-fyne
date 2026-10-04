package inputApplicationCommandIrPredictor

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	domain "landan-desktop-fyne/internal/domain"
	usecasePortAnyIrPredictor "landan-desktop-fyne/internal/usecase/port/any/ir/predictor"
)

// PokerPredictorHandler 辨識目錄下所有圖片中的撲克牌，並印出報告。
type PokerPredictorHandler struct {
	irPokerPredictorUsecase usecasePortAnyIrPredictor.PokerPredictorUsecase
}

func NewPokerPredictorHandler(oPokerPredictorUsecase usecasePortAnyIrPredictor.PokerPredictorUsecase) *PokerPredictorHandler {
	return &PokerPredictorHandler{
		irPokerPredictorUsecase: oPokerPredictorUsecase,
	}
}

// Handle 辨識 sWorkdir 目錄下（只讀第一層）所有圖片中的撲克牌；nLimit 大於 0 時只辨識前 nLimit 張。
func (oSelf *PokerPredictorHandler) Handle(sWorkdir string, nLimit int, oWriter io.Writer) error {
	oReport, err := oSelf.irPokerPredictorUsecase.Recognize(sWorkdir, nLimit, func(oResult domain.PokerPredictionResult) {
		fmt.Fprintf(oWriter, "%s  pipeline %.1f ms\n", filepath.Base(oResult.Image), milliseconds(oResult.Elapsed))
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(oWriter, "========== 報告 ==========")
	fmt.Fprintf(oWriter, "照片數量: %d\n", len(oReport.Results))
	fmt.Fprintf(oWriter, "總共時間: %.1f ms\n", float64(oReport.Elapsed.Microseconds())/1000)

	return nil
}

func milliseconds(oDuration time.Duration) float64 {
	return float64(oDuration.Microseconds()) / 1000
}
