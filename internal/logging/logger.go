package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

type Logger struct {
	logger   *log.Logger
	level    Level
	mu       sync.RWMutex
	reqIDKey string
	writer   io.Writer // The underlying writer (can be MultiWriter)
}

func NewLogger(prefix, level string) *Logger {
	return NewLoggerWithWriter(os.Stderr, prefix, level)
}

func NewLoggerWithWriter(w io.Writer, prefix, level string) *Logger {
	l := &Logger{
		logger:   log.New(w, prefix, log.LstdFlags|log.Lmicroseconds),
		level:    parseLevel(level),
		reqIDKey: "request_id",
		writer:   w,
	}
	return l
}

func parseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) GetLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

func (l *Logger) shouldLog(level Level) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level >= l.level
}

func (l *Logger) Debug(format string, v ...interface{}) {
	if l.shouldLog(DebugLevel) {
		l.logger.Printf("[DEBUG] "+format, v...)
	}
}

func (l *Logger) Info(format string, v ...interface{}) {
	if l.shouldLog(InfoLevel) {
		l.logger.Printf("[INFO] "+format, v...)
	}
}

func (l *Logger) Warn(format string, v ...interface{}) {
	if l.shouldLog(WarnLevel) {
		l.logger.Printf("[WARN] "+format, v...)
	}
}

func (l *Logger) Error(format string, v ...interface{}) {
	if l.shouldLog(ErrorLevel) {
		l.logger.Printf("[ERROR] "+format, v...)
	}
}

func (l *Logger) WithRequestID(reqID string) *Logger {
	writer := l.writer
	if writer == nil {
		writer = os.Stderr
	}
	newLogger := &Logger{
		logger:   log.New(writer, fmt.Sprintf("%s[%s] ", l.logger.Prefix(), reqID), log.LstdFlags|log.Lmicroseconds),
		level:    l.level,
		reqIDKey: l.reqIDKey,
		writer:   writer,
	}
	return newLogger
}

func (l *Logger) Printf(format string, v ...interface{}) {
	l.Info(format, v...)
}

func (l *Logger) Fatal(format string, v ...interface{}) {
	l.logger.Fatalf("[FATAL] "+format, v...)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.Fatal(format, v...)
}
