package inputApplicationCommandIrPredictor

import (
	"fmt"
	"io"
	"path/filepath"

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
		fmt.Fprintf(oWriter, "%s：共偵測到 %d 張撲克牌，執行時間 %.1f ms\n", filepath.Base(oResult.Image), len(oResult.Pokers), float64(oResult.Elapsed.Microseconds())/1000)
		for iPoker, oPoker := range oResult.Pokers {
			fmt.Fprintf(oWriter, "  #%d poker (%.2f) box=(%d,%d,%d,%d) %s (%.2f)", iPoker+1, oPoker.Confidence, oPoker.X, oPoker.Y, oPoker.X+oPoker.Width, oPoker.Y+oPoker.Height, oPoker.Face.Name, oPoker.Face.Confidence)
			if oPoker.Suit != nil && oPoker.Rank != nil {
				fmt.Fprintf(oWriter, " %s %s (%.2f, %.2f)", oPoker.Suit.Name, oPoker.Rank.Name, oPoker.Suit.Confidence, oPoker.Rank.Confidence)
			}
			fmt.Fprintln(oWriter)
		}
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(oWriter, "========== 報告 ==========")
	fmt.Fprintf(oWriter, "照片數量: %d\n", len(oReport.Results))
	fmt.Fprintf(oWriter, "總共時間: %.1f ms\n", float64(oReport.Elapsed.Microseconds())/1000)

	return nil
}
