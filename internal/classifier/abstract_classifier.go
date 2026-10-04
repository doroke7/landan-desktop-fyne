package classifier

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"slices"

	onnxruntime "github.com/yalue/onnxruntime_go"
	xdraw "golang.org/x/image/draw"

	outputApplicationOnnx "landan-desktop-fyne/internal/output/application/onnx"
)

// AbstractClassifier 負責分類模型共用的 onnx 細節：載入 session、前處理、解析輸出。
//   - 輸入 [1, 3, H, W]，RGB，0~1，等比例縮放後置中、其餘補黑（跟 python 的 ClassifierInference 一樣）
//   - 輸出 [1, C]，各類別的機率
type AbstractClassifier struct {
	*outputApplicationOnnx.AbstractOnnx
	session    *onnxruntime.DynamicAdvancedSession
	inputName  string
	outputName string
	width      int
	height     int
}

func NewAbstractClassifier(oAbstractOnnx *outputApplicationOnnx.AbstractOnnx, sPath string) (*AbstractClassifier, error) {
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

	return &AbstractClassifier{
		AbstractOnnx: oAbstractOnnx,
		session:      oSession,
		inputName:    aInputs[0].Name,
		outputName:   aOutputs[0].Name,
		width:        int(aShape[3]),
		height:       int(aShape[2]),
	}, nil
}

// Recognize 對編碼過的（JPEG 或 PNG）圖跑一次模型，回傳各類別的機率，index 就是類別編號。
func (oSelf *AbstractClassifier) Recognize(aImage []byte) ([]float32, error) {
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
	if len(aShape) != 2 || aShape[0] != 1 {
		return nil, fmt.Errorf("unsupported output shape %v, want [1, C]", aShape)
	}

	// tensor 在 return 時就釋放了，所以要複製一份。
	return slices.Clone(oOutput.GetData()), nil
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
	return oSelf.session.Destroy()
}
