package detector

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"sort"

	onnxruntime "github.com/yalue/onnxruntime_go"
	xdraw "golang.org/x/image/draw"

	outputApplicationOnnx "landan-desktop-fyne/internal/output/application/onnx"
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
	*outputApplicationOnnx.AbstractOnnx
	session    *onnxruntime.DynamicAdvancedSession
	inputName  string
	outputName string
	width      int
	height     int
}

func NewAbstractDetector(oAbstractOnnx *outputApplicationOnnx.AbstractOnnx, sPath string) (*AbstractDetector, error) {
	aInputs, aOutputs, err := onnxruntime.GetInputOutputInfo(sPath)
	if err != nil {
		return nil, fmt.Errorf("read model %s: %w", sPath, err)
	}
	if len(aInputs) != 1 || len(aOutputs) != 1 {
		return nil, fmt.Errorf("model %s: want 1 input and 1 output, got %d and %d", sPath, len(aInputs), len(aOutputs))
	}
	aShape := aInputs[0].Dimensions
	if len(aShape) != 4 || aShape[1] != 3 || aShape[2] <= 0 || aShape[3] <= 0 {
		return nil, fmt.Errorf("model %s: input shape %v is not a fixed [1, 3, H, W]", sPath, aShape)
	}

	oSession, err := onnxruntime.NewDynamicAdvancedSession(sPath, []string{aInputs[0].Name}, []string{aOutputs[0].Name}, nil)
	if err != nil {
		return nil, fmt.Errorf("load model %s: %w", sPath, err)
	}

	return &AbstractDetector{
		AbstractOnnx: oAbstractOnnx,
		session:      oSession,
		inputName:    aInputs[0].Name,
		outputName:   aOutputs[0].Name,
		width:        int(aShape[3]),
		height:       int(aShape[2]),
	}, nil
}

// Recognize 在編碼過的（JPEG 或 PNG）圖上找框，信心低於 fThreshold 的直接丟掉。
func (oSelf *AbstractDetector) Recognize(aImage []byte, fThreshold float32) ([]*Detection, error) {
	oSource, _, err := image.Decode(bytes.NewReader(aImage))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	oInput, err := onnxruntime.NewTensor(onnxruntime.NewShape(1, 3, int64(oSelf.height), int64(oSelf.width)), oSelf.toPlanes(oSource))
	if err != nil {
		return nil, err
	}
	defer oInput.Destroy()

	aOutputs := []onnxruntime.Value{nil}
	if err := oSelf.session.Run([]onnxruntime.Value{oInput}, aOutputs); err != nil {
		return nil, fmt.Errorf("run model: %w", err)
	}
	defer aOutputs[0].Destroy()

	oOutput, ok := aOutputs[0].(*onnxruntime.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("model output is not float32")
	}
	aShape := oOutput.GetShape()
	if len(aShape) != 3 {
		return nil, fmt.Errorf("unsupported output shape %v, want [1, N, 6] or [1, 4+C, N]", aShape)
	}

	// 座標在模型輸入圖上，換算回原圖。
	fScaleX := float32(oSource.Bounds().Dx()) / float32(oSelf.width)
	fScaleY := float32(oSource.Bounds().Dy()) / float32(oSelf.height)

	var aBoxes []*Detection // 座標還在輸入圖的像素上
	if aShape[2] == 6 {
		aBoxes = decodeEndToEnd(oOutput.GetData(), int(aShape[1]), fThreshold)
	} else if aShape[1] > 4 && aShape[1] < aShape[2] {
		// 臨時相容：沒做 NMS 的原始輸出，之後模型都重新匯出成 end2end 就可以拿掉。
		aBoxes = nonMaxSuppression(decodeRaw(oOutput.GetData(), int(aShape[1]), int(aShape[2]), fThreshold), fNmsIoU)
	} else {
		return nil, fmt.Errorf("unsupported output shape %v, want [1, N, 6] or [1, 4+C, N]", aShape)
	}

	for _, oBox := range aBoxes {
		oBox.X = int(float32(oBox.X) * fScaleX)
		oBox.Y = int(float32(oBox.Y) * fScaleY)
		oBox.Width = int(float32(oBox.Width) * fScaleX)
		oBox.Height = int(float32(oBox.Height) * fScaleY)
	}

	return aBoxes, nil
}

// fNmsIoU 是原始輸出做 NMS 時，同類別框重疊超過這個比例就視為同一個。
const fNmsIoU = 0.45

// decodeEndToEnd 解析 [1, N, 6]：x1, y1, x2, y2, confidence, class。
func decodeEndToEnd(aData []float32, nRows int, fThreshold float32) []*Detection {
	var aDetections []*Detection
	for iRow := 0; iRow < nRows; iRow++ {
		aRow := aData[iRow*6 : iRow*6+6]
		if aRow[4] < fThreshold {
			continue
		}
		aDetections = append(aDetections, &Detection{
			X:          int(aRow[0]),
			Y:          int(aRow[1]),
			Width:      int(aRow[2] - aRow[0]),
			Height:     int(aRow[3] - aRow[1]),
			Confidence: aRow[4],
			Class:      int(aRow[5]),
		})
	}
	return aDetections
}

// decodeRaw 解析 [1, 4+C, N]：前 4 列是 cx, cy, w, h，其餘是每個類別的分數。
// 資料是列優先（每一列有 N 個值）。
func decodeRaw(aData []float32, nRows, nCols int, fThreshold float32) []*Detection {
	nClasses := nRows - 4
	var aDetections []*Detection
	for iCol := 0; iCol < nCols; iCol++ {
		iClass, fScore := 0, float32(0)
		for iC := 0; iC < nClasses; iC++ {
			if fValue := aData[(4+iC)*nCols+iCol]; fValue > fScore {
				iClass, fScore = iC, fValue
			}
		}
		if fScore < fThreshold {
			continue
		}
		fCx, fCy := aData[iCol], aData[nCols+iCol]
		fW, fH := aData[2*nCols+iCol], aData[3*nCols+iCol]
		aDetections = append(aDetections, &Detection{
			X:          int(fCx - fW/2),
			Y:          int(fCy - fH/2),
			Width:      int(fW),
			Height:     int(fH),
			Confidence: fScore,
			Class:      iClass,
		})
	}
	return aDetections
}

// nonMaxSuppression 依信心由高到低挑框，丟掉和已挑框同類別且 IoU 超過 fIoU 的。
func nonMaxSuppression(aDetections []*Detection, fIoU float32) []*Detection {
	sort.Slice(aDetections, func(i, j int) bool { return aDetections[i].Confidence > aDetections[j].Confidence })
	var aKept []*Detection
	for _, oCandidate := range aDetections {
		bSuppressed := false
		for _, oKept := range aKept {
			if oKept.Class == oCandidate.Class && iou(oKept, oCandidate) > fIoU {
				bSuppressed = true
				break
			}
		}
		if !bSuppressed {
			aKept = append(aKept, oCandidate)
		}
	}
	return aKept
}

func iou(oA, oB *Detection) float32 {
	iLeft, iTop := max(oA.X, oB.X), max(oA.Y, oB.Y)
	iRight, iBottom := min(oA.X+oA.Width, oB.X+oB.Width), min(oA.Y+oA.Height, oB.Y+oB.Height)
	if iRight <= iLeft || iBottom <= iTop {
		return 0
	}
	fInter := float32((iRight - iLeft) * (iBottom - iTop))
	return fInter / (float32(oA.Width*oA.Height) + float32(oB.Width*oB.Height) - fInter)
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
	return oSelf.session.Destroy()
}
