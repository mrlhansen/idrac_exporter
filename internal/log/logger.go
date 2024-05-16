package log

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	LevelFatal = 0
	LevelError = 1
	LevelWarn  = 2
	LevelInfo  = 3
	LevelDebug = 4
)

type Logger struct {
	level      int
	dateFormat string
	console    bool
	logFile    string
	file       *os.File
	writer     io.Writer
	mu         sync.Mutex
}

func NewLogger(level int, console bool) *Logger {
	var w io.Writer
	if console {
		w = os.Stdout
	}

	return &Logger{
		level:      level,
		console:    console,
		dateFormat: "2006-01-02T15:04:05.000",
		writer:     w,
	}
}

func (log *Logger) SetLogFile(path string) error {
	log.mu.Lock()
	defer log.mu.Unlock()

	log.logFile = path
	return log.open()
}

func (log *Logger) SetLevel(level int) {
	log.level = level
}

func (log *Logger) Fatal(format string, args ...any) {
	log.write(LevelFatal, format, args...)
	if log.file != nil {
		log.file.Close()
	}
	os.Exit(1)
}

func (log *Logger) Error(format string, args ...any) {
	if LevelError > log.level {
		return
	}
	log.write(LevelError, format, args...)
}

func (log *Logger) Warn(format string, args ...any) {
	if LevelWarn > log.level {
		return
	}
	log.write(LevelWarn, format, args...)
}

func (log *Logger) Info(format string, args ...any) {
	if LevelInfo > log.level {
		return
	}
	log.write(LevelInfo, format, args...)
}

func (log *Logger) Debug(format string, args ...any) {
	if LevelDebug > log.level {
		return
	}
	log.write(LevelDebug, format, args...)
}

func (log *Logger) open() error {
	perms := os.O_WRONLY | os.O_APPEND | os.O_CREATE

	f, err := os.OpenFile(log.logFile, perms, 0o640)
	if err != nil {
		return err
	}

	if log.console {
		log.writer = io.MultiWriter(os.Stdout, f)
	} else {
		log.writer = f
	}

	log.file = f
	return nil
}

func (log *Logger) write(level int, format string, args ...any) {
	var lvlstr string

	if log.writer == nil {
		return
	}

	switch level {
	case LevelFatal:
		lvlstr = "FATAL"
	case LevelError:
		lvlstr = "ERROR"
	case LevelWarn:
		lvlstr = "WARN"
	case LevelInfo:
		lvlstr = "INFO"
	case LevelDebug:
		lvlstr = "DEBUG"
	}

	log.mu.Lock()
	defer log.mu.Unlock()

	dt := time.Now()
	f := fmt.Sprintf(format, args...)
	f = fmt.Sprintf("%s %-5s %s\n", dt.Format(log.dateFormat), lvlstr, strings.TrimSpace(f))
	log.writer.Write([]byte(f))
}
