package outputApplicationOnnx

import (
	"context"
	"fmt"
	"log"
	"strings"

	onnxruntime "github.com/yalue/onnxruntime_go"

	"landan-desktop-fyne/bootstrap"
	"landan-desktop-fyne/internal/inference"
	"landan-desktop-fyne/pkg/onnx"
	"landan-desktop-fyne/pkg/openvino"
)

// AbstractOnnx 負責推論後端的全域環境（onnxruntime 一個程序只能初始化一次），
// 各 model 共用 Context。
type AbstractOnnx struct {
	Context context.Context

	provider        string
	providerOptions map[string]string
	useOpenvino     bool
	openvinoDevice  string
}

func NewAbstractOnnx(oContext context.Context, oConfig bootstrap.Config) (*AbstractOnnx, error) {
	sProvider := strings.ToLower(strings.TrimSpace(oConfig.ONNX.PROVIDER))
	switch sProvider {
	case "", "cpu", "openvino", "coreml":
	default:
		return nil, fmt.Errorf("onnx provider %q 不支援，可用: cpu, openvino, coreml", sProvider)
	}

	// 用 OpenVINO 直接讀 .xml 的話不需要 onnxruntime。
	if !oConfig.OPENVINO.ENABLED {
		if oConfig.ONNX.LIBRARY != "" {
			onnxruntime.SetSharedLibraryPath(oConfig.ONNX.LIBRARY)
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
		providerOptions: oConfig.ONNX.PROVIDER_OPTIONS,
		useOpenvino:     oConfig.OPENVINO.ENABLED,
		openvinoDevice:  oConfig.OPENVINO.DEVICE,
	}, nil
}

// LoadModel 依設定載入模型：用 OpenVINO 就讀 sOpenvinoPath（.xml），否則讀 sOnnxPath（.onnx）。
// 載入成功會印出這個模型實際用的格式與引擎。
func (oSelf *AbstractOnnx) LoadModel(sOnnxPath string, sOpenvinoPath string) (inference.Model, error) {
	if !oSelf.useOpenvino {
		oSessionOptions, err := oSelf.NewSessionOptions()
		if err != nil {
			return nil, fmt.Errorf("model %s: %w", sOnnxPath, err)
		}
		if oSessionOptions != nil {
			defer oSessionOptions.Destroy()
		}

		oModel, err := onnx.Load(sOnnxPath, oSessionOptions)
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
