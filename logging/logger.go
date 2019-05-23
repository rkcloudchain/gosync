package logging

import (
	"fmt"
	"log"
	"os"
)

const (
	calldepth = 2
)

var (
	gosyncLogger Logger = &defaultLogger{Logger: log.New(os.Stderr, "gosync", log.LstdFlags)}
)

// SetLogger sets logger that used in gosync service
func SetLogger(l Logger) { gosyncLogger = l }

// Debug logs to the DEBUG log
func Debug(args ...interface{}) {
	gosyncLogger.Debug(args...)
}

// Debugf logs to the DEBUG log
func Debugf(format string, args ...interface{}) {
	gosyncLogger.Debugf(format, args...)
}

// Info logs to the INFO log
func Info(args ...interface{}) {
	gosyncLogger.Info(args...)
}

// Infof logs to the INFO log.
func Infof(format string, args ...interface{}) {
	gosyncLogger.Infof(format, args...)
}

// Warning logs to the WARN log
func Warning(args ...interface{}) {
	gosyncLogger.Warning(args...)
}

// Warningf logs to the WARN log.
func Warningf(format string, args ...interface{}) {
	gosyncLogger.Warningf(format, args...)
}

// Error logs to the ERROR log
func Error(args ...interface{}) {
	gosyncLogger.Error(args...)
}

// Errorf logs to the ERROR log
func Errorf(format string, args ...interface{}) {
	gosyncLogger.Errorf(format, args...)
}

// Fatal logs to the FATAL log
func Fatal(args ...interface{}) {
	gosyncLogger.Fatal(args...)
}

// Fatalf logs to the FATAL log
func Fatalf(format string, args ...interface{}) {
	gosyncLogger.Fatalf(format, args...)
}

// Logger does underlying logging work for rksync
type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warning(args ...interface{})
	Warningf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

// defaultLogger is a default implementation of the Logger interface
type defaultLogger struct {
	*log.Logger
	debug bool
}

func (l *defaultLogger) EnableDebug() {
	l.debug = true
}

func (l *defaultLogger) Debug(args ...interface{}) {
	if l.debug {
		l.Output(calldepth, logHeader("DEBUG", fmt.Sprint(args...)))
	}
}

func (l *defaultLogger) Debugf(format string, args ...interface{}) {
	if l.debug {
		l.Output(calldepth, logHeader("DEBUG", fmt.Sprintf(format, args...)))
	}
}

func (l *defaultLogger) Info(args ...interface{}) {
	l.Output(calldepth, logHeader("INFO", fmt.Sprint(args...)))
}

func (l *defaultLogger) Infof(format string, args ...interface{}) {
	l.Output(calldepth, logHeader("INFO", fmt.Sprintf(format, args...)))
}

func (l *defaultLogger) Warning(args ...interface{}) {
	l.Output(calldepth, logHeader("WARN", fmt.Sprint(args...)))
}

func (l *defaultLogger) Warningf(format string, args ...interface{}) {
	l.Output(calldepth, logHeader("WARN", fmt.Sprintf(format, args...)))
}

func (l *defaultLogger) Error(args ...interface{}) {
	l.Output(calldepth, logHeader("ERROR", fmt.Sprint(args...)))
}

func (l *defaultLogger) Errorf(format string, args ...interface{}) {
	l.Output(calldepth, logHeader("ERROR", fmt.Sprintf(format, args...)))
}

func (l *defaultLogger) Fatal(args ...interface{}) {
	l.Output(calldepth, logHeader("FATAL", fmt.Sprint(args...)))
	os.Exit(1)
}

func (l *defaultLogger) Fatalf(format string, args ...interface{}) {
	l.Output(calldepth, logHeader("FATAL", fmt.Sprintf(format, args...)))
	os.Exit(1)
}

func logHeader(level, msg string) string {
	return fmt.Sprintf("%s: %s", level, msg)
}
