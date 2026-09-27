package outputApplicationOnnxModel

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"

	onnxruntime "github.com/yalue/onnxruntime_go"
	xdraw "golang.org/x/image/draw"

	bootstrap "landan-desktop-fyne/bootstrap"
	domain "landan-desktop-fyne/internal/domain"
	outputPortAnyModel "landan-desktop-fyne/internal/output/port/any/model"
)

// 骰子點數的類別數，順序是 1 ~ 6（見 pk-studio-ir-model 的 cfg/classify/die/value-data.yaml）。
const nDieValueClasses = 6

// DieValueClassifierModel 用 onnx 跑 die 點數的分類模型。
//   - 輸入 [1, 3, H, W]，RGB，0~1，等比例縮放後置中、其餘補黑（跟 python 的 ClassifierInference 一樣）
//   - 輸出 [1, 6]，各點數的機率，index 0 對應點數 1
type DieValueClassifierModel struct {
	*AbstractModel
	session    *onnxruntime.DynamicAdvancedSession
	inputName  string
	outputName string
	width      int
	height     int
}

// NewDieValueClassifierModel 從 config/onnx.yaml 的 classify.die.value 讀模型路徑。
func NewDieValueClassifierModel(oAbstractModel *AbstractModel) (outputPortAnyModel.DieValueClassifierModel, error) {
	sModelPath := bootstrap.CONFIG.ONNX.CLASSIFY.DIE.VALUE
	if sModelPath == "" {
		return nil, fmt.Errorf("onnx.classify.die.value is empty (is config/onnx.yaml filled in? run from the project root)")
	}

	aInputs, aOutputs, err := onnxruntime.GetInputOutputInfo(sModelPath)
	if err != nil {
		return nil, fmt.Errorf("read model %s: %w", sModelPath, err)
	}
	if len(aInputs) != 1 || len(aOutputs) != 1 {
		return nil, fmt.Errorf("model %s: want 1 input and 1 output, got %d and %d", sModelPath, len(aInputs), len(aOutputs))
	}
	aShape := aInputs[0].Dimensions
	if len(aShape) != 4 || aShape[1] != 3 || aShape[2] <= 0 || aShape[3] <= 0 {
		return nil, fmt.Errorf("model %s: input shape %v is not a fixed [1, 3, H, W]", sModelPath, aShape)
	}

	oSession, err := onnxruntime.NewDynamicAdvancedSession(sModelPath, []string{aInputs[0].Name}, []string{aOutputs[0].Name}, nil)
	if err != nil {
		return nil, fmt.Errorf("load model %s: %w", sModelPath, err)
	}

	return &DieValueClassifierModel{
		AbstractModel: oAbstractModel,
		session:       oSession,
		inputName:     aInputs[0].Name,
		outputName:    aOutputs[0].Name,
		width:         int(aShape[3]),
		height:        int(aShape[2]),
	}, nil
}

func (oSelf *DieValueClassifierModel) Classify(aImage []byte) (*domain.DieValue, error) {
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
	if len(aShape) != 2 || aShape[1] != nDieValueClasses {
		return nil, fmt.Errorf("unsupported output shape %v, want [1, %d]", aShape, nDieValueClasses)
	}

	aProbabilities := oOutput.GetData()
	iBest := 0
	for iIndex, fProbability := range aProbabilities {
		if fProbability > aProbabilities[iBest] {
			iBest = iIndex
		}
	}

	return &domain.DieValue{
		Value:      iBest + 1,
		Confidence: aProbabilities[iBest],
	}, nil
}

// toPlanes 把圖等比例縮放後置中放進模型輸入大小（其餘補黑），排成 RGB 三個平面（NCHW），數值 0~1。
func (oSelf *DieValueClassifierModel) toPlanes(oSource image.Image) []float32 {
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
func (oSelf *DieValueClassifierModel) Close() error {
	return oSelf.session.Destroy()
}
