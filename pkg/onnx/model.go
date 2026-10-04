// Package onnx 用 onnxruntime 載入並執行 ONNX 模型。onnxruntime 的環境（共用函式庫路徑、InitializeEnvironment）要由呼叫端先初始化。
package onnx

import (
	"fmt"
	"slices"

	onnxruntime "github.com/yalue/onnxruntime_go"
)

// Model 是一個載入好的 ONNX 模型，只有一個輸入、一個輸出，輸入是固定的 [1, 3, H, W]、float32。
type Model struct {
	session *onnxruntime.DynamicAdvancedSession
	height  int
	width   int
}

// Load 載入 sPath。oSessionOptions 可以是 nil（用 onnxruntime 預設，也就是 CPU），Load 回來之後就可以 Destroy 它。
func Load(sPath string, oSessionOptions *onnxruntime.SessionOptions) (*Model, error) {
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

	oSession, err := onnxruntime.NewDynamicAdvancedSession(sPath, []string{aInputs[0].Name}, []string{aOutputs[0].Name}, oSessionOptions)
	if err != nil {
		return nil, fmt.Errorf("load model %s: %w", sPath, err)
	}

	return &Model{session: oSession, height: int(aShape[2]), width: int(aShape[3])}, nil
}

func (oSelf *Model) InputSize() (int, int) {
	return oSelf.height, oSelf.width
}

func (oSelf *Model) Run(aInput []float32) ([]float32, []int64, error) {
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

func (oSelf *Model) Close() error {
	return oSelf.session.Destroy()
}
