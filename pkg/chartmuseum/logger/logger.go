package logger

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	// Logger handles all logger from application
	Logger struct {
		*zap.SugaredLogger
	}

	// LoggerOptions are options for constructing a Logger
	LoggerOptions struct {
		Debug   bool
		LogJSON bool
	}

	// LoggingFn is generic logging function with some additonal context
	LoggingFn func(level logLevel, msg string, keysAndValues ...interface{})

	logLevel string
)

const (
	DebugLevel logLevel = "DEBUG"
	InfoLevel  logLevel = "INFO"
	WarnLevel  logLevel = "WARN"
	ErrorLevel logLevel = "ERROR"
)

// NewLogger creates a new Logger instance
func NewLogger(options LoggerOptions) (*Logger, error) {
	config := zap.NewDevelopmentConfig()
	config.DisableStacktrace = true
	config.Development = false
	config.DisableCaller = true
	if options.LogJSON {
		config.Encoding = "json"
	} else {
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	if !options.Debug {
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	logger, err := config.Build()
	if err != nil {
		return new(Logger), err
	}
	defer logger.Sync()
	return &Logger{logger.Sugar()}, nil
}

/*
ContextLoggingFn creates a LoggingFn to be used in
places that do not necessarily need access to the gin context
*/
func (logger *Logger) ContextLoggingFn(c *gin.Context) LoggingFn {
	return func(level logLevel, msg string, keysAndValues ...interface{}) {
		switch level {
		case DebugLevel:
			logger.Debugc(c, msg, keysAndValues...)
		case InfoLevel:
			logger.Infoc(c, msg, keysAndValues...)
		case WarnLevel:
			logger.Warnc(c, msg, keysAndValues...)
		case ErrorLevel:
			logger.Errorc(c, msg, keysAndValues...)
		}
	}
}

// Debugc wraps Debugw provided by zap, adding data from gin request context
func (logger *Logger) Debugc(c *gin.Context, msg string, keysAndValues ...interface{}) {
	msg, keysAndValues = transformLogcArgs(c, msg, keysAndValues)
	logger.Debugw(msg, keysAndValues...)
}

// Infoc wraps Infow provided by zap, adding data from gin request context
func (logger *Logger) Infoc(c *gin.Context, msg string, keysAndValues ...interface{}) {
	msg, keysAndValues = transformLogcArgs(c, msg, keysAndValues)
	logger.Infow(msg, keysAndValues...)
}

// Warnc wraps Warnw provided by zap, adding data from gin request context
func (logger *Logger) Warnc(c *gin.Context, msg string, keysAndValues ...interface{}) {
	msg, keysAndValues = transformLogcArgs(c, msg, keysAndValues)
	logger.Warnw(msg, keysAndValues...)
}

// Errorc wraps Errorw provided by zap, adding data from gin request context
func (logger *Logger) Errorc(c *gin.Context, msg string, keysAndValues ...interface{}) {
	msg, keysAndValues = transformLogcArgs(c, msg, keysAndValues)
	logger.Errorw(msg, keysAndValues...)
}

// transformLogcArgs prefixes msg with RequestCount and adds RequestId to keysAndValues
func transformLogcArgs(c *gin.Context, msg string, keysAndValues []interface{}) (string, []interface{}) {
	if reqCount, exists := c.Get("requestcount"); exists {
		msg = fmt.Sprintf("[%s] %s", reqCount, msg)
		keysAndValues = append(keysAndValues, "reqID", c.MustGet("requestid"))
	}
	return msg, keysAndValues
}

func init() {
	logrus.SetLevel(logrus.WarnLevel) // silence logs from zsais/go-gin-prometheus
}
