package log

import (
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"time"
)

var logger *zap.Logger

func init() {
	w := getLogWriter()

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,  // 小写不加颜色编码器
		EncodeTime:     zapcore.ISO8601TimeEncoder,     // ISO8601 UTC 时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder, //
		EncodeCaller:   zapcore.ShortCallerEncoder,     // 全路径编码器
		EncodeName:     zapcore.FullNameEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		w,
		zap.NewAtomicLevel(),
	)

	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
}
func getLogWriter() zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   "./grpc.log",
		MaxSize:    100,   //MB
		MaxBackups: 5,     //保留旧文件的最大个数
		MaxAge:     30,    //保留旧文件的最大天数
		Compress:   false, //是否压缩/归档旧文件
	}
	return zapcore.AddSync(lumberJackLogger)
}

//TimeFormat 时间格式化
func TimeFormat(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000 -0700"))
}

// ZapInterceptor 返回zap.logger实例(把日志写到文件中)
func ZapInterceptor() *zap.Logger {
	grpc_zap.ReplaceGrpcLoggerV2(logger)
	return logger
}
func Fatal(msg string, err error) {
	logger.Fatal(msg, zap.Error(err))
}
func ErrorE(msg string, err error) {
	logger.Error(msg, zap.Error(err))
}
func Error(msg string, fields ...zapcore.Field) {
	logger.Error(msg, fields...)
}
func Debug(msg string, fields ...zapcore.Field) {
	logger.Debug(msg, fields...)
}
func Info(msg string, fields ...zapcore.Field) {
	logger.Info(msg, fields...)
}
func FlushLogger() {
	logger.Sync()
}
