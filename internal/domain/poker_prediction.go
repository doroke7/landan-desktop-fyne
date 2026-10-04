package domain

import (
	"time"
)

// PokerPredictionResult 是單張圖片的辨識結果。
type PokerPredictionResult struct {
	Image   string
	Pokers  []*Poker
	Elapsed time.Duration
	// DetectElapsed 是這張圖偵測牌框用掉的時間，Elapsed 扣掉它和讀圖就是各張牌分類的時間。
	DetectElapsed time.Duration
}

// PokerPredictionReport 是整個目錄的辨識結果，Results 依檔名排序。
type PokerPredictionReport struct {
	Results []PokerPredictionResult
	Elapsed time.Duration
}
