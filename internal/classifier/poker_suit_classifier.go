package classifier

import (
	"fmt"

	bootstrap "landan-desktop-fyne/bootstrap"
	domain "landan-desktop-fyne/internal/domain"
	outputApplicationOnnx "landan-desktop-fyne/internal/output/application/onnx"
)

// 撲克牌的花色，順序見 pk-studio-ir-model 的 cfg/classify/poker/suit-data.yaml。
var aPokerSuitNames = []string{"Spade", "Heart", "Diamond", "Club"}

// PokerSuitClassifier 用 onnx 跑 撲克牌的花色，onnx 的細節都在 AbstractClassifier。
type PokerSuitClassifier struct {
	*AbstractClassifier
}

// NewPokerSuitClassifier 從 config/onnx.yaml 的 classify.poker.suit 讀模型路徑。
func NewPokerSuitClassifier(oAbstractOnnx *outputApplicationOnnx.AbstractOnnx) (*PokerSuitClassifier, error) {
	if bootstrap.CONFIG.ONNX.CLASSIFY.POKER.SUIT.PATH == "" {
		return nil, fmt.Errorf("onnx.classify.poker.suit.path is empty (is config/onnx.yaml filled in? run from the project root)")
	}

	oAbstractClassifier, err := NewAbstractClassifier(oAbstractOnnx, bootstrap.CONFIG.ONNX.CLASSIFY.POKER.SUIT.PATH, bootstrap.CONFIG.OPENVINO.CLASSIFY.POKER.SUIT.PATH)
	if err != nil {
		return nil, err
	}

	return &PokerSuitClassifier{AbstractClassifier: oAbstractClassifier}, nil
}

func (oSelf *PokerSuitClassifier) Classify(aImage []byte) (*domain.PokerSuit, error) {
	sName, fConfidence, err := oSelf.AbstractClassifier.Best(aImage, aPokerSuitNames)
	if err != nil {
		return nil, err
	}

	return &domain.PokerSuit{Name: sName, Confidence: fConfidence}, nil
}
