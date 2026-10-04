package classifier

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"

	xdraw "golang.org/x/image/draw"

	"landan-desktop-fyne/internal/inference"
	outputApplicationOnnx "landan-desktop-fyne/internal/output/application/onnx"
)

// AbstractClassifier 負責分類模型共用的 onnx 細節：載入 session、前處理、解析輸出。
//   - 輸入 [1, 3, H, W]，RGB，0~1，等比例縮放後置中、其餘補黑（跟 python 的 ClassifierInference 一樣）
//   - 輸出 [1, C]，各類別的機率
type AbstractClassifier struct {
	*outputApplicationOnnx.AbstractOnnx
	model  inference.Model
	width  int
	height int
}

// NewAbstractClassifier 載入模型：config 啟用 OpenVINO 時讀 sOpenvinoPath（.xml），否則讀 sPath（.onnx）。
func NewAbstractClassifier(oAbstractOnnx *outputApplicationOnnx.AbstractOnnx, sPath string, sOpenvinoPath string) (*AbstractClassifier, error) {
	oModel, err := oAbstractOnnx.LoadModel(sPath, sOpenvinoPath)
	if err != nil {
		return nil, err
	}
	iHeight, iWidth := oModel.InputSize()

	return &AbstractClassifier{
		AbstractOnnx: oAbstractOnnx,
		model:        oModel,
		width:        iWidth,
		height:       iHeight,
	}, nil
}

// Recognize 對編碼過的（JPEG 或 PNG）圖跑一次模型，回傳各類別的機率，index 就是類別編號。
func (oSelf *AbstractClassifier) Recognize(aImage []byte) ([]float32, error) {
	oSource, _, err := image.Decode(bytes.NewReader(aImage))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	aData, aShape, err := oSelf.model.Run(oSelf.toPlanes(oSource))
	if err != nil {
		return nil, fmt.Errorf("run model: %w", err)
	}
	if len(aShape) != 2 || aShape[0] != 1 {
		return nil, fmt.Errorf("unsupported output shape %v, want [1, C]", aShape)
	}

	return aData, nil
}

// toPlanes 把圖等比例縮放後置中放進模型輸入大小（其餘補黑），排成 RGB 三個平面（NCHW），數值 0~1。
func (oSelf *AbstractClassifier) toPlanes(oSource image.Image) []float32 {
	iSourceWidth, iSourceHeight := oSource.Bounds().Dx(), oSource.Bounds().Dy()
	fScale := min(float64(oSelf.width)/float64(iSourceWidth), float64(oSelf.height)/float64(iSourceHeight))
	iNewWidth := max(1, int(float64(iSourceWidth)*fScale))
	iNewHeight := max(1, int(float64(iSourceHeight)*fScale))
	iLeft := (oSelf.width - iNewWidth) / 2
	iTop := (oSelf.height - iNewHeight) / 2

	oPadded := image.NewRGBA(image.Rect(0, 0, oSelf.width, oSelf.height))
	draw.Draw(oPadded, oPadded.Bounds(), image.Black, image.Point{}, draw.Src)
	xdraw.ApproxBiLinear.Scale(oPadded, image.Rect(iLeft, iTop, iLeft+iNewWidth, iTop+iNewHeight), oSource, oSource.Bounds(), xdraw.Src, nil)

	nPlane := oSelf.width * oSelf.height
	aData := make([]float32, 3*nPlane)
	for iY := 0; iY < oSelf.height; iY++ {
		for iX := 0; iX < oSelf.width; iX++ {
			aPixel := oPadded.Pix[oPadded.PixOffset(iX, iY):]
			iIndex := iY*oSelf.width + iX
			aData[iIndex] = float32(aPixel[0]) / 255
			aData[nPlane+iIndex] = float32(aPixel[1]) / 255
			aData[2*nPlane+iIndex] = float32(aPixel[2]) / 255
		}
	}
	return aData
}

// Close 釋放 onnx session。
func (oSelf *AbstractClassifier) Close() error {
	return oSelf.model.Close()
}

// Best 對圖跑一次模型，回傳機率最高的類別名稱與機率；aNames 的順序就是類別編號。
func (oSelf *AbstractClassifier) Best(aImage []byte, aNames []string) (string, float32, error) {
	aProbabilities, err := oSelf.Recognize(aImage)
	if err != nil {
		return "", 0, err
	}
	if len(aProbabilities) != len(aNames) {
		return "", 0, fmt.Errorf("unsupported output size %d, want [1, %d]", len(aProbabilities), len(aNames))
	}

	iBest := 0
	for iIndex, fProbability := range aProbabilities {
		if fProbability > aProbabilities[iBest] {
			iBest = iIndex
		}
	}

	return aNames[iBest], aProbabilities[iBest], nil
}
