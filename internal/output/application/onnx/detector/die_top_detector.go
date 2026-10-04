package outputApplicationOnnxDetector

import (
	"fmt"

	bootstrap "landan-desktop-fyne/bootstrap"
	domain "landan-desktop-fyne/internal/domain"
	outputApplicationOnnx "landan-desktop-fyne/internal/output/application/onnx"
	outputPortAnyDetector "landan-desktop-fyne/internal/output/port/any/detector"
)

// DieTopDetectorModel 用 onnx 跑 die 的偵測模型，onnx 的細節都在 AbstractDetector。
type DieTopDetectorModel struct {
	*AbstractDetector
	threshold float32
}

// NewDieTopDetectorModel 從 config/onnx.yaml 的 detect.die.top 讀模型路徑與信心門檻。
func NewDieTopDetectorModel(oAbstractOnnx *outputApplicationOnnx.AbstractOnnx) (outputPortAnyDetector.DieTopDetectorModel, error) {
	oModelConfig := bootstrap.CONFIG.ONNX.DETECT.DIE.TOP
	sModelPath := oModelConfig.PATH
	if sModelPath == "" {
		return nil, fmt.Errorf("onnx.detect.die.top.path is empty (is config/onnx.yaml filled in? run from the project root)")
	}

	oAbstractDetector, err := NewAbstractDetector(oAbstractOnnx, sModelPath)
	if err != nil {
		return nil, err
	}

	return &DieTopDetectorModel{AbstractDetector: oAbstractDetector, threshold: oModelConfig.THRESHOLD}, nil
}

func (oSelf *DieTopDetectorModel) Recognize(aImage []byte) ([]*domain.Die, error) {
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
