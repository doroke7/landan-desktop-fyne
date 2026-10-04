//go:build !openvino && !darwin

// Package openvino 在 macOS 以外、沒加 -tags openvino 時是空殼：載入會直接報錯，不影響沒裝 OpenVINO 的機器編譯。
package openvino

import "errors"

type Model struct{}

var errNotBuilt = errors.New("這個執行檔沒有編進 OpenVINO；請先安裝 OpenVINO（macOS: brew install openvino），並用 -tags openvino 編譯（macOS 預設就會編進來），例如 go run -tags openvino main.go ...")

func Load(sXmlPath string, sDevice string) (*Model, error) {
	return nil, errNotBuilt
}

func (oSelf *Model) InputSize() (int, int) {
	return 0, 0
}

func (oSelf *Model) Run(aInput []float32) ([]float32, []int64, error) {
	return nil, nil, errNotBuilt
}

func (oSelf *Model) Close() error {
	return nil
}
