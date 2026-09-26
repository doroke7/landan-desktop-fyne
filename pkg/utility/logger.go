package pkgUtility

import (
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Module string

const (
	Default  Module = "default"
	Command  Module = "command"
	Cron     Module = "cron"
	Consumer Module = "consumer"

	Tcp Module = "tcp"
	Udp Module = "udp"

	Http                Module = "http"
	HttpAdmin           Module = "http-admin"
	HttpApp             Module = "http-app"
	HttpThird           Module = "http-third"
	HttpTable           Module = "http-table"
	HttpGame            Module = "http-game"
	HttpAdminMiddleware Module = "http-admin-middleware"
	HttpAppMiddleware   Module = "http-app-middleware"
	HttpThirdMiddleware Module = "http-third-middleware"
	HttpTableMiddleware Module = "http-table-middleware"
	HttpGameMiddleware  Module = "http-game-middleware"

	Websocket                Module = "websocket"
	WebsocketAdmin           Module = "websocket-admin"
	WebsocketMiddleware      Module = "websocket-middleware"
	WebsocketAdminMiddleware Module = "websocket-admin-middleware"
	WebsocketAppMiddleware   Module = "websocket-app-middleware"
	WebsocketThirdMiddleware Module = "websocket-third-middleware"

	Socketio   Module = "socketio"
	Socket     Module = "socket"
	Centrifuge Module = "centrifuge"

	Facade                    Module = "facade"
	FacadeAdmin               Module = "facade-admin"
	FacadeGame                Module = "facade-game"
	FacadeTable               Module = "facade-table"
	FacadeRegister            Module = "facade-register"
	FacadeAdminInterceptor    Module = "facade-admin-interceptor"
	FacadeGameInterceptor     Module = "facade-game-interceptor"
	FacadeTableInterceptor    Module = "facade-table-interceptor"
	FacadeRegisterInterceptor Module = "facade-register-interceptor"

	Resource                 Module = "resource"
	ResourceLogic            Module = "resource-logic"
	ResourceModel            Module = "resource-model"
	ResourceLogicInterceptor Module = "resource-logic-interceptor"
	ResourceModelInterceptor Module = "resource-model-interceptor"

	Source Module = "source"

	Client        Module = "client"
	Deamon        Module = "deamon"
	DeamonWatcher Module = "deamon-watcher"

	Repository Module = "repository"
	Sdk        Module = "sdk"
	Publisher  Module = "publisher"
)

var (
	cache = make(map[Module]*zap.Logger)
	mu    sync.Mutex
)

// Get 取得指定模組的 Logger
func Logger(module Module) *zap.Logger {
	mu.Lock()
	defer mu.Unlock()

	if logger, ok := cache[module]; ok {
		return logger
	}

	dir := filepath.Join("runtime", "log", string(module))
	_ = os.MkdirAll(dir, 0755)

	newWriter := func(filename string) zapcore.WriteSyncer {
		return zapcore.AddSync(&lumberjack.Logger{
			Filename:   filepath.Join(dir, filename),
			MaxSize:    100, // MB
			MaxBackups: 10,
			MaxAge:     30, // Days
			Compress:   true,
		})
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)

	oDebugCore := zapcore.NewCore(
		jsonEncoder,
		newWriter("debug.log"),
		zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zap.DebugLevel
		}),
	)

	oInfoCore := zapcore.NewCore(
		jsonEncoder,
		newWriter("info.log"),
		zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zap.InfoLevel
		}),
	)

	oWarnCore := zapcore.NewCore(
		jsonEncoder,
		newWriter("warn.log"),
		zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zap.WarnLevel
		}),
	)

	oErrorCore := zapcore.NewCore(
		jsonEncoder,
		newWriter("error.log"),
		zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zap.ErrorLevel
		}),
	)

	oPanicCore := zapcore.NewCore(
		jsonEncoder,
		newWriter("panic.log"),
		zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zap.PanicLevel
		}),
	)

	oFatalCore := zapcore.NewCore(
		jsonEncoder,
		newWriter("fatal.log"),
		zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zap.FatalLevel
		}),
	)

	// Console（帶顏色，只影響終端輸出，不影響寫檔的 json log）
	consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
	consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	oStdoutCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(consoleEncoderConfig),
		zapcore.AddSync(os.Stdout),
		zap.DebugLevel,
	)

	logger := zap.New(
		zapcore.NewTee(
			oDebugCore,
			oInfoCore,
			oWarnCore,
			oErrorCore,
			oPanicCore,
			oFatalCore,
			oStdoutCore,
		),
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
	).With(
		zap.String("module", string(module)),
	)

	cache[module] = logger

	return logger
}
