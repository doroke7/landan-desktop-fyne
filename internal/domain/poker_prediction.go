package domain

import (
	"time"
)

// PokerPredictionResult 是單張圖片的辨識結果。
type PokerPredictionResult struct {
	Image   string
	Pokers  []*Poker
	Elapsed time.Duration
}

// PokerPredictionReport 是整個目錄的辨識結果，Results 依檔名排序。
type PokerPredictionReport struct {
	Results []PokerPredictionResult
	Elapsed time.Duration
}
