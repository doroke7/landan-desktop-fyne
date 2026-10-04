package detector

import (
	"fmt"

	bootstrap "landan-desktop-fyne/bootstrap"
	domain "landan-desktop-fyne/internal/domain"
	outputApplicationOnnx "landan-desktop-fyne/internal/output/application/onnx"
)

// DieTopDetector 用 onnx 跑 die 的偵測模型，onnx 的細節都在 AbstractDetector。
type DieTopDetector struct {
	*AbstractDetector
	threshold float32
}

// NewDieTopDetector 從 config/onnx.yaml 的 detect.die.top 讀模型路徑與信心門檻。
func NewDieTopDetector(oAbstractOnnx *outputApplicationOnnx.AbstractOnnx) (*DieTopDetector, error) {
	if bootstrap.CONFIG.ONNX.DETECT.DIE.TOP.PATH == "" {
		return nil, fmt.Errorf("onnx.detect.die.top.path is empty (is config/onnx.yaml filled in? run from the project root)")
	}

	oAbstractDetector, err := NewAbstractDetector(oAbstractOnnx, bootstrap.CONFIG.ONNX.DETECT.DIE.TOP.PATH, bootstrap.CONFIG.OPENVINO.DETECT.DIE.TOP.PATH)
	if err != nil {
		return nil, err
	}

	return &DieTopDetector{AbstractDetector: oAbstractDetector, threshold: bootstrap.CONFIG.ONNX.DETECT.DIE.TOP.THRESHOLD}, nil
}

func (oSelf *DieTopDetector) Recognize(aImage []byte) ([]*domain.Die, error) {
	aDetections, err := oSelf.AbstractDetector.Recognize(aImage, oSelf.threshold)
	if err != nil {
		return nil, err
	}

	var aDies []*domain.Die
	for _, oDetection := range aDetections {
		aDies = append(aDies, &domain.Die{
			X:          oDetection.X,
			Y:          oDetection.Y,
			Width:      oDetection.Width,
			Height:     oDetection.Height,
			Confidence: oDetection.Confidence,
		})
	}

	return aDies, nil
}

// main desktop compose up supervisor
// main desktop compose up
// main desktop
