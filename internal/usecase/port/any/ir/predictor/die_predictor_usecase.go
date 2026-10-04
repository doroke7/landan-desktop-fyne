package usecasePortAnyIrPredictor

import (
	domain "landan-desktop-fyne/internal/domain"
)

type DiePredictorUsecase interface {
	// Recognize 辨識 sWorkdir 目錄下（只讀第一層）所有圖片中的骰子。
	Recognize(sWorkdir string) (*domain.DiePredictionReport, error)
}
