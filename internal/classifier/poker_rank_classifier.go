package classifier

import (
	"fmt"

	bootstrap "landan-desktop-fyne/bootstrap"
	domain "landan-desktop-fyne/internal/domain"
	outputApplicationInference "landan-desktop-fyne/internal/output/application/inference"
)

// 撲克牌的點數，順序見 pk-studio-ir-model 的 cfg/classify/poker/rank-data.yaml。
var aPokerRankNames = []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}

// PokerRankClassifier 用 onnx 跑 撲克牌的點數，onnx 的細節都在 AbstractClassifier。
type PokerRankClassifier struct {
	*AbstractClassifier
}

// NewPokerRankClassifier 從 config/onnx.yaml 的 classify.poker.rank 讀模型路徑。
func NewPokerRankClassifier(oAbstractInference *outputApplicationInference.AbstractInference) (*PokerRankClassifier, error) {
	if bootstrap.CONFIG.ONNX.CLASSIFY.POKER.RANK.PATH == "" {
		return nil, fmt.Errorf("onnx.classify.poker.rank.path is empty (is config/onnx.yaml filled in? run from the project root)")
	}

	oAbstractClassifier, err := NewAbstractClassifier(oAbstractInference, bootstrap.CONFIG.ONNX.CLASSIFY.POKER.RANK.PATH, bootstrap.CONFIG.OPENVINO.CLASSIFY.POKER.RANK.PATH)
	if err != nil {
		return nil, err
	}

	return &PokerRankClassifier{AbstractClassifier: oAbstractClassifier}, nil
}

func (oSelf *PokerRankClassifier) Classify(aImage []byte) (*domain.PokerRank, error) {
	sName, fConfidence, err := oSelf.AbstractClassifier.Best(aImage, aPokerRankNames)
	if err != nil {
		return nil, err
	}

	return &domain.PokerRank{Name: sName, Confidence: fConfidence}, nil
}
