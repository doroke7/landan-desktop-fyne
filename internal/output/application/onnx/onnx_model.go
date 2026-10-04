package outputApplicationOnnx

import (
	"fmt"
	"slices"

	onnxruntime "github.com/yalue/onnxruntime_go"

	"landan-desktop-fyne/internal/inference"
)

// OnnxModel 是用 onnxruntime 跑的模型。
type OnnxModel struct {
	session *onnxruntime.DynamicAdvancedSession
	height  int
	width   int
}

// loadOnnxModel 載入 sPath，檢查是一個輸入、一個輸出，而且輸入是固定的 [1, 3, H, W]。
func (oSelf *AbstractOnnx) loadOnnxModel(sPath string) (inference.Model, error) {
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

	oSessionOptions, err := oSelf.NewSessionOptions()
	if err != nil {
		return nil, fmt.Errorf("model %s: %w", sPath, err)
	}
	if oSessionOptions != nil {
		defer oSessionOptions.Destroy()
	}

	oSession, err := onnxruntime.NewDynamicAdvancedSession(sPath, []string{aInputs[0].Name}, []string{aOutputs[0].Name}, oSessionOptions)
	if err != nil {
		return nil, fmt.Errorf("load model %s: %w", sPath, err)
	}

	return &OnnxModel{session: oSession, height: int(aShape[2]), width: int(aShape[3])}, nil
}

func (oSelf *OnnxModel) InputSize() (int, int) {
	return oSelf.height, oSelf.width
}

func (oSelf *OnnxModel) Run(aInput []float32) ([]float32, []int64, error) {
	oInput, err := onnxruntime.NewTensor(onnxruntime.NewShape(1, 3, int64(oSelf.height), int64(oSelf.width)), aInput)
	if err != nil {
		return nil, nil, err
	}
	defer oInput.Destroy()

	aOutputs := []onnxruntime.Value{nil}
	if err := oSelf.session.Run([]onnxruntime.Value{oInput}, aOutputs); err != nil {
		return nil, nil, fmt.Errorf("run model: %w", err)
	}
	defer aOutputs[0].Destroy()

	oOutput, ok := aOutputs[0].(*onnxruntime.Tensor[float32])
	if !ok {
		return nil, nil, fmt.Errorf("model output is not float32")
	}

	// tensor 在 return 時就釋放了，所以要複製一份。
	return slices.Clone(oOutput.GetData()), slices.Clone(oOutput.GetShape()), nil
}

func (oSelf *OnnxModel) Close() error {
	return oSelf.session.Destroy()
}
