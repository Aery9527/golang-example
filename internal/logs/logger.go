package logs

import (
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

// Logger 持有 per-level chain 和 With() 綁定的 kv-pairs。
type Logger struct {
	bound  []any
	chains [4]*Chain // index = Level
}

// NewLogger 建立 Logger。chains 中 nil 的 level 表示該 level 無輸出（lazy skip）。
func NewLogger(chains [4]*Chain) *Logger {
	return &Logger{chains: chains}
}

// With 回傳共享 chain 但綁定額外 kv-pairs 的新 Logger。
func (l *Logger) With(args ...any) *Logger {
	merged := make([]any, len(l.bound)+len(args))
	copy(merged, l.bound)
	copy(merged[len(l.bound):], args)
	return &Logger{
		bound:  merged,
		chains: l.chains,
	}
}

func (l *Logger) Debug(msg string, fn func() []any)             { l.logKV(LevelDebug, msg, fn) }
func (l *Logger) Info(msg string, fn func() []any)              { l.logKV(LevelInfo, msg, fn) }
func (l *Logger) Warn(msg string, fn func() []any)              { l.logKV(LevelWarn, msg, fn) }
func (l *Logger) Error(msg string, fn func() []any)             { l.logKV(LevelError, msg, fn) }

func (l *Logger) DebugWith(msg string, fn func() (error, []any)) { l.logErr(LevelDebug, msg, fn) }
func (l *Logger) InfoWith(msg string, fn func() (error, []any))  { l.logErr(LevelInfo, msg, fn) }
func (l *Logger) WarnWith(msg string, fn func() (error, []any))  { l.logErr(LevelWarn, msg, fn) }
func (l *Logger) ErrorWith(msg string, fn func() (error, []any)) { l.logErr(LevelError, msg, fn) }

func (l *Logger) logKV(level Level, msg string, fn func() []any) {
	chain := l.chains[level]
	if chain == nil {
		return
	}
	var args []any
	if fn != nil {
		args = fn()
	}
	l.dispatch(level, msg, args, nil)
}

func (l *Logger) logErr(level Level, msg string, fn func() (error, []any)) {
	chain := l.chains[level]
	if chain == nil {
		return
	}
	var (
		err  error
		args []any
	)
	if fn != nil {
		err, args = fn()
	}
	l.dispatch(level, msg, args, err)
}

// dispatch 建構 Entry 並送入 chain。caller 在此預捕獲。
// Call chain: caller → Debug/Info/... → logKV/logErr → dispatch → runtime.Caller(4)
func (l *Logger) dispatch(level Level, msg string, args []any, err error) {
	entry := Entry{
		Time:    time.Now(),
		Level:   level,
		Message: msg,
		Args:    args,
		Error:   err,
		Bound:   l.bound,
	}
	_, file, line, ok := runtime.Caller(4)
	if ok {
		entry.caller = filepath.Base(file) + ":" + strconv.Itoa(line)
	}
	l.chains[level].Execute(entry)
}
