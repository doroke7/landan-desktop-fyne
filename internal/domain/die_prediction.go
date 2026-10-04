package domain

import (
	"time"
)

// DiePredictionResult 是單張圖片的辨識結果。
type DiePredictionResult struct {
	Image   string
	Dies    []*Die
	Elapsed time.Duration
}

// DiePredictionReport 是整個目錄的辨識結果，Results 依檔名排序。
type DiePredictionReport struct {
	Results []DiePredictionResult
	Elapsed time.Duration
}
