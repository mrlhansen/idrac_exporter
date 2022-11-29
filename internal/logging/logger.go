package logging

import (
	"runtime"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const RFC3339Millis = "2006-01-02T15:04:05.999"

var llMutex sync.Mutex
var loggersLevels = make(map[string]zap.AtomicLevel)

func SetVerboseLevel() {
	llMutex.Lock()
	for _, level := range loggersLevels {
		level.SetLevel(zap.DebugLevel)
	}
	llMutex.Unlock()
	NewLogger().WithOptions()
}

func SetLoggerLevel(l zapcore.Level) {
	name := getCallerPackageName()

	llMutex.Lock()
	level, ok := loggersLevels[name]
	llMutex.Unlock()

	if ok {
		level.SetLevel(l)
	}
}

func NewLogger(options ...CfgOption) *zap.Logger {
	name := getCallerPackageName()

	cfg := NewCustomConfig()

	llMutex.Lock()
	level, ok := loggersLevels[name]
	if !ok {
		loggersLevels[name] = cfg.Level
	} else {
		cfg.Level = level
	}
	llMutex.Unlock()

	// Apply config options
	for _, option := range options {
		option.apply(&cfg)
	}

	return zap.Must(cfg.Build(zap.AddStacktrace(zap.DPanicLevel))).Named(name)
}

func NewCustomConfig() zap.Config {
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "@timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout(RFC3339Millis),
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	return zap.Config{
		Level:         zap.NewAtomicLevel(),
		DisableCaller: true,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "console",
		EncoderConfig:    encoderCfg,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
}

// An CfgOption configures a Logger.
type CfgOption interface {
	apply(config *zap.Config)
}

// cfgOptionFunc wraps a func so that it satisfies the Option interface.
type cfgOptionFunc func(*zap.Config)

func (f cfgOptionFunc) apply(cfg *zap.Config) {
	f(cfg)
}

func WithOutputs(outputs ...string) CfgOption {
	return cfgOptionFunc(func(cfg *zap.Config) {
		cfg.OutputPaths = outputs
	})
}

func WithLevel(level zapcore.Level) CfgOption {
	return cfgOptionFunc(func(cfg *zap.Config) {
		cfg.Level.SetLevel(level)
	})
}

func WithCallerEnabled() CfgOption {
	return cfgOptionFunc(func(cfg *zap.Config) {
		cfg.DisableCaller = false
	})
}

func WithJsonOutput() CfgOption {
	return cfgOptionFunc(func(cfg *zap.Config) {
		cfg.Encoding = "json"
	})
}

func WithTextOutput() CfgOption {
	return cfgOptionFunc(func(cfg *zap.Config) {
		cfg.Encoding = "console"
	})
}

func getCallerPackageName() string {
	// We need the caller of this function's caller, so skip 2 frames
	frame := getFrame(2)
	return guessPackageName(frame.Function)
}

func getFrame(skipFrames int) runtime.Frame {
	// We never want runtime.Callers and getFrame, so add 2 skip frames
	targetFrameIndex := skipFrames + 2

	// Set size to targetFrameIndex+2 to ensure we have room for one more caller than we need
	programCounters := make([]uintptr, targetFrameIndex+2)
	n := runtime.Callers(0, programCounters)

	frame := runtime.Frame{Function: "unknown"}
	if n > 0 {
		frames := runtime.CallersFrames(programCounters[:n])
		for more, frameIndex := true, 0; more && frameIndex <= targetFrameIndex; frameIndex++ {
			var frameCandidate runtime.Frame
			frameCandidate, more = frames.Next()
			if frameIndex == targetFrameIndex {
				frame = frameCandidate
			}
		}
	}

	return frame
}

func guessPackageName(path string) string {
	// training slashes are tolerated, get rid of one if it exists
	name := strings.TrimSuffix(path, "/")
	if slashIdx := strings.LastIndex(name, "/"); slashIdx >= 0 {
		// if the path contains a "/", use the last part
		name = name[slashIdx+1:]
	}
	if dotIdx := strings.LastIndex(name, "."); dotIdx >= 0 {
		// package name is the first part of function fully qualified name
		name = name[:dotIdx]
	}
	return name
}
