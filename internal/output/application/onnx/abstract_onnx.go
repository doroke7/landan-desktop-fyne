package outputApplicationOnnx

import (
	"context"
	"fmt"
	"strings"

	onnxruntime "github.com/yalue/onnxruntime_go"
)

// AbstractOnnx 負責 onnxruntime 的全域環境（一個程序只能初始化一次），
// 各 model 共用 Context。
type AbstractOnnx struct {
	Context context.Context

	provider        string
	providerOptions map[string]string
}

// NewAbstractOnnx 的 sProvider 是 cpu（空字串也當 cpu）、openvino 或 coreml，
// mProviderOptions 原樣傳給該 provider。
func NewAbstractOnnx(oContext context.Context, sLibraryPath string, sProvider string, mProviderOptions map[string]string) (*AbstractOnnx, error) {
	sProvider = strings.ToLower(strings.TrimSpace(sProvider))
	switch sProvider {
	case "", "cpu", "openvino", "coreml":
	default:
		return nil, fmt.Errorf("onnx provider %q 不支援，可用: cpu, openvino, coreml", sProvider)
	}

	if sLibraryPath != "" {
		onnxruntime.SetSharedLibraryPath(sLibraryPath)
	}
	if !onnxruntime.IsInitialized() {
		if err := onnxruntime.InitializeEnvironment(); err != nil {
			return nil, fmt.Errorf("initialize onnxruntime: %w", err)
		}
	}

	return &AbstractOnnx{
		Context:         oContext,
		provider:        sProvider,
		providerOptions: mProviderOptions,
	}, nil
}

// NewSessionOptions 依設定的 provider 建 session options；cpu 回傳 nil（用 onnxruntime 預設）。
// 呼叫端建完 session 之後要 Destroy 回傳的 options（非 nil 時）。
func (oSelf *AbstractOnnx) NewSessionOptions() (*onnxruntime.SessionOptions, error) {
	if oSelf.provider == "" || oSelf.provider == "cpu" {
		return nil, nil
	}

	oOptions, err := onnxruntime.NewSessionOptions()
	if err != nil {
		return nil, fmt.Errorf("create session options: %w", err)
	}

	switch oSelf.provider {
	case "openvino":
		err = oOptions.AppendExecutionProviderOpenVINO(oSelf.providerOptions)
	case "coreml":
		err = oOptions.AppendExecutionProviderCoreMLV2(oSelf.providerOptions)
	}
	if err != nil {
		_ = oOptions.Destroy()
		return nil, fmt.Errorf("enable %s provider（libonnxruntime 需要有編進這個 provider）: %w", oSelf.provider, err)
	}

	return oOptions, nil
}
