package detector

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	xdraw "golang.org/x/image/draw"

	outputApplicationInference "landan-desktop-fyne/internal/output/application/inference"
	inference "landan-desktop-fyne/pkg/inference"
)

// Detection 是模型吐出的一個框，座標已換算回原圖的像素。
type Detection struct {
	X, Y, Width, Height int
	Confidence          float32
	Class               int
}

// AbstractDetector 負責偵測模型共用的 onnx 細節：載入 session、前處理、解析輸出。
// 假設模型是 ultralytics 匯出的 end2end（不需要 NMS）：
//   - 輸入 [1, 3, H, W]，RGB，0~1
//   - 輸出 [1, N, 6]，每列是 x1, y1, x2, y2, confidence, class（座標在輸入圖的像素上）
type AbstractDetector struct {
	*outputApplicationInference.AbstractInference
	model  inference.Model
	width  int
	height int
}

// NewAbstractDetector 載入模型：inference engine 為 openvino 時讀 sOpenvinoPath（.xml），否則讀 sOnnxPath（.onnx）。
func NewAbstractDetector(oAbstractInference *outputApplicationInference.AbstractInference, sOnnxPath string, sOpenvinoPath string) (*AbstractDetector, error) {
	oModel, err := oAbstractInference.LoadModel(sOnnxPath, sOpenvinoPath)
	if err != nil {
		return nil, err
	}
	iHeight, iWidth := oModel.InputSize()

	return &AbstractDetector{
		AbstractInference: oAbstractInference,
		model:             oModel,
		width:             iWidth,
		height:            iHeight,
	}, nil
}

// Recognize 在編碼過的（JPEG 或 PNG）圖上找框，信心低於 fThreshold 的直接丟掉。
func (oSelf *AbstractDetector) Recognize(aImage []byte, fThreshold float32) ([]*Detection, error) {
	oSource, _, err := image.Decode(bytes.NewReader(aImage))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	aData, aShape, err := oSelf.model.Run(oSelf.toPlanes(oSource))
	if err != nil {
		return nil, fmt.Errorf("run model: %w", err)
	}
	if len(aShape) != 3 || aShape[2] != 6 {
		return nil, fmt.Errorf("unsupported output shape %v, want [1, N, 6]", aShape)
	}

	// 座標在模型輸入圖上，換算回原圖。
	fScaleX := float32(oSource.Bounds().Dx()) / float32(oSelf.width)
	fScaleY := float32(oSource.Bounds().Dy()) / float32(oSelf.height)

	var aDetections []*Detection
	for iRow := 0; iRow < int(aShape[1]); iRow++ {
		aRow := aData[iRow*6 : iRow*6+6]
		if aRow[4] < fThreshold {
			continue
		}
		aDetections = append(aDetections, &Detection{
			X:          int(aRow[0] * fScaleX),
			Y:          int(aRow[1] * fScaleY),
			Width:      int((aRow[2] - aRow[0]) * fScaleX),
			Height:     int((aRow[3] - aRow[1]) * fScaleY),
			Confidence: aRow[4],
			Class:      int(aRow[5]),
		})
	}

	return aDetections, nil
}

// toPlanes 把圖縮到模型輸入大小，排成 RGB 三個平面（NCHW），數值 0~1。
func (oSelf *AbstractDetector) toPlanes(oSource image.Image) []float32 {
	oResized := image.NewRGBA(image.Rect(0, 0, oSelf.width, oSelf.height))
	xdraw.ApproxBiLinear.Scale(oResized, oResized.Bounds(), oSource, oSource.Bounds(), xdraw.Src, nil)

	nPlane := oSelf.width * oSelf.height
	aData := make([]float32, 3*nPlane)
	for iY := 0; iY < oSelf.height; iY++ {
		for iX := 0; iX < oSelf.width; iX++ {
			aPixel := oResized.Pix[oResized.PixOffset(iX, iY):]
			iIndex := iY*oSelf.width + iX
			aData[iIndex] = float32(aPixel[0]) / 255
			aData[nPlane+iIndex] = float32(aPixel[1]) / 255
			aData[2*nPlane+iIndex] = float32(aPixel[2]) / 255
		}
	}
	return aData
}

// Close 釋放 onnx session。
func (oSelf *AbstractDetector) Close() error {
	return oSelf.model.Close()
}
