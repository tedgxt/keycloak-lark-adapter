package log

import (
	"keycloak-lark-adapter/internal/config"

	"github.com/sirupsen/logrus"
)

const (
	logTraceLevel = "trace"
	logDebugLevel = "debug"
	logInfoLevel  = "info"
	logErrorLevel = "error"
	logWarnLevel  = "warn"
	logFatalLevel = "fatal"
	logPanicLevel = "panic"
)

var (
	Logger      *logrus.Logger
	loggerLevel = logrus.DebugLevel
)

func Init() {
	Logger = logrus.New()

	//logLevelStr := os.Getenv("LOG_LEVEL")
	switch config.LogLevel {
	case logTraceLevel:
		loggerLevel = logrus.TraceLevel
	case logDebugLevel:
		loggerLevel = logrus.DebugLevel
	case logInfoLevel:
		loggerLevel = logrus.InfoLevel
	case logErrorLevel:
		loggerLevel = logrus.ErrorLevel
	case logWarnLevel:
		loggerLevel = logrus.WarnLevel
	case logFatalLevel:
		loggerLevel = logrus.FatalLevel
	case logPanicLevel:
		loggerLevel = logrus.PanicLevel
	default:
		loggerLevel = logrus.DebugLevel
	}

	Logger.SetLevel(loggerLevel)
	Logger.SetReportCaller(true)

	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	Logger.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
}
