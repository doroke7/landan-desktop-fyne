package outputApplicationOnnxPipeline

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"

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
		aCrop, err := crop(oSource, oDie)
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

// crop 把 oDie 的框（超出圖片的部分會裁掉）從 oSource 切出來，編成 PNG。
func crop(oSource image.Image, oDie *domain.Die) ([]byte, error) {
	oBox := image.Rect(oDie.X, oDie.Y, oDie.X+oDie.Width, oDie.Y+oDie.Height).Intersect(oSource.Bounds())
	if oBox.Empty() {
		return nil, fmt.Errorf("box (%d,%d,%d,%d) is outside the image", oDie.X, oDie.Y, oDie.X+oDie.Width, oDie.Y+oDie.Height)
	}

	oSubImager, ok := oSource.(interface {
		SubImage(image.Rectangle) image.Image
	})
	if !ok {
		return nil, fmt.Errorf("image type %T cannot be cropped", oSource)
	}

	var oBuffer bytes.Buffer
	if err := png.Encode(&oBuffer, oSubImager.SubImage(oBox)); err != nil {
		return nil, err
	}

	return oBuffer.Bytes(), nil
}
