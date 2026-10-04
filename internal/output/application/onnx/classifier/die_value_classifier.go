package outputApplicationOnnxClassifier

import (
	"fmt"

	bootstrap "landan-desktop-fyne/bootstrap"
	domain "landan-desktop-fyne/internal/domain"
	outputApplicationOnnx "landan-desktop-fyne/internal/output/application/onnx"
	outputPortAnyClassifier "landan-desktop-fyne/internal/output/port/any/classifier"
)

// 骰子點數的類別數，順序是 1 ~ 6（見 pk-studio-ir-model 的 cfg/classify/die/value-data.yaml）。
const nDieValueClasses = 6

// DieValueClassifierModel 用 onnx 跑 die 點數的分類模型，onnx 的細節都在 AbstractClassifier。
// 輸出 [1, 6]，各點數的機率，index 0 對應點數 1。
type DieValueClassifierModel struct {
	*AbstractClassifier
}

// NewDieValueClassifierModel 從 config/onnx.yaml 的 classify.die.value 讀模型路徑。
func NewDieValueClassifierModel(oAbstractOnnx *outputApplicationOnnx.AbstractOnnx) (outputPortAnyClassifier.DieValueClassifierModel, error) {
	sModelPath := bootstrap.CONFIG.ONNX.CLASSIFY.DIE.VALUE
	if sModelPath == "" {
		return nil, fmt.Errorf("onnx.classify.die.value is empty (is config/onnx.yaml filled in? run from the project root)")
	}

	oAbstractClassifier, err := NewAbstractClassifier(oAbstractOnnx, sModelPath)
	if err != nil {
		return nil, err
	}

	return &DieValueClassifierModel{AbstractClassifier: oAbstractClassifier}, nil
}

func (oSelf *DieValueClassifierModel) Classify(aImage []byte) (*domain.DieValue, error) {
	aProbabilities, err := oSelf.AbstractClassifier.Recognize(aImage)
	if err != nil {
		return nil, err
	}
	if len(aProbabilities) != nDieValueClasses {
		return nil, fmt.Errorf("unsupported output size %d, want [1, %d]", len(aProbabilities), nDieValueClasses)
	}

	iBest := 0
	for iIndex, fProbability := range aProbabilities {
		if fProbability > aProbabilities[iBest] {
			iBest = iIndex
		}
	}

	return &domain.DieValue{
		Value:      iBest + 1,
		Confidence: aProbabilities[iBest],
	}, nil
}
