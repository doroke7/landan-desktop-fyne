package usecasePortAnyIrPredictor

import (
	domain "landan-desktop-fyne/internal/domain"
)

type PokerPredictorUsecase interface {
	// Recognize 辨識 sWorkdir 目錄下（只讀第一層）所有圖片中的撲克牌。
	// nLimit 大於 0 時只辨識檔名排序後的前 nLimit 張；0 代表全部。
	// fnOnResult 不為 nil 時，每辨識完一張圖就呼叫一次，讓呼叫端可以邊跑邊顯示進度。
	Recognize(sWorkdir string, nLimit int, fnOnResult func(domain.PokerPredictionResult)) (*domain.PokerPredictionReport, error)
}
