package inputApplicationCommandIrPredictor

import (
	"fmt"
	"io"
	"path/filepath"

	usecasePortAnyIrPredictor "landan-desktop-fyne/internal/usecase/port/any/ir/predictor"
)

// DiePredictorHandler 辨識目錄下所有圖片中的骰子，並印出報告。
type DiePredictorHandler struct {
	irDiePredictorUsecase usecasePortAnyIrPredictor.DiePredictorUsecase
}

func NewDiePredictorHandler(oDiePredictorUsecase usecasePortAnyIrPredictor.DiePredictorUsecase) *DiePredictorHandler {
	return &DiePredictorHandler{
		irDiePredictorUsecase: oDiePredictorUsecase,
	}
}

// Handle 辨識 sWorkdir 目錄下（只讀第一層）所有圖片中的骰子。
func (oSelf *DiePredictorHandler) Handle(sWorkdir string, oWriter io.Writer) error {
	oReport, err := oSelf.irDiePredictorUsecase.Recognize(sWorkdir)
	if err != nil {
		return err
	}

	for _, oResult := range oReport.Results {
		fmt.Fprintf(oWriter, "%s：共偵測到 %d 顆骰子，執行時間 %.1f ms\n", filepath.Base(oResult.Image), len(oResult.Dies), float64(oResult.Elapsed.Microseconds())/1000)
		for iDie, oDie := range oResult.Dies {
			fmt.Fprintf(oWriter, "  #%d die (%.2f) box=(%d,%d,%d,%d)\n", iDie+1, oDie.Confidence, oDie.X, oDie.Y, oDie.X+oDie.Width, oDie.Y+oDie.Height)
		}
	}

	fmt.Fprintln(oWriter, "========== 報告 ==========")
	fmt.Fprintf(oWriter, "照片數量: %d\n", len(oReport.Results))
	fmt.Fprintf(oWriter, "總共時間: %.1f ms\n", float64(oReport.Elapsed.Microseconds())/1000)

	return nil
}
