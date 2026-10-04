package outputApplicationOnnxPipeline

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	classifier "landan-desktop-fyne/internal/classifier"
	detector "landan-desktop-fyne/internal/detector"
	domain "landan-desktop-fyne/internal/domain"
	outputPortAnyPipeline "landan-desktop-fyne/internal/output/port/any/pipeline"
)

// DiePipeline 先用偵測模型找出每顆 die，再把每顆裁下來交給分類模型讀點數。
type DiePipeline struct {
	dieTopDetector     *detector.DieTopDetector
	dieValueClassifier *classifier.DieValueClassifier
}

func NewDiePipeline(
	oDieTopDetector *detector.DieTopDetector,
	oDieValueClassifier *classifier.DieValueClassifier,
) outputPortAnyPipeline.DiePipeline {
	return &DiePipeline{
		dieTopDetector:     oDieTopDetector,
		dieValueClassifier: oDieValueClassifier,
	}
}

func (oSelf *DiePipeline) Recognize(aImage []byte) ([]*domain.Die, error) {
	aDies, err := oSelf.dieTopDetector.Recognize(aImage)
	if err != nil {
		return nil, fmt.Errorf("detect dice: %w", err)
	}
	if len(aDies) == 0 {
		return aDies, nil
	}

	oSource, _, err := image.Decode(bytes.NewReader(aImage))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	for iDie, oDie := range aDies {
		aCrop, err := crop(oSource, oDie.X, oDie.Y, oDie.Width, oDie.Height)
		if err != nil {
			return nil, fmt.Errorf("crop die #%d: %w", iDie+1, err)
		}
		oValue, err := oSelf.dieValueClassifier.Classify(aCrop)
		if err != nil {
			return nil, fmt.Errorf("classify die #%d: %w", iDie+1, err)
		}
		oDie.Value = oValue
	}

	return aDies, nil
}
