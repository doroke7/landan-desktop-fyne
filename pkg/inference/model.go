// Package inference 定義偵測、分類模型共用的推論介面，讓上層不用知道底下是 onnxruntime 還是 OpenVINO。
package inference

// Model 是一個載入好、輸入尺寸固定的模型。
type Model interface {
	// InputSize 是模型輸入 [1, 3, H, W] 的高與寬。
	InputSize() (iHeight, iWidth int)
	// Run 輸入 RGB 三個平面（NCHW，0~1）的資料，回傳第一個輸出的資料與形狀。回傳的資料已複製出來，呼叫端可以留著。
	Run(aInput []float32) ([]float32, []int64, error)
	// Close 釋放模型。
	Close() error
}
