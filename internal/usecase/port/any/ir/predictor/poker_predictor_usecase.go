package usecasePortAnyIrPredictor

import (
	domain "landan-desktop-fyne/internal/domain"
)

type PokerPredictorUsecase interface {
	// Recognize 辨識 sWorkdir 目錄下（只讀第一層）所有圖片中的撲克牌。
	Recognize(sWorkdir string) (*domain.PokerPredictionReport, error)
}
