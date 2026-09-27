package outputApplicationOnnx

import (
	"context"
	"fmt"

	onnxruntime "github.com/yalue/onnxruntime_go"
)

// AbstractOnnx 負責 onnxruntime 的全域環境（一個程序只能初始化一次），
// 各 model 共用 Context。
type AbstractOnnx struct {
	Context context.Context
}

func NewAbstractOnnx(oContext context.Context, sLibraryPath string) (*AbstractOnnx, error) {
	if sLibraryPath != "" {
		onnxruntime.SetSharedLibraryPath(sLibraryPath)
	}
	if !onnxruntime.IsInitialized() {
		if err := onnxruntime.InitializeEnvironment(); err != nil {
			return nil, fmt.Errorf("initialize onnxruntime: %w", err)
		}
	}

	return &AbstractOnnx{
		Context: oContext,
	}, nil
}
