package bootstrap

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

//nolint:stylecheck,revive
type Config struct {
	SERVICES struct {
		IR struct {
			PORT string `mapstructure:"port"`
		} `mapstructure:"ir"`
	} `mapstructure:"services"`
	ONNX struct {
		LIBRARY string `mapstructure:"library"`
		DETECT  struct {
			DIE struct {
				CUBE string `mapstructure:"cube"`
				TOP  string `mapstructure:"top"`
			} `mapstructure:"die"`
			POKER struct {
				CARD string `mapstructure:"card"`
			} `mapstructure:"poker"`
		} `mapstructure:"detect"`
		CLASSIFY struct {
			DIE struct {
				VALUE string `mapstructure:"value"`
			} `mapstructure:"die"`
			POKER struct {
				CARD string `mapstructure:"card"`
				RANK string `mapstructure:"rank"`
				SUIT string `mapstructure:"suit"`
			} `mapstructure:"poker"`
		} `mapstructure:"classify"`
	} `mapstructure:"onnx"`
	DEFAULT struct {
		DEBUG bool `mapstructure:"debug"`
	} `mapstructure:"default"`
	DESKTOP struct {
		WIDTH  int `mapstructure:"width"`
		HEIGHT int `mapstructure:"height"`
	} `mapstructure:"desktop"`
	CAMERA struct {
		DEVICE             string `mapstructure:"device"`
		WIDTH              int    `mapstructure:"width"`
		HEIGHT             int    `mapstructure:"height"`
		FRAMERATE          int    `mapstructure:"framerate"`
		RECORD_DIRECTORY   string `mapstructure:"record_directory"`
		RECORD_BITRATE     string `mapstructure:"record_bitrate"`
		PREVIEW_WIDTH      int    `mapstructure:"preview_width"`
		PREVIEW_HEIGHT     int    `mapstructure:"preview_height"`
		PREVIEW_FRAMERATE  int    `mapstructure:"preview_framerate"`
		SNAPSHOT_DIRECTORY string `mapstructure:"snapshot_directory"`
		SNAPSHOT_INTERVAL  int    `mapstructure:"snapshot_interval"`
	} `mapstructure:"camera"`
}

var CONFIG Config

func init() {
	// 加载 .env 到系统环境变量（文件不存在时不报错）
	_ = godotenv.Load()

	oConfig, oErr := Load("./config")
	if oErr != nil {
		log.Fatalf("failed to load config: %v", oErr)
	}
	CONFIG = oConfig
}

// Load 读取 sDirectory 下所有 yaml，文件名作为顶层命名空间。
// 例: camera.yaml 内的 width → camera.width，环境变量 CAMERA_WIDTH 可覆盖它。
func Load(sDirectory string) (Config, error) {

	var oConfig Config

	oViper := viper.New()
	oViper.SetConfigType("yaml")
	oViper.AutomaticEnv()                                   // 读取环境变量
	oViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // 嵌套字段用 _ 连接

	aFiles, oErr := filepath.Glob(filepath.Join(sDirectory, "*.yaml"))
	if oErr != nil {
		return oConfig, fmt.Errorf("glob config dir: %w", oErr)
	}
	for _, sFile := range aFiles {
		sName := strings.TrimSuffix(filepath.Base(sFile), filepath.Ext(sFile))

		oSub := viper.New()
		oSub.SetConfigFile(sFile)
		if oErr := oSub.ReadInConfig(); oErr != nil {
			return oConfig, fmt.Errorf("read config %s: %w", sFile, oErr)
		}
		// 以文件名包一层后合并到主 viper
		if oErr := oViper.MergeConfigMap(map[string]interface{}{sName: oSub.AllSettings()}); oErr != nil {
			return oConfig, fmt.Errorf("merge config %s: %w", sFile, oErr)
		}
	}

	// 自动绑定所有 key，让环境变量覆盖嵌套字段生效
	for _, sKey := range oViper.AllKeys() {
		_ = oViper.BindEnv(sKey)
	}

	if oErr := oViper.Unmarshal(&oConfig); oErr != nil {
		return oConfig, fmt.Errorf("unmarshal config: %w", oErr)
	}

	return oConfig, nil
}

// Validate 检查启动所必需的配置是否齐全，由程序启动时调用。
func (c Config) Validate() error {

	if c.DESKTOP.WIDTH <= 0 || c.DESKTOP.HEIGHT <= 0 {
		return fmt.Errorf("desktop.width/height must be > 0 (is config/desktop.yaml present? run from the project root)")
	}
	if c.CAMERA.WIDTH <= 0 || c.CAMERA.HEIGHT <= 0 || c.CAMERA.FRAMERATE <= 0 {
		return fmt.Errorf("camera.width/height/framerate must be > 0 (is config/camera.yaml present? run from the project root)")
	}
	if c.CAMERA.PREVIEW_WIDTH <= 0 || c.CAMERA.PREVIEW_HEIGHT <= 0 || c.CAMERA.PREVIEW_FRAMERATE <= 0 {
		return fmt.Errorf("camera.preview_width/preview_height/preview_framerate must be > 0")
	}
	if c.CAMERA.SNAPSHOT_INTERVAL < 0 {
		return fmt.Errorf("camera.snapshot_interval must be >= 0")
	}
	return nil
}
