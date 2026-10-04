package outputApplicationOnnxDetector

import (
	"fmt"

	bootstrap "landan-desktop-fyne/bootstrap"
	domain "landan-desktop-fyne/internal/domain"
	outputApplicationOnnx "landan-desktop-fyne/internal/output/application/onnx"
	outputPortAnyDetector "landan-desktop-fyne/internal/output/port/any/detector"
)

// 信心低於這個值的框直接丟掉。
const fDieConfidenceThreshold = 0.25

// DieTopDetectorModel 用 onnx 跑 die 的偵測模型，onnx 的細節都在 AbstractDetector。
type DieTopDetectorModel struct {
	*AbstractDetector
}

// NewDieTopDetectorModel 從 config/onnx.yaml 的 detect.die.top 讀模型路徑。
func NewDieTopDetectorModel(oAbstractOnnx *outputApplicationOnnx.AbstractOnnx) (outputPortAnyDetector.DieTopDetectorModel, error) {
	sModelPath := bootstrap.CONFIG.ONNX.DETECT.DIE.TOP
	if sModelPath == "" {
		return nil, fmt.Errorf("onnx.detect.die.top is empty (is config/onnx.yaml filled in? run from the project root)")
	}

	oAbstractDetector, err := NewAbstractDetector(oAbstractOnnx, sModelPath)
	if err != nil {
		return nil, err
	}

	return &DieTopDetectorModel{AbstractDetector: oAbstractDetector}, nil
}

func (oSelf *DieTopDetectorModel) Recognize(aImage []byte) ([]*domain.Die, error) {
	aDetections, err := oSelf.AbstractDetector.Recognize(aImage, fDieConfidenceThreshold)
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
