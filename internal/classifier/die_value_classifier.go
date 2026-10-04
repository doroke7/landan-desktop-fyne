package classifier

import (
	"fmt"

	bootstrap "landan-desktop-fyne/bootstrap"
	domain "landan-desktop-fyne/internal/domain"
	outputApplicationOnnx "landan-desktop-fyne/internal/output/application/onnx"
)

// 骰子點數的類別數，順序是 1 ~ 6（見 pk-studio-ir-model 的 cfg/classify/die/value-data.yaml）。
const nDieValueClasses = 6

// DieValueClassifier 用 onnx 跑 die 點數的分類模型，onnx 的細節都在 AbstractClassifier。
// 輸出 [1, 6]，各點數的機率，index 0 對應點數 1。
type DieValueClassifier struct {
	*AbstractClassifier
}

// NewDieValueClassifier 從 config/onnx.yaml 的 classify.die.value 讀模型路徑。
func NewDieValueClassifier(oAbstractOnnx *outputApplicationOnnx.AbstractOnnx) (*DieValueClassifier, error) {
	if bootstrap.CONFIG.ONNX.CLASSIFY.DIE.VALUE.PATH == "" {
		return nil, fmt.Errorf("onnx.classify.die.value.path is empty (is config/onnx.yaml filled in? run from the project root)")
	}

	oAbstractClassifier, err := NewAbstractClassifier(oAbstractOnnx, bootstrap.CONFIG.ONNX.CLASSIFY.DIE.VALUE.PATH)
	if err != nil {
		return nil, err
	}

	return &DieValueClassifier{AbstractClassifier: oAbstractClassifier}, nil
}

func (oSelf *DieValueClassifier) Classify(aImage []byte) (*domain.DieValue, error) {
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
