package outputApplicationOnnxPipeline

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"time"

	classifier "landan-desktop-fyne/internal/classifier"
	detector "landan-desktop-fyne/internal/detector"
	domain "landan-desktop-fyne/internal/domain"
	outputPortAnyPipeline "landan-desktop-fyne/internal/output/port/any/pipeline"
)

// pokerFaceFront 是 PokerFaceClassifier 認為牌面朝上時回的名稱。
const pokerFaceFront = "Front"

// PokerPipeline 先用偵測模型找出每張牌，再把每張裁下來分類是哪一面朝上；
// 只有朝上（Front）的牌才會再讀花色與點數，Back、Flow 不做事。
type PokerPipeline struct {
	pokerCardDetector   *detector.PokerCardDetector
	pokerFaceClassifier *classifier.PokerFaceClassifier
	pokerRankClassifier *classifier.PokerRankClassifier
	pokerSuitClassifier *classifier.PokerSuitClassifier
}

func NewPokerPipeline(
	oPokerCardDetector *detector.PokerCardDetector,
	oPokerFaceClassifier *classifier.PokerFaceClassifier,
	oPokerRankClassifier *classifier.PokerRankClassifier,
	oPokerSuitClassifier *classifier.PokerSuitClassifier,
) outputPortAnyPipeline.PokerPipeline {
	return &PokerPipeline{
		pokerCardDetector:   oPokerCardDetector,
		pokerFaceClassifier: oPokerFaceClassifier,
		pokerRankClassifier: oPokerRankClassifier,
		pokerSuitClassifier: oPokerSuitClassifier,
	}
}

func (oSelf *PokerPipeline) Recognize(aImage []byte) (*domain.PokerRecognition, error) {
	tDetectStarted := time.Now()
	aPokers, err := oSelf.pokerCardDetector.Recognize(aImage)
	if err != nil {
		return nil, fmt.Errorf("detect pokers: %w", err)
	}
	oRecognition := &domain.PokerRecognition{Pokers: aPokers, DetectElapsed: time.Since(tDetectStarted)}
	if len(aPokers) == 0 {
		return oRecognition, nil
	}

	oSource, _, err := image.Decode(bytes.NewReader(aImage))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	for iPoker, oPoker := range aPokers {
		tPokerStarted := time.Now()

		aCrop, err := crop(oSource, oPoker.X, oPoker.Y, oPoker.Width, oPoker.Height)
		if err != nil {
			return nil, fmt.Errorf("crop poker #%d: %w", iPoker+1, err)
		}

		oPoker.Face, err = oSelf.pokerFaceClassifier.Classify(aCrop)
		if err != nil {
			return nil, fmt.Errorf("classify poker #%d face: %w", iPoker+1, err)
		}
		if oPoker.Face.Name != pokerFaceFront {
			oPoker.Elapsed = time.Since(tPokerStarted)
			continue
		}

		oPoker.Suit, err = oSelf.pokerSuitClassifier.Classify(aCrop)
		if err != nil {
			return nil, fmt.Errorf("classify poker #%d suit: %w", iPoker+1, err)
		}
		oPoker.Rank, err = oSelf.pokerRankClassifier.Classify(aCrop)
		if err != nil {
			return nil, fmt.Errorf("classify poker #%d rank: %w", iPoker+1, err)
		}
		oPoker.Elapsed = time.Since(tPokerStarted)
	}

	return oRecognition, nil
}
