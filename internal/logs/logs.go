package logs

import (
	"sync"
)

var (
	defaultLogger *Logger
	initOnce      sync.Once
)

// Init 以指定的 chains 初始化 defaultLogger。由 pkg/logs.Configure 呼叫。
func Init(chains [4]*Chain) {
	initOnce.Do(func() {
		defaultLogger = NewLogger(chains)
		SetWarnChain(chains[LevelWarn])
	})
}

// ensureInit 確保 defaultLogger 已初始化。未呼叫 Init/Configure 時使用預設配置。
func ensureInit() {
	initOnce.Do(func() {
		chains := defaultChains()
		defaultLogger = NewLogger(chains)
		SetWarnChain(chains[LevelWarn])
	})
}

func defaultChains() [4]*Chain {
	plain := NewPlainFormatter()
	console := NewConsoleOutput()
	caller := &CallerEnricher{}

	var chains [4]*Chain
	for i := 0; i < 4; i++ {
		level := Level(i)
		writer := console.Resolve(ResolveContext{Level: level})
		sink := NewSink(plain, writer)
		chains[i] = NewChain(
			[]Handler{caller},
			FanOut([]SinkWriter{sink}),
		)
	}
	return chains
}

// DefaultLogger 回傳 default logger instance。
func DefaultLogger() *Logger {
	ensureInit()
	return defaultLogger
}

// With 回傳綁定 kv-pairs 的新 Logger。
func With(args ...any) *Logger {
	ensureInit()
	return defaultLogger.With(args...)
}

// Package-level convenience functions — call logKV/logErr directly (same depth as Logger methods).
func Debug(msg string, fn func() []any)              { ensureInit(); defaultLogger.logKV(LevelDebug, msg, fn) }
func Info(msg string, fn func() []any)               { ensureInit(); defaultLogger.logKV(LevelInfo, msg, fn) }
func Warn(msg string, fn func() []any)               { ensureInit(); defaultLogger.logKV(LevelWarn, msg, fn) }
func Error(msg string, fn func() []any)              { ensureInit(); defaultLogger.logKV(LevelError, msg, fn) }

func DebugWith(msg string, fn func() (error, []any)) { ensureInit(); defaultLogger.logErr(LevelDebug, msg, fn) }
func InfoWith(msg string, fn func() (error, []any))  { ensureInit(); defaultLogger.logErr(LevelInfo, msg, fn) }
func WarnWith(msg string, fn func() (error, []any))  { ensureInit(); defaultLogger.logErr(LevelWarn, msg, fn) }
func ErrorWith(msg string, fn func() (error, []any)) { ensureInit(); defaultLogger.logErr(LevelError, msg, fn) }

// ResetForTest 重設 internal state（僅供測試使用）。
func ResetForTest() {
	initOnce = sync.Once{}
	defaultLogger = nil
	SetWarnChain(nil)
}
