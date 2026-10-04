package outputApplicationOnnx

import (
	"context"
	"fmt"
	"log"
	"strings"

	onnxruntime "github.com/yalue/onnxruntime_go"

	"landan-desktop-fyne/internal/inference"
	"landan-desktop-fyne/pkg/openvino"
)

// Settings 是 onnxruntime 與 OpenVINO 的共用設定。
type Settings struct {
	// Library 是 libonnxruntime 的路徑，空字串就用系統預設位置。
	Library string
	// Provider 是 cpu（空字串也當 cpu）、openvino 或 coreml，是 onnxruntime 的 execution provider，仍然讀 onnx。
	Provider string
	// ProviderOptions 原樣傳給 Provider。
	ProviderOptions map[string]string
	// UseOpenvino 為 true 時改用 OpenVINO 直接讀 .xml，不再用 onnxruntime。
	UseOpenvino bool
	// OpenvinoDevice 是 OpenVINO 編譯的裝置，例如 CPU、GPU、NPU。
	OpenvinoDevice string
}

// AbstractOnnx 負責推論後端的全域環境（onnxruntime 一個程序只能初始化一次），
// 各 model 共用 Context。
type AbstractOnnx struct {
	Context context.Context

	provider        string
	providerOptions map[string]string
	useOpenvino     bool
	openvinoDevice  string
}

func NewAbstractOnnx(oContext context.Context, oSettings Settings) (*AbstractOnnx, error) {
	sProvider := strings.ToLower(strings.TrimSpace(oSettings.Provider))
	switch sProvider {
	case "", "cpu", "openvino", "coreml":
	default:
		return nil, fmt.Errorf("onnx provider %q 不支援，可用: cpu, openvino, coreml", sProvider)
	}

	// 用 OpenVINO 直接讀 .xml 的話不需要 onnxruntime。
	if !oSettings.UseOpenvino {
		if oSettings.Library != "" {
			onnxruntime.SetSharedLibraryPath(oSettings.Library)
		}
		if !onnxruntime.IsInitialized() {
			if err := onnxruntime.InitializeEnvironment(); err != nil {
				return nil, fmt.Errorf("initialize onnxruntime: %w", err)
			}
		}
	}

	return &AbstractOnnx{
		Context:         oContext,
		provider:        sProvider,
		providerOptions: oSettings.ProviderOptions,
		useOpenvino:     oSettings.UseOpenvino,
		openvinoDevice:  oSettings.OpenvinoDevice,
	}, nil
}

// LoadModel 依設定載入模型：用 OpenVINO 就讀 sOpenvinoPath（.xml），否則讀 sOnnxPath（.onnx）。
// 載入成功會印出這個模型實際用的格式與引擎。
func (oSelf *AbstractOnnx) LoadModel(sOnnxPath string, sOpenvinoPath string) (inference.Model, error) {
	if !oSelf.useOpenvino {
		oModel, err := oSelf.loadOnnxModel(sOnnxPath)
		if err != nil {
			return nil, err
		}
		sProvider := oSelf.provider
		if sProvider == "" {
			sProvider = "cpu"
		}
		log.Printf("載入模型 %s  格式: onnx  引擎: onnxruntime %s  provider: %s", sOnnxPath, onnxruntime.GetVersion(), sProvider)
		return oModel, nil
	}

	if sOpenvinoPath == "" {
		return nil, fmt.Errorf("openvino 已啟用，但 %s 沒有對應的 openvino 模型路徑（config/openvino.yaml）", sOnnxPath)
	}
	sDevice := oSelf.openvinoDevice
	if sDevice == "" {
		sDevice = "CPU"
	}
	oModel, err := openvino.Load(sOpenvinoPath, sDevice)
	if err != nil {
		return nil, err
	}
	log.Printf("載入模型 %s  格式: openvino  引擎: OpenVINO  裝置: %s", sOpenvinoPath, sDevice)
	return oModel, nil
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
