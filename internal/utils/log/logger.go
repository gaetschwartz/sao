package log

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gaetschwartz/devcleaner-go/internal/config"
	"github.com/gaetschwartz/devcleaner-go/internal/utils/ansi"
)

type Logger struct {
	CurrentLevel LogLevel
}

func NewFromEnv() *Logger {
	logger := New()
	if l, err := ParseLevel(config.Runtime.LogLevel); err == nil {
		logger.CurrentLevel = l
	} else {
		logger.Warn("Failed to parse log level in env '%s': %s", config.LogLevelEnvKey, err)
	}
	return logger
}
func New() *Logger {
	return &Logger{
		CurrentLevel: LevelInfo,
	}
}

type LogLevel int

const (
	LevelDebug = LogLevel(iota)
	LevelInfo  = LogLevel(iota)
	LevelWarn  = LogLevel(iota)
	LevelError = LogLevel(iota)
	LevelFatal = LogLevel(iota)
	LevelNone  = LogLevel(iota)
)

func (level LogLevel) String() string {
	switch level {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	case LevelFatal:
		return "fatal"
	default:
		panic("unknown log level")
	}
}

func (l LogLevel) shortString() string {
	return strings.ToUpper(l.String()[0:1])
}

func (l LogLevel) color() ansi.Code {
	switch l {
	case LevelNone:
		return ansi.Empty
	case LevelDebug:
		return ansi.Blue
	case LevelInfo:
		return ansi.Blue
	case LevelWarn:
		return ansi.Yellow
	case LevelError:
		return ansi.Red
	default:
		return ansi.Empty
	}
}

func ParseLevel(s string) (LogLevel, error) {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	case "fatal":
		return LevelFatal, nil
	default:
		return LevelNone, fmt.Errorf("unknown log level %s", s)
	}
}

func (l LogLevel) textColor() ansi.Code {
	switch l {
	case LevelDebug:
		return ansi.Dim
	default:
		return ansi.Empty
	}
}

func (l LogLevel) writer() io.Writer {
	switch l {
	case LevelDebug, LevelInfo, LevelWarn:
		return os.Stdout
	case LevelError, LevelFatal:
		return os.Stderr
	default:
		return os.Stdout
	}
}

func (l *Logger) log(level LogLevel, msg string, args ...any) {
	code := level.color()
	textStyle := level.textColor()
	fmt.Fprintf(level.writer(), string(ansi.Str("[%-5s] ").Style(code, textStyle)+ansi.Str("%s\n").Style(textStyle)), level, fmt.Sprintf(msg, args...))
}

func (l *Logger) Debug(msg string, args ...any) {
	if l.CurrentLevel <= LevelDebug {
		l.log(LevelDebug, msg, args...)
	}
}

func (l *Logger) Info(msg string, args ...any) {
	if l.CurrentLevel <= LevelInfo {
		l.log(LevelInfo, msg, args...)
	}
}

func (l *Logger) Warn(msg string, args ...any) {
	if l.CurrentLevel <= LevelWarn {
		l.log(LevelWarn, msg, args...)
	}
}

func (l *Logger) Error(msg string, args ...any) {
	if l.CurrentLevel <= LevelError {
		l.log(LevelError, msg, args...)
	}
}
