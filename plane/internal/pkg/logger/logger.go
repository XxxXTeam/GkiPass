/*
Package logger 全局日志系统

提供基于 zap 的结构化日志，支持：
  - 多级别输出：debug / info / warn / error / fatal
  - 双格式：console（开发）/ json（生产）
  - 文件轮转：基于 lumberjack，按大小、数量、天数自动切割和压缩
  - 双输出：配置文件路径后同时写文件和控制台
  - 子日志器：Named() 创建带模块名前缀的子 logger
  - 全局替换：Init 后自动替换 zap.L() / zap.S()，第三方库也能共享同一日志器

使用示例：

	logger.Init(&logger.Config{Level: "info", Format: "console"})
	logger.Info("服务器启动", zap.String("addr", ":8080"))
	subLog := logger.Named("ws-server")
	subLog.Info("WebSocket 就绪")
*/
package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	/* Logger 全局结构化日志器（字段式 API） */
	Logger *zap.Logger
	/* Sugar 全局格式化日志器（printf 式 API） */
	Sugar *zap.SugaredLogger
)

/*
Config 日志配置
功能：控制日志级别、输出格式、文件轮转策略
*/
type Config struct {
	Level      string /* 日志级别：debug, info, warn, error */
	Format     string /* 输出格式：json（生产环境）, console（开发环境，带颜色） */
	OutputPath string /* 日志文件路径，为空则仅输出到控制台 */
	MaxSize    int    /* 单个日志文件最大大小（MB），默认 100 */
	MaxBackups int    /* 保留的旧日志文件数量，默认 10 */
	MaxAge     int    /* 旧日志保留天数，默认 30 */
	Compress   bool   /* 是否 gzip 压缩归档日志 */
}

/*
Init 初始化或重置全局日志系统
功能：根据配置创建 zap.Logger，支持多次调用（如启动时先用默认配置，加载配置后再用配置重建）。
每次调用都会替换全局 zap.L() 和 zap.S()。
*/
func Init(cfg *Config) error {
	/* 填充默认值 */
	if cfg.Level == "" {
		cfg.Level = "info"
	}
	if cfg.Format == "" {
		cfg.Format = "console"
	}
	if cfg.MaxSize == 0 {
		cfg.MaxSize = 100
	}
	if cfg.MaxBackups == 0 {
		cfg.MaxBackups = 10
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 30
	}

	/* 解析日志级别 */
	var level zapcore.Level
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	/* console 编码器：带颜色，适合终端 */
	consoleEncoderCfg := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	/* json 编码器：无颜色，适合日志收集系统 */
	jsonEncoderCfg := consoleEncoderCfg
	jsonEncoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(jsonEncoderCfg)
	} else {
		encoder = zapcore.NewConsoleEncoder(consoleEncoderCfg)
	}

	/* 构建输出目标 */
	var writeSyncer zapcore.WriteSyncer
	if cfg.OutputPath != "" {
		logDir := filepath.Dir(cfg.OutputPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return err
		}
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.OutputPath,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		/* 同时写文件和控制台 */
		writeSyncer = zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(fileWriter),
			zapcore.AddSync(os.Stdout),
		)
	} else {
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	core := zapcore.NewCore(encoder, writeSyncer, level)
	Logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	Sugar = Logger.Sugar()

	/* 替换 zap 全局日志器，使 zap.L() / zap.S() 指向同一实例 */
	zap.ReplaceGlobals(Logger)

	return nil
}

/* Sync 刷新日志缓冲区，应在程序退出前调用 */
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

/*
Named 创建带模块名前缀的子日志器
功能：输出时自动附加模块名，便于过滤和定位
示例：logger.Named("ws-server").Info("连接建立") → [ws-server] 连接建立
*/
func Named(name string) *zap.Logger {
	return Logger.Named(name)
}

/* Debug 输出 DEBUG 级别日志 */
func Debug(msg string, fields ...zap.Field) {
	Logger.Debug(msg, fields...)
}

/* Info 输出 INFO 级别日志 */
func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
}

/* Warn 输出 WARN 级别日志 */
func Warn(msg string, fields ...zap.Field) {
	Logger.Warn(msg, fields...)
}

/* Error 输出 ERROR 级别日志（自动附加堆栈） */
func Error(msg string, fields ...zap.Field) {
	Logger.Error(msg, fields...)
}

/* Fatal 输出 FATAL 级别日志并调用 os.Exit(1) */
func Fatal(msg string, fields ...zap.Field) {
	Logger.Fatal(msg, fields...)
}

/* Debugf printf 风格 DEBUG 日志 */
func Debugf(template string, args ...interface{}) {
	Sugar.Debugf(template, args...)
}

/* Infof printf 风格 INFO 日志 */
func Infof(template string, args ...interface{}) {
	Sugar.Infof(template, args...)
}

/* Warnf printf 风格 WARN 日志 */
func Warnf(template string, args ...interface{}) {
	Sugar.Warnf(template, args...)
}

/* Errorf printf 风格 ERROR 日志 */
func Errorf(template string, args ...interface{}) {
	Sugar.Errorf(template, args...)
}

/* Fatalf printf 风格 FATAL 日志 */
func Fatalf(template string, args ...interface{}) {
	Sugar.Fatalf(template, args...)
}

/* With 返回携带预设字段的子日志器 */
func With(fields ...zap.Field) *zap.Logger {
	return Logger.With(fields...)
}
