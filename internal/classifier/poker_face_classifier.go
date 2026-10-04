package classifier

import (
	"fmt"

	bootstrap "landan-desktop-fyne/bootstrap"
	domain "landan-desktop-fyne/internal/domain"
	outputApplicationOnnx "landan-desktop-fyne/internal/output/application/onnx"
)

// 撲克牌的面，順序見 pk-studio-ir-model 的 cfg/classify/poker/card-data.yaml。
var aPokerFaceNames = []string{"Front", "Flow", "Back"}

// PokerFaceClassifier 用 onnx 跑 撲克牌哪一面朝上（Front / Flow / Back），onnx 的細節都在 AbstractClassifier。
type PokerFaceClassifier struct {
	*AbstractClassifier
}

// NewPokerFaceClassifier 從 config/onnx.yaml 的 classify.poker.card 讀模型路徑。
func NewPokerFaceClassifier(oAbstractOnnx *outputApplicationOnnx.AbstractOnnx) (*PokerFaceClassifier, error) {
	if bootstrap.CONFIG.ONNX.CLASSIFY.POKER.CARD.PATH == "" {
		return nil, fmt.Errorf("onnx.classify.poker.card.path is empty (is config/onnx.yaml filled in? run from the project root)")
	}

	oAbstractClassifier, err := NewAbstractClassifier(oAbstractOnnx, bootstrap.CONFIG.ONNX.CLASSIFY.POKER.CARD.PATH, bootstrap.CONFIG.OPENVINO.CLASSIFY.POKER.CARD.PATH)
	if err != nil {
		return nil, err
	}

	return &PokerFaceClassifier{AbstractClassifier: oAbstractClassifier}, nil
}

func (oSelf *PokerFaceClassifier) Classify(aImage []byte) (*domain.PokerFace, error) {
	sName, fConfidence, err := oSelf.AbstractClassifier.Best(aImage, aPokerFaceNames)
	if err != nil {
		return nil, err
	}

	return &domain.PokerFace{Name: sName, Confidence: fConfidence}, nil
}
