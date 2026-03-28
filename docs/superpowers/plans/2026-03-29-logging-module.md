# Logging Module Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a decorator-pattern logging engine (`internal/logs`) with per-level independent chains, full lazy closure API, and a public config DSL (`pkg/logs`).

**Architecture:** `internal/logs` holds the engine (Entry, Handler chain, Formatters, Sink, Logger). `pkg/logs` is a thin config layer that builds chains and calls `internal/logs.Init()`. All filter/enrichment/formatter/output implementations live in `internal/logs`; `pkg/logs` re-exports builder functions via type aliases. Zero external dependencies (stdlib only; `testify` in tests).

**Tech Stack:** Go 1.26, stdlib (`fmt`, `io`, `os`, `sync`, `time`, `runtime`, `encoding/json`, `path/filepath`, `strconv`, `strings`, `bytes`, `compress/gzip`), `github.com/stretchr/testify` for tests.

**Design Spec:** [`docs/logs-design.md`](../../logs-design.md)

---

## File Map

```
internal/logs/                  (engine + API)
├── entry.go                    ← Level enum, Entry struct
├── entry_test.go
├── handler.go                  ← Handler interface, Chain struct, NewChain, FanOut
├── handler_test.go
├── sink.go                     ← Formatter interface, FormatterOption, Sink struct
├── sink_test.go
├── detect.go                   ← duck-typing interfaces, ErrorInfo, ExtractError
├── detect_test.go
├── format_plain.go             ← PlainFormatter
├── format_plain_test.go
├── format_json.go              ← JSONFormatter
├── format_json_test.go
├── internal_warn.go            ← internalWarn, stderrFallback, SetWarnChain
├── internal_warn_test.go
├── output.go                   ← Output interface, ResolveContext, Console/Stdout/Stderr/Writer outputs
├── output_test.go
├── filter.go                   ← MessageFilter, KeyFilter (Handler implementations)
├── filter_test.go
├── enrichment.go               ← CallerEnricher, StaticEnricher (Handler implementations)
├── enrichment_test.go
├── logger.go                   ← Logger struct, With, dispatch, 8 log methods (logKV/logErr helpers)
├── logger_test.go
├── logs.go                     ← defaultLogger, Init, ensureInit, package-level convenience funcs
└── logs_test.go

pkg/logs/                       (public config entry)
├── rotate.go                   ← RotatingFileWriter, RotateConfig
├── rotate_test.go
├── output.go                   ← type aliases + ToConsole/ToFile/... re-exports
├── filter.go                   ← FilterByKey/FilterByMessage re-exports
├── enrichment.go               ← Caller/Static/NoCaller re-exports
├── formatter.go                ← type aliases + Plain/JSON/WithTimeFormat re-exports
├── config.go                   ← Configure, Option, ForXxx, Pipe, WithFilter, WithEnrichment, NoInherit, NoCaller merge logic
└── config_test.go

刪除:
├── pkg/logger/logger.go        ← 舊 logger（已確認零 import）
└── docs-plan/log-plan.md       ← 舊計畫（由 logs-design.md 取代）
```

---

### Task 1: Cleanup — Remove Old Files

**Files:**
- Delete: `pkg/logger/logger.go`
- Delete: `docs-plan/log-plan.md`

- [ ] **Step 1: Verify zero imports of old logger**

Run: `grep -r "pkg/logger" --include="*.go" .`
Expected: zero matches

- [ ] **Step 2: Delete old files**

```bash
rm pkg/logger/logger.go
rmdir pkg/logger
rm docs-plan/log-plan.md
```

- [ ] **Step 3: Verify project still compiles**

Run: `go build ./...`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore: remove old pkg/logger and outdated log-plan.md"
```

---

### Task 2: Entry & Level (`internal/logs/entry.go`)

**Files:**
- Create: `internal/logs/entry.go`
- Create: `internal/logs/entry_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/logs/entry_test.go
package logs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.level.String())
	}
}

func TestLevel_Values(t *testing.T) {
	// Ensure iota ordering: Debug=0, Info=1, Warn=2, Error=3
	assert.Equal(t, Level(0), LevelDebug)
	assert.Equal(t, Level(1), LevelInfo)
	assert.Equal(t, Level(2), LevelWarn)
	assert.Equal(t, Level(3), LevelError)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/logs/ -run TestLevel -v`
Expected: FAIL (package does not exist)

- [ ] **Step 3: Implement Entry & Level**

```go
// internal/logs/entry.go
package logs

import "time"

// Level 表示 log 嚴重程度。int8 只有 4 個值，省記憶體且比較更快。
type Level int8

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var levelNames = [4]string{"DEBUG", "INFO", "WARN", "ERROR"}

// String 回傳 level 名稱，供 formatter 使用。
func (l Level) String() string {
	if l >= LevelDebug && l <= LevelError {
		return levelNames[l]
	}
	return "UNKNOWN"
}

// Entry 是 log pipeline 的傳輸單元。所有欄位在建構後由 chain 內各 Handler 讀取或修改。
type Entry struct {
	Time     time.Time // chain 入口捕獲，確保 fan-out 時 timestamp 一致
	Level    Level
	Message  string
	Args     []any  // kv-pairs（lazy closure 的回傳值）
	Error    error  // WithXxx variant 的 error，一般 variant 為 nil
	Bound    []any  // With() 綁定的 kv-pairs（由 Logger 注入）
	internal bool   // internal warn 標記，用於遞迴保護
	caller   string // runtime.Caller 預捕獲結果，由 CallerEnricher 注入 Bound
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/logs/ -run TestLevel -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/logs/entry.go internal/logs/entry_test.go
git commit -m "feat(logs): add Level enum and Entry struct"
```

---

### Task 3: Handler & Chain (`internal/logs/handler.go`)

**Files:**
- Create: `internal/logs/handler.go`
- Create: `internal/logs/handler_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/logs/handler_test.go
package logs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 記錄用 spy Handler
type spyHandler struct {
	called bool
	entry  Entry
}

func (s *spyHandler) Handle(entry Entry, next func(Entry)) {
	s.called = true
	s.entry = entry
	next(entry)
}

// 攔截用 Handler（不呼叫 next）
type blockHandler struct{}

func (blockHandler) Handle(entry Entry, next func(Entry)) {}

func TestChain_Execute_NoHandlers(t *testing.T) {
	var got Entry
	terminal := func(e Entry) { got = e }
	chain := NewChain(nil, terminal)

	entry := Entry{Message: "hello"}
	chain.Execute(entry)
	assert.Equal(t, "hello", got.Message)
}

func TestChain_Execute_SingleHandler(t *testing.T) {
	spy := &spyHandler{}
	var got Entry
	terminal := func(e Entry) { got = e }
	chain := NewChain([]Handler{spy}, terminal)

	entry := Entry{Message: "test"}
	chain.Execute(entry)

	assert.True(t, spy.called)
	assert.Equal(t, "test", got.Message)
}

func TestChain_Execute_HandlerModifiesEntry(t *testing.T) {
	modifier := HandlerFunc(func(entry Entry, next func(Entry)) {
		entry.Message = "modified"
		next(entry)
	})
	var got Entry
	terminal := func(e Entry) { got = e }
	chain := NewChain([]Handler{modifier}, terminal)

	chain.Execute(Entry{Message: "original"})
	assert.Equal(t, "modified", got.Message)
}

func TestChain_Execute_HandlerBlocks(t *testing.T) {
	reached := false
	terminal := func(e Entry) { reached = true }
	chain := NewChain([]Handler{blockHandler{}}, terminal)

	chain.Execute(Entry{})
	assert.False(t, reached)
}

func TestChain_Execute_MultipleHandlers_Order(t *testing.T) {
	var order []string
	h1 := HandlerFunc(func(e Entry, next func(Entry)) {
		order = append(order, "h1")
		next(e)
	})
	h2 := HandlerFunc(func(e Entry, next func(Entry)) {
		order = append(order, "h2")
		next(e)
	})
	terminal := func(e Entry) { order = append(order, "terminal") }
	chain := NewChain([]Handler{h1, h2}, terminal)

	chain.Execute(Entry{})
	assert.Equal(t, []string{"h1", "h2", "terminal"}, order)
}

func TestFanOut(t *testing.T) {
	var count int
	s1 := &mockSink{fn: func(Entry) { count++ }}
	s2 := &mockSink{fn: func(Entry) { count++ }}
	terminal := FanOut([]SinkWriter{s1, s2})

	terminal(Entry{})
	assert.Equal(t, 2, count)
}

// mockSink implements SinkWriter
type mockSink struct {
	fn func(Entry)
}

func (m *mockSink) Write(entry Entry) { m.fn(entry) }
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/logs/ -run TestChain -v`
Expected: FAIL

- [ ] **Step 3: Implement Handler & Chain**

```go
// internal/logs/handler.go
package logs

// Handler 是 decorator chain 的單一節點。
// 不回傳 error——logging 系統內部消化所有失敗。
type Handler interface {
	Handle(entry Entry, next func(Entry))
}

// HandlerFunc 將普通函式轉為 Handler（便於測試與輕量場景）。
type HandlerFunc func(Entry, func(Entry))

func (f HandlerFunc) Handle(entry Entry, next func(Entry)) { f(entry, next) }

// SinkWriter 是 chain terminal 節點的介面——Sink 實作此介面。
type SinkWriter interface {
	Write(entry Entry)
}

// Chain 持有預組裝好的 closure chain，Execute 時是純 function call。
type Chain struct {
	exec func(Entry)
}

// NewChain 從 handlers + terminal function 組裝 closure chain。
// handlers 從前到後執行，最後呼叫 terminal（fan-out 到所有 Sink）。
func NewChain(handlers []Handler, terminal func(Entry)) *Chain {
	next := terminal
	for i := len(handlers) - 1; i >= 0; i-- {
		h := handlers[i]
		n := next
		next = func(e Entry) {
			h.Handle(e, n)
		}
	}
	return &Chain{exec: next}
}

// Execute 送 entry 進 chain。
func (c *Chain) Execute(entry Entry) {
	if c.exec != nil {
		c.exec(entry)
	}
}

// FanOut 建立 terminal function，將 entry fan-out 到多個 SinkWriter。
func FanOut(sinks []SinkWriter) func(Entry) {
	return func(entry Entry) {
		for _, s := range sinks {
			s.Write(entry)
		}
	}
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/logs/ -run "TestChain|TestFanOut" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/logs/handler.go internal/logs/handler_test.go
git commit -m "feat(logs): add Handler interface and Chain with closure-based execution"
```

---

### Task 4: Formatter Interface & Sink (`internal/logs/sink.go`)

**Files:**
- Create: `internal/logs/sink.go`
- Create: `internal/logs/sink_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/logs/sink_test.go
package logs

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubFormatter struct {
	output []byte
	err    error
}

func (f *stubFormatter) Format(entry Entry) ([]byte, error) {
	return f.output, f.err
}

func TestSink_Write_Success(t *testing.T) {
	var buf bytes.Buffer
	sink := NewSink(&stubFormatter{output: []byte("hello\n")}, &buf)

	sink.Write(Entry{Message: "test"})
	assert.Equal(t, "hello\n", buf.String())
}

func TestSink_Write_FormatError_InternalEntry_FallsBackToStderr(t *testing.T) {
	var buf bytes.Buffer
	sink := NewSink(&stubFormatter{err: errors.New("bad format")}, &buf)

	// internal entry → should fallback to stderr, NOT recurse
	sink.Write(Entry{Message: "test", internal: true})
	assert.Empty(t, buf.String()) // nothing written to the normal writer
}

func TestSink_Write_WriteError_InternalEntry_FallsBackToStderr(t *testing.T) {
	sink := NewSink(
		&stubFormatter{output: []byte("data")},
		&failWriter{err: errors.New("disk full")},
	)

	// internal entry → should fallback to stderr
	sink.Write(Entry{Message: "test", internal: true})
}

type failWriter struct {
	err error
}

func (w *failWriter) Write(p []byte) (int, error) {
	return 0, w.err
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/logs/ -run TestSink -v`
Expected: FAIL

- [ ] **Step 3: Implement Formatter interface & Sink**

```go
// internal/logs/sink.go
package logs

import "io"

// Formatter 將 Entry 格式化為 byte slice。
type Formatter interface {
	Format(entry Entry) ([]byte, error)
}

// formatterConfig 是 PlainFormatter / JSONFormatter 共用的設定。
type formatterConfig struct {
	timeLayout string
}

// FormatterOption 用於自訂 Formatter 行為。
type FormatterOption func(*formatterConfig)

// WithTimeFormat 設定 time 輸出格式（Go time layout string）。
func WithTimeFormat(layout string) FormatterOption {
	return func(c *formatterConfig) {
		c.timeLayout = layout
	}
}

func applyFormatterOpts(defaults formatterConfig, opts []FormatterOption) formatterConfig {
	cfg := defaults
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

// Sink 是 chain 的 terminal node——format + write。
type Sink struct {
	formatter Formatter
	writer    io.Writer
}

// NewSink 建立 Sink。
func NewSink(f Formatter, w io.Writer) *Sink {
	return &Sink{formatter: f, writer: w}
}

// Write 格式化 entry 並寫入 writer。失敗時走 internal warn 或 stderr fallback。
func (s *Sink) Write(entry Entry) {
	data, err := s.formatter.Format(entry)
	if err != nil {
		s.handleError("format failed", err, entry)
		return
	}
	if _, err := s.writer.Write(data); err != nil {
		s.handleError("write failed", err, entry)
	}
}

func (s *Sink) handleError(msg string, err error, original Entry) {
	if original.internal {
		stderrFallback(msg, err)
		return
	}
	internalWarn(msg, "error", err)
}
```

> **注意**：此 task 的 `stderrFallback` 和 `internalWarn` 在 Task 7 實作。此處先建立 stub 使編譯通過：

```go
// internal/logs/internal_warn.go（stub，Task 7 完成實作）
package logs

import (
	"fmt"
	"os"
)

func stderrFallback(msg string, err error) {
	fmt.Fprintf(os.Stderr, "[LOGS_INTERNAL] %s: %v\n", msg, err)
}

func internalWarn(msg string, kvs ...any) {
	// stub — 先 fallback 到 stderr，Task 7 實作完整版
	fmt.Fprintf(os.Stderr, "[LOGS_INTERNAL_WARN] %s %v\n", msg, kvs)
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/logs/ -run TestSink -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/logs/sink.go internal/logs/sink_test.go internal/logs/internal_warn.go
git commit -m "feat(logs): add Formatter interface, Sink struct, and internal_warn stubs"
```

---

### Task 5: Duck-Typing Detect (`internal/logs/detect.go`)

**Files:**
- Create: `internal/logs/detect.go`
- Create: `internal/logs/detect_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/logs/detect_test.go
package logs

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 完整 duck-typing error（模擬 *errs.Error）
type fullError struct {
	code    string
	message string
	stack   string
}

func (e *fullError) Error() string        { return "[" + e.code + "] " + e.message }
func (e *fullError) Code() string         { return e.code }
func (e *fullError) Message() string      { return e.message }
func (e *fullError) FormatStack() string  { return e.stack }

// 只有 fmt.Formatter 的 error
type formatterError struct {
	msg   string
	trace string
}

func (e *formatterError) Error() string { return e.msg }
func (e *formatterError) Format(f fmt.State, verb rune) {
	if verb == 'v' && f.Flag('+') {
		fmt.Fprintf(f, "%s\n%s", e.msg, e.trace)
		return
	}
	fmt.Fprint(f, e.msg)
}

func TestExtractError_FullDuckTyping(t *testing.T) {
	err := &fullError{code: "DB_FAIL", message: "conn lost", stack: "svc.Load (svc.go:42)"}
	info := ExtractError(err)

	assert.Equal(t, "DB_FAIL", info.Code)
	assert.Equal(t, "conn lost", info.Message)
	assert.Equal(t, "svc.Load (svc.go:42)", info.Stack)
}

func TestExtractError_PlainError(t *testing.T) {
	err := errors.New("something broke")
	info := ExtractError(err)

	assert.Equal(t, "", info.Code)
	assert.Equal(t, "something broke", info.Message)
	assert.Equal(t, "", info.Stack)
}

func TestExtractError_FmtFormatterFallback(t *testing.T) {
	err := &formatterError{msg: "fail", trace: "at main.go:1"}
	info := ExtractError(err)

	assert.Equal(t, "", info.Code)
	assert.Equal(t, "fail", info.Message)
	assert.Contains(t, info.Stack, "at main.go:1")
}

func TestExtractError_Nil(t *testing.T) {
	info := ExtractError(nil)
	assert.Equal(t, ErrorInfo{}, info)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/logs/ -run TestExtractError -v`
Expected: FAIL

- [ ] **Step 3: Implement duck-typing detection**

```go
// internal/logs/detect.go
package logs

import "fmt"

// duck-typing 偵測介面——不 import pkg/errs，任何 error 實作同簽名即相容。

type codeProvider interface {
	Code() string
}

type messageProvider interface {
	Message() string
}

type stackProvider interface {
	FormatStack() string
}

// ErrorInfo 萃取自 error 的結構化資訊。
type ErrorInfo struct {
	Code    string
	Message string
	Stack   string
}

// ExtractError 以 duck-typing 從 error 萃取 code / message / stack。
func ExtractError(err error) ErrorInfo {
	if err == nil {
		return ErrorInfo{}
	}

	var info ErrorInfo

	// Message: messageProvider → err.Error()
	if mp, ok := err.(messageProvider); ok {
		info.Message = mp.Message()
	} else {
		info.Message = err.Error()
	}

	// Code: codeProvider → 空字串
	if cp, ok := err.(codeProvider); ok {
		info.Code = cp.Code()
	}

	// Stack: stackProvider → fmt.Formatter(%+v) → 空字串
	if sp, ok := err.(stackProvider); ok {
		info.Stack = sp.FormatStack()
	} else if _, ok := err.(fmt.Formatter); ok {
		info.Stack = fmt.Sprintf("%+v", err)
	}

	return info
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/logs/ -run TestExtractError -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/logs/detect.go internal/logs/detect_test.go
git commit -m "feat(logs): add duck-typing error detection (zero import of pkg/errs)"
```

---

### Task 6: PlainFormatter (`internal/logs/format_plain.go`)

**Files:**
- Create: `internal/logs/format_plain.go`
- Create: `internal/logs/format_plain_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/logs/format_plain_test.go
package logs

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testTime = time.Date(2026, 3, 28, 14, 5, 23, 456000000, time.Local)

func TestPlainFormatter_BasicMessage(t *testing.T) {
	f := NewPlainFormatter()
	out, err := f.Format(Entry{
		Time:    testTime,
		Level:   LevelInfo,
		Message: "server started",
	})
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "[INFO ]")
	assert.Contains(t, s, "server started")
	assert.True(t, strings.HasSuffix(s, "\n"))
}

func TestPlainFormatter_KVPairs_Numbering(t *testing.T) {
	f := NewPlainFormatter()
	out, err := f.Format(Entry{
		Time:    testTime,
		Level:   LevelInfo,
		Message: "user login",
		Args:    []any{"user_id", 42, "ip", "1.2.3.4"},
	})
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "(1)")
	assert.Contains(t, s, "user_id")
	assert.Contains(t, s, "(2)")
	assert.Contains(t, s, "ip")
}

func TestPlainFormatter_BoundAndArgs_SharedCounter(t *testing.T) {
	f := NewPlainFormatter()
	out, err := f.Format(Entry{
		Time:    testTime,
		Level:   LevelInfo,
		Message: "test",
		Bound:   []any{"rid", "abc"},
		Args:    []any{"uid", 1},
	})
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "(1)")
	assert.Contains(t, s, "rid")
	assert.Contains(t, s, "(2)")
	assert.Contains(t, s, "uid")
}

func TestPlainFormatter_KeyAlignment(t *testing.T) {
	f := NewPlainFormatter()
	out, err := f.Format(Entry{
		Time:    testTime,
		Level:   LevelInfo,
		Message: "test",
		Args:    []any{"request_id", "abc-123", "id", 42},
	})
	require.NoError(t, err)
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	// kv lines should have aligned ":"
	require.True(t, len(lines) >= 3)
	col1 := strings.Index(lines[1], ":")
	col2 := strings.Index(lines[2], ":")
	assert.Equal(t, col1, col2, "colons should be aligned")
}

func TestPlainFormatter_WithError_WidthAligned(t *testing.T) {
	f := NewPlainFormatter()
	out, err := f.Format(Entry{
		Time:    testTime,
		Level:   LevelError,
		Message: "query failed",
		Args:    []any{"table", "users"},
		Error:   &fullError{code: "DB_TIMEOUT", message: "pool exhausted", stack: "svc.Load (svc.go:42)\nhandler.Get (h.go:18)"},
	})
	require.NoError(t, err)
	s := string(out)
	// 有 error 時編號寬度固定 5，對齊 (error)
	assert.Contains(t, s, "(    1)")
	assert.Contains(t, s, "(error)")
	assert.Contains(t, s, "[DB_TIMEOUT]")
	assert.Contains(t, s, "at svc.Load (svc.go:42)")
}

func TestPlainFormatter_NoError_NarrowWidth(t *testing.T) {
	f := NewPlainFormatter()
	out, err := f.Format(Entry{
		Time:    testTime,
		Level:   LevelInfo,
		Message: "test",
		Args:    []any{"a", 1},
	})
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "(1)")
	assert.NotContains(t, s, "(    1)")
}

func TestPlainFormatter_ErrorNoCode(t *testing.T) {
	f := NewPlainFormatter()
	out, err := f.Format(Entry{
		Time:    testTime,
		Level:   LevelError,
		Message: "fail",
		Error:   errors.New("plain error"),
	})
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "(error)")
	assert.Contains(t, s, "plain error")
	assert.NotContains(t, s, "[") // no code brackets
}

func TestPlainFormatter_CustomTimeFormat(t *testing.T) {
	f := NewPlainFormatter(WithTimeFormat(time.RFC3339))
	out, err := f.Format(Entry{
		Time:    testTime,
		Level:   LevelInfo,
		Message: "test",
	})
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "2026-03-28T")
}

func TestPlainFormatter_NilArgsNilBound(t *testing.T) {
	f := NewPlainFormatter()
	out, err := f.Format(Entry{
		Time:    testTime,
		Level:   LevelInfo,
		Message: "bare message",
	})
	require.NoError(t, err)
	s := string(out)
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	assert.Equal(t, 1, len(lines), "no kv lines")
}
```

注意：需要在 test 檔案加 `import "errors"`（供 `TestPlainFormatter_ErrorNoCode` 使用）。

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/logs/ -run TestPlainFormatter -v`
Expected: FAIL

- [ ] **Step 3: Implement PlainFormatter**

```go
// internal/logs/format_plain.go
package logs

import (
	"fmt"
	"strconv"
	"strings"
)

const defaultPlainTimeLayout = "060102 15:04:05.000"

// PlainFormatter 將 Entry 格式化為人類可讀的 plain text。
type PlainFormatter struct {
	cfg formatterConfig
}

// NewPlainFormatter 建立 PlainFormatter。預設 time layout 為 "060102 15:04:05.000"（local）。
func NewPlainFormatter(opts ...FormatterOption) *PlainFormatter {
	cfg := applyFormatterOpts(formatterConfig{timeLayout: defaultPlainTimeLayout}, opts)
	return &PlainFormatter{cfg: cfg}
}

func (f *PlainFormatter) Format(entry Entry) ([]byte, error) {
	var b strings.Builder

	// 第一行: timestamp [LEVEL] message
	b.WriteString(entry.Time.Format(f.cfg.timeLayout))
	b.WriteString(" [")
	b.WriteString(fmt.Sprintf("%-5s", entry.Level.String()))
	b.WriteString("] ")
	b.WriteString(entry.Message)

	// 合併 kv-pairs: Bound 先、Args 後
	allKVs := mergeKVPairs(entry.Bound, entry.Args)
	hasError := entry.Error != nil

	if len(allKVs) == 0 && !hasError {
		b.WriteByte('\n')
		return []byte(b.String()), nil
	}

	// 計算編號寬度
	indexWidth := computeIndexWidth(len(allKVs), hasError)

	// 找最長 key 用於對齊
	maxKeyLen := 0
	for i := 0; i < len(allKVs)-1; i += 2 {
		if k, ok := allKVs[i].(string); ok && len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}
	if hasError && len("error") > maxKeyLen {
		// (error) label 不影響 key 對齊，但 error label 本身的寬度需考慮
	}

	// 輸出 kv-pairs
	idx := 1
	for i := 0; i < len(allKVs)-1; i += 2 {
		key := fmt.Sprintf("%v", allKVs[i])
		val := fmt.Sprintf("%v", allKVs[i+1])
		b.WriteByte('\n')
		b.WriteString("  (")
		b.WriteString(padIndex(idx, indexWidth))
		b.WriteString(") ")
		b.WriteString(padRight(key, maxKeyLen))
		b.WriteString(" : ")
		b.WriteString(val)
		idx++
	}

	// 輸出 error 區塊
	if hasError {
		info := ExtractError(entry.Error)
		b.WriteByte('\n')
		b.WriteString("  (error) ")

		if info.Code != "" {
			b.WriteByte('[')
			b.WriteString(info.Code)
			b.WriteString("] ")
		}
		b.WriteString(info.Message)

		if info.Stack != "" {
			for _, line := range strings.Split(info.Stack, "\n") {
				if line != "" {
					b.WriteString("\n    at ")
					b.WriteString(line)
				}
			}
		}
	}

	b.WriteByte('\n')
	return []byte(b.String()), nil
}

// mergeKVPairs 合併 Bound 和 Args 為單一 kv-pair slice。
func mergeKVPairs(bound, args []any) []any {
	if len(bound) == 0 && len(args) == 0 {
		return nil
	}
	result := make([]any, 0, len(bound)+len(args))
	result = append(result, bound...)
	result = append(result, args...)
	return result
}

// computeIndexWidth 計算編號寬度。有 error 時固定 5（對齊 "error"）。
func computeIndexWidth(kvCount int, hasError bool) int {
	if hasError {
		return 5
	}
	n := kvCount / 2
	if n == 0 {
		return 1
	}
	return len(strconv.Itoa(n))
}

// padIndex 將數字 i 右對齊填充到指定寬度。
func padIndex(i, width int) string {
	s := strconv.Itoa(i)
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}

// padRight 將字串右填充空白至指定寬度。
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/logs/ -run TestPlainFormatter -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/logs/format_plain.go internal/logs/format_plain_test.go
git commit -m "feat(logs): add PlainFormatter with aligned kv-pairs and error block"
```

---

### Task 7: JSONFormatter (`internal/logs/format_json.go`)

**Files:**
- Create: `internal/logs/format_json.go`
- Create: `internal/logs/format_json_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/logs/format_json_test.go
package logs

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testTimeUTC = time.Date(2026, 3, 28, 14, 5, 23, 456000000, time.UTC)

func TestJSONFormatter_BasicStructure(t *testing.T) {
	f := NewJSONFormatter()
	out, err := f.Format(Entry{
		Time:    testTimeUTC,
		Level:   LevelError,
		Message: "query failed",
	})
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))

	assert.Equal(t, "ERROR", m["level"])
	assert.Equal(t, "query failed", m["msg"])
	assert.Contains(t, m["time"], "2026-03-28T14:05:23.456")
}

func TestJSONFormatter_KVPairs(t *testing.T) {
	f := NewJSONFormatter()
	out, err := f.Format(Entry{
		Time:    testTimeUTC,
		Level:   LevelInfo,
		Message: "test",
		Bound:   []any{"service", "api"},
		Args:    []any{"user_id", 42},
	})
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))

	assert.Equal(t, "api", m["service"])
	assert.Equal(t, float64(42), m["user_id"]) // JSON numbers are float64
}

func TestJSONFormatter_KeyConflict(t *testing.T) {
	f := NewJSONFormatter()
	out, err := f.Format(Entry{
		Time:    testTimeUTC,
		Level:   LevelInfo,
		Message: "test",
		Args:    []any{"key", "first", "key", "second"},
	})
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))

	assert.Equal(t, "first", m["key"])
	assert.Equal(t, "second", m["key_2"])
}

func TestJSONFormatter_ErrorFields(t *testing.T) {
	f := NewJSONFormatter()
	out, err := f.Format(Entry{
		Time:    testTimeUTC,
		Level:   LevelError,
		Message: "fail",
		Error:   &fullError{code: "DB", message: "timeout", stack: "svc.go:1"},
	})
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))

	assert.Equal(t, "DB", m["err_code"])
	assert.Equal(t, "timeout", m["err_msg"])
	assert.Equal(t, "svc.go:1", m["err_stack"])
}

func TestJSONFormatter_ErrorFieldsPlainError(t *testing.T) {
	f := NewJSONFormatter()
	out, err := f.Format(Entry{
		Time:    testTimeUTC,
		Level:   LevelError,
		Message: "fail",
		Error:   errors.New("plain error"),
	})
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))

	assert.Nil(t, m["err_code"])
	assert.Equal(t, "plain error", m["err_msg"])
	assert.Nil(t, m["err_stack"])
}

func TestJSONFormatter_ErrorFieldsAlwaysPresent(t *testing.T) {
	f := NewJSONFormatter()
	out, err := f.Format(Entry{
		Time:    testTimeUTC,
		Level:   LevelError,
		Message: "fail",
		Error:   errors.New("x"),
	})
	require.NoError(t, err)

	// 三個 key 必定存在
	s := string(out)
	assert.Contains(t, s, "err_code")
	assert.Contains(t, s, "err_msg")
	assert.Contains(t, s, "err_stack")
}

func TestJSONFormatter_NumberPreservesType(t *testing.T) {
	f := NewJSONFormatter()
	out, err := f.Format(Entry{
		Time:    testTimeUTC,
		Level:   LevelInfo,
		Message: "test",
		Args:    []any{"count", 100, "rate", 3.14},
	})
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))

	assert.Equal(t, float64(100), m["count"])
	assert.Equal(t, 3.14, m["rate"])
}

func TestJSONFormatter_RawMessageValid(t *testing.T) {
	f := NewJSONFormatter()
	out, err := f.Format(Entry{
		Time:    testTimeUTC,
		Level:   LevelInfo,
		Message: "test",
		Args:    []any{"data", json.RawMessage(`{"nested":true}`)},
	})
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))

	nested, ok := m["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, nested["nested"])
}

func TestJSONFormatter_RawMessageInvalid(t *testing.T) {
	f := NewJSONFormatter()
	out, err := f.Format(Entry{
		Time:    testTimeUTC,
		Level:   LevelInfo,
		Message: "test",
		Args:    []any{"data", json.RawMessage(`{broken`)},
	})
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))

	// 降級為 string
	assert.IsType(t, "", m["data"])
}

func TestJSONFormatter_OddArgs(t *testing.T) {
	f := NewJSONFormatter()
	out, err := f.Format(Entry{
		Time:    testTimeUTC,
		Level:   LevelInfo,
		Message: "test",
		Args:    []any{"orphan"},
	})
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))

	assert.Equal(t, "orphan", m["_arg0"])
}

func TestJSONFormatter_CustomTimeFormat(t *testing.T) {
	f := NewJSONFormatter(WithTimeFormat(time.Kitchen))
	out, err := f.Format(Entry{
		Time:    testTimeUTC,
		Level:   LevelInfo,
		Message: "test",
	})
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(out, &m))

	assert.Equal(t, "2:05PM", m["time"])
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/logs/ -run TestJSONFormatter -v`
Expected: FAIL

- [ ] **Step 3: Implement JSONFormatter**

```go
// internal/logs/format_json.go
package logs

import (
	"encoding/json"
	"fmt"
	"strconv"
)

const defaultJSONTimeLayout = "2006-01-02T15:04:05.000Z07:00"

// JSONFormatter 將 Entry 格式化為單行 JSON。
type JSONFormatter struct {
	cfg formatterConfig
}

// NewJSONFormatter 建立 JSONFormatter。預設 time layout 為 ISO 8601 UTC。
func NewJSONFormatter(opts ...FormatterOption) *JSONFormatter {
	cfg := applyFormatterOpts(formatterConfig{timeLayout: defaultJSONTimeLayout}, opts)
	return &JSONFormatter{cfg: cfg}
}

func (f *JSONFormatter) Format(entry Entry) ([]byte, error) {
	obj := newOrderedMap()

	// 固定欄位
	obj.set("time", entry.Time.Format(f.cfg.timeLayout))
	obj.set("level", entry.Level.String())
	obj.set("msg", entry.Message)

	// kv-pairs: Bound 先、Args 後
	allKVs := mergeKVPairs(entry.Bound, entry.Args)
	f.writeKVPairs(obj, allKVs)

	// Error 欄位（三個 key 一定存在）
	if entry.Error != nil {
		info := ExtractError(entry.Error)
		if info.Code != "" {
			obj.set("err_code", info.Code)
		} else {
			obj.setRaw("err_code", []byte("null"))
		}
		obj.set("err_msg", info.Message)
		if info.Stack != "" {
			obj.set("err_stack", info.Stack)
		} else {
			obj.setRaw("err_stack", []byte("null"))
		}
	}

	data := obj.marshal()
	data = append(data, '\n')
	return data, nil
}

func (f *JSONFormatter) writeKVPairs(obj *orderedMap, kvs []any) {
	if len(kvs) == 0 {
		return
	}

	// 奇數 args → 用 index 當 key
	if len(kvs)%2 != 0 {
		internalWarn("odd number of args, using index keys", "count", len(kvs))
		for i, v := range kvs {
			key := "_arg" + strconv.Itoa(i)
			obj.set(key, f.encodeValue(v))
		}
		return
	}

	for i := 0; i < len(kvs)-1; i += 2 {
		key := fmt.Sprintf("%v", kvs[i])
		val := kvs[i+1]
		obj.setDedup(key, f.encodeValue(val))
	}
}

func (f *JSONFormatter) encodeValue(v any) any {
	switch val := v.(type) {
	case string:
		return val
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return val
	case json.RawMessage:
		if json.Valid(val) {
			return rawJSON(val)
		}
		internalWarn("invalid json.RawMessage, degrading to string", "value", string(val))
		return string(val)
	case json.Marshaler:
		data, err := val.MarshalJSON()
		if err != nil {
			internalWarn("json.Marshaler failed, degrading to string", "error", err)
			return fmt.Sprintf("%+v", v)
		}
		if !json.Valid(data) {
			internalWarn("json.Marshaler produced invalid JSON, degrading to string", "value", string(data))
			return fmt.Sprintf("%+v", v)
		}
		return rawJSON(data)
	default:
		return fmt.Sprintf("%+v", v)
	}
}

// rawJSON 包裝已驗證的 JSON bytes，避免被 json.Marshal 二次編碼。
type rawJSON []byte

func (r rawJSON) MarshalJSON() ([]byte, error) { return r, nil }

// orderedMap 維持 key 插入順序的 JSON object builder。
type orderedMap struct {
	keys   []string
	values map[string]any
}

func newOrderedMap() *orderedMap {
	return &orderedMap{values: make(map[string]any)}
}

func (m *orderedMap) set(key string, val any) {
	if _, exists := m.values[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.values[key] = val
}

func (m *orderedMap) setRaw(key string, raw []byte) {
	m.set(key, rawJSON(raw))
}

// setDedup 設定 key-value，key 重複時加 _2, _3 suffix。
func (m *orderedMap) setDedup(key string, val any) {
	if _, exists := m.values[key]; !exists {
		m.set(key, val)
		return
	}
	// key 衝突，找到可用的 suffix
	internalWarn("duplicate key in log args", "key", key)
	for i := 2; ; i++ {
		candidate := key + "_" + strconv.Itoa(i)
		if _, exists := m.values[candidate]; !exists {
			m.set(candidate, val)
			return
		}
	}
}

func (m *orderedMap) marshal() []byte {
	var buf []byte
	buf = append(buf, '{')
	for i, key := range m.keys {
		if i > 0 {
			buf = append(buf, ',')
		}
		keyBytes, _ := json.Marshal(key)
		buf = append(buf, keyBytes...)
		buf = append(buf, ':')
		valBytes, err := json.Marshal(m.values[key])
		if err != nil {
			valBytes, _ = json.Marshal(fmt.Sprintf("%+v", m.values[key]))
		}
		buf = append(buf, valBytes...)
	}
	buf = append(buf, '}')
	return buf
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/logs/ -run TestJSONFormatter -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/logs/format_json.go internal/logs/format_json_test.go
git commit -m "feat(logs): add JSONFormatter with type preservation and key dedup"
```

---

### Task 8: Internal Warn (`internal/logs/internal_warn.go`)

**Files:**
- Modify: `internal/logs/internal_warn.go` (replace stubs from Task 4)
- Create: `internal/logs/internal_warn_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/logs/internal_warn_test.go
package logs

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInternalWarn_WithWarnChain(t *testing.T) {
	var captured Entry
	terminal := func(e Entry) { captured = e }
	chain := NewChain(nil, terminal)
	SetWarnChain(chain)
	defer SetWarnChain(nil)

	internalWarn("test warning", "key", "val")

	assert.Equal(t, LevelWarn, captured.Level)
	assert.Equal(t, "test warning", captured.Message)
	assert.True(t, captured.internal)
	assert.Equal(t, []any{"key", "val"}, captured.Args)
}

func TestInternalWarn_NoWarnChain_FallsBackToStderr(t *testing.T) {
	SetWarnChain(nil)

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	internalWarn("no chain", "detail", "x")

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	assert.Contains(t, buf.String(), "no chain")
}

func TestInternalWarn_RecursionProtection(t *testing.T) {
	// Chain whose terminal always fails → triggers handleError → sees internal=true → stderr
	failSink := &Sink{
		formatter: &stubFormatter{output: nil, err: assert.AnError},
		writer:    &bytes.Buffer{},
	}
	chain := NewChain(nil, func(e Entry) { failSink.Write(e) })
	SetWarnChain(chain)
	defer SetWarnChain(nil)

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	internalWarn("will fail in sink", "k", "v")

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	s := buf.String()
	// Should see stderr fallback, not stack overflow
	assert.True(t, strings.Contains(s, "format failed") || strings.Contains(s, "LOGS_INTERNAL"))
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/logs/ -run TestInternalWarn -v`
Expected: FAIL (current stub doesn't use chain)

- [ ] **Step 3: Implement full internal_warn**

```go
// internal/logs/internal_warn.go
package logs

import (
	"fmt"
	"os"
	"time"
)

var warnChain *Chain

// SetWarnChain 設定 internal warn 使用的 chain。由 Init/ensureInit 呼叫。
func SetWarnChain(c *Chain) {
	warnChain = c
}

// internalWarn 走自身系統的 Warn chain，帶遞迴保護。
func internalWarn(msg string, kvs ...any) {
	entry := Entry{
		Time:     time.Now(),
		Level:    LevelWarn,
		Message:  msg,
		Args:     kvs,
		internal: true,
	}
	if warnChain != nil {
		warnChain.Execute(entry)
		return
	}
	stderrFallback(msg, nil)
}

// stderrFallback 是最終 fallback——直接寫 stderr，永遠不會失敗（忽略寫入錯誤）。
func stderrFallback(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "[LOGS_INTERNAL] %s: %v\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "[LOGS_INTERNAL] %s\n", msg)
	}
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/logs/ -run TestInternalWarn -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/logs/internal_warn.go internal/logs/internal_warn_test.go
git commit -m "feat(logs): implement internal warn with recursion protection and stderr fallback"
```

---

### Task 9: Output Implementations (`internal/logs/output.go`)

**Files:**
- Create: `internal/logs/output.go`
- Create: `internal/logs/output_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/logs/output_test.go
package logs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConsoleOutput_DebugInfo_Stdout(t *testing.T) {
	out := NewConsoleOutput()
	assert.Equal(t, os.Stdout, out.Resolve(ResolveContext{Level: LevelDebug}))
	assert.Equal(t, os.Stdout, out.Resolve(ResolveContext{Level: LevelInfo}))
}

func TestConsoleOutput_WarnError_Stderr(t *testing.T) {
	out := NewConsoleOutput()
	assert.Equal(t, os.Stderr, out.Resolve(ResolveContext{Level: LevelWarn}))
	assert.Equal(t, os.Stderr, out.Resolve(ResolveContext{Level: LevelError}))
}

func TestStdoutOutput(t *testing.T) {
	out := NewStdoutOutput()
	assert.Equal(t, os.Stdout, out.Resolve(ResolveContext{Level: LevelError}))
}

func TestStderrOutput(t *testing.T) {
	out := NewStderrOutput()
	assert.Equal(t, os.Stderr, out.Resolve(ResolveContext{Level: LevelDebug}))
}

func TestWriterOutput(t *testing.T) {
	w := &failWriter{} // any io.Writer
	out := NewWriterOutput(w)
	assert.Equal(t, w, out.Resolve(ResolveContext{Level: LevelInfo}))
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/logs/ -run "TestConsole|TestStdout|TestStderr|TestWriter" -v`
Expected: FAIL

- [ ] **Step 3: Implement Output**

```go
// internal/logs/output.go
package logs

import (
	"io"
	"os"
)

// ResolveContext 傳入 Output.Resolve 的參數——使用 struct 避免未來擴充改動簽名。
type ResolveContext struct {
	Level Level
}

// Output 根據 context 解析出 io.Writer。
type Output interface {
	Resolve(ctx ResolveContext) io.Writer
}

// consoleOutput: Debug+Info → stdout, Warn+Error → stderr。
type consoleOutput struct{}

func NewConsoleOutput() Output           { return &consoleOutput{} }
func (*consoleOutput) Resolve(ctx ResolveContext) io.Writer {
	if ctx.Level <= LevelInfo {
		return os.Stdout
	}
	return os.Stderr
}

type stdoutOutput struct{}

func NewStdoutOutput() Output           { return &stdoutOutput{} }
func (*stdoutOutput) Resolve(_ ResolveContext) io.Writer { return os.Stdout }

type stderrOutput struct{}

func NewStderrOutput() Output           { return &stderrOutput{} }
func (*stderrOutput) Resolve(_ ResolveContext) io.Writer { return os.Stderr }

type writerOutput struct{ w io.Writer }

func NewWriterOutput(w io.Writer) Output { return &writerOutput{w: w} }
func (o *writerOutput) Resolve(_ ResolveContext) io.Writer { return o.w }
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/logs/ -run "TestConsole|TestStdout|TestStderr|TestWriter" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/logs/output.go internal/logs/output_test.go
git commit -m "feat(logs): add Output interface with Console/Stdout/Stderr/Writer implementations"
```

---

### Task 10: Filter Handlers (`internal/logs/filter.go`)

**Files:**
- Create: `internal/logs/filter.go`
- Create: `internal/logs/filter_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/logs/filter_test.go
package logs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageFilter_Pass(t *testing.T) {
	f := NewMessageFilter(func(m string) bool { return m != "heartbeat" })
	reached := false
	f.Handle(Entry{Message: "request"}, func(e Entry) { reached = true })
	assert.True(t, reached)
}

func TestMessageFilter_Block(t *testing.T) {
	f := NewMessageFilter(func(m string) bool { return m != "heartbeat" })
	reached := false
	f.Handle(Entry{Message: "heartbeat"}, func(e Entry) { reached = true })
	assert.False(t, reached)
}

func TestKeyFilter_Pass(t *testing.T) {
	f := NewKeyFilter("env", func(v string) bool { return v == "prod" })
	reached := false
	f.Handle(Entry{Args: []any{"env", "prod"}}, func(e Entry) { reached = true })
	assert.True(t, reached)
}

func TestKeyFilter_Block(t *testing.T) {
	f := NewKeyFilter("env", func(v string) bool { return v == "prod" })
	reached := false
	f.Handle(Entry{Args: []any{"env", "dev"}}, func(e Entry) { reached = true })
	assert.False(t, reached)
}

func TestKeyFilter_KeyNotFound_Pass(t *testing.T) {
	f := NewKeyFilter("env", func(v string) bool { return v == "prod" })
	reached := false
	f.Handle(Entry{Args: []any{"other", "val"}}, func(e Entry) { reached = true })
	assert.True(t, reached, "key not found → pass through")
}

func TestKeyFilter_SearchesBoundAndArgs(t *testing.T) {
	f := NewKeyFilter("rid", func(v string) bool { return v == "abc" })
	reached := false
	f.Handle(Entry{Bound: []any{"rid", "abc"}, Args: []any{"x", 1}}, func(e Entry) { reached = true })
	assert.True(t, reached)
}

func TestMultipleFilters_AND(t *testing.T) {
	f1 := NewMessageFilter(func(m string) bool { return m == "ok" })
	f2 := NewKeyFilter("env", func(v string) bool { return v == "prod" })

	reached := false
	chain := NewChain([]Handler{f1, f2}, func(e Entry) { reached = true })

	// 兩個都通過
	chain.Execute(Entry{Message: "ok", Args: []any{"env", "prod"}})
	assert.True(t, reached)

	// 第一個不通過
	reached = false
	chain.Execute(Entry{Message: "nope", Args: []any{"env", "prod"}})
	assert.False(t, reached)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/logs/ -run "TestMessageFilter|TestKeyFilter|TestMultipleFilters" -v`
Expected: FAIL

- [ ] **Step 3: Implement Filters**

```go
// internal/logs/filter.go
package logs

import "fmt"

// MessageFilter 根據 message 內容過濾。match 回傳 true 時放行。
type MessageFilter struct {
	match func(string) bool
}

func NewMessageFilter(match func(string) bool) *MessageFilter {
	return &MessageFilter{match: match}
}

func (f *MessageFilter) Handle(entry Entry, next func(Entry)) {
	if f.match(entry.Message) {
		next(entry)
	}
}

// KeyFilter 根據 kv-pairs 中指定 key 的 value 過濾。key 不存在時放行。
type KeyFilter struct {
	key   string
	match func(string) bool
}

func NewKeyFilter(key string, match func(string) bool) *KeyFilter {
	return &KeyFilter{key: key, match: match}
}

func (f *KeyFilter) Handle(entry Entry, next func(Entry)) {
	val, found := findKeyValue(f.key, entry.Bound, entry.Args)
	if !found || f.match(val) {
		next(entry)
	}
}

// findKeyValue 在 kv-pairs 中找指定 key 的 value（Bound 先、Args 後）。
func findKeyValue(key string, bound, args []any) (string, bool) {
	for _, kvs := range [2][]any{bound, args} {
		for i := 0; i < len(kvs)-1; i += 2 {
			if fmt.Sprintf("%v", kvs[i]) == key {
				return fmt.Sprintf("%v", kvs[i+1]), true
			}
		}
	}
	return "", false
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/logs/ -run "TestMessageFilter|TestKeyFilter|TestMultipleFilters" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/logs/filter.go internal/logs/filter_test.go
git commit -m "feat(logs): add MessageFilter and KeyFilter with func(string) bool matching"
```

---

### Task 11: Enrichment Handlers (`internal/logs/enrichment.go`)

**Files:**
- Create: `internal/logs/enrichment.go`
- Create: `internal/logs/enrichment_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/logs/enrichment_test.go
package logs

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCallerEnricher_AddsCallerToFront(t *testing.T) {
	e := &CallerEnricher{}
	entry := Entry{
		caller: "service.go:42",
		Bound:  []any{"existing", "val"},
	}
	var got Entry
	e.Handle(entry, func(e Entry) { got = e })

	assert.Equal(t, "caller", got.Bound[0])
	assert.Equal(t, "service.go:42", got.Bound[1])
	assert.Equal(t, "existing", got.Bound[2])
}

func TestCallerEnricher_EmptyCaller_NoOp(t *testing.T) {
	e := &CallerEnricher{}
	entry := Entry{Bound: []any{"k", "v"}}
	var got Entry
	e.Handle(entry, func(e Entry) { got = e })

	assert.Equal(t, []any{"k", "v"}, got.Bound)
}

func TestStaticEnricher_AddsKVToFront(t *testing.T) {
	e := NewStaticEnricher("service", "api")
	entry := Entry{Bound: []any{"k", "v"}}
	var got Entry
	e.Handle(entry, func(e Entry) { got = e })

	assert.Equal(t, "service", got.Bound[0])
	assert.Equal(t, "api", got.Bound[1])
	assert.Equal(t, "k", got.Bound[2])
}

func TestEnricher_PriorityLowest(t *testing.T) {
	// Enricher 注入的放前面 → With 和 Args 同名 key 在後面 → Formatter 取後面的值
	// (此處只測 Bound 插入位置)
	enricher := NewStaticEnricher("env", "default")
	entry := Entry{
		Bound: []any{"env", "prod"},
		Args:  []any{"x", 1},
	}
	var got Entry
	enricher.Handle(entry, func(e Entry) { got = e })

	// enricher 的 "env" 在前，With 綁定的 "env" 在後
	assert.Equal(t, "env", got.Bound[0])
	assert.Equal(t, "default", got.Bound[1])
	assert.Equal(t, "env", got.Bound[2])
	assert.Equal(t, "prod", got.Bound[3])
}

func TestCallerEnricher_WithActualCaller(t *testing.T) {
	// 整合測試：驗證 caller 字串格式
	entry := Entry{caller: "enrichment_test.go:99"}
	e := &CallerEnricher{}
	var got Entry
	e.Handle(entry, func(e Entry) { got = e })

	callerVal := got.Bound[1].(string)
	assert.True(t, strings.Contains(callerVal, ".go:"))
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/logs/ -run "TestCallerEnricher|TestStaticEnricher|TestEnricher" -v`
Expected: FAIL

- [ ] **Step 3: Implement Enrichers**

```go
// internal/logs/enrichment.go
package logs

// CallerEnricher 將 entry 預捕獲的 caller 資訊注入 Bound 前端。
// caller 由 Logger.dispatch 透過 runtime.Caller 預捕獲並存入 entry.caller。
type CallerEnricher struct{}

func (c *CallerEnricher) Handle(entry Entry, next func(Entry)) {
	if entry.caller != "" {
		entry.Bound = prepend(entry.Bound, "caller", entry.caller)
	}
	next(entry)
}

// StaticEnricher 每筆 log 固定附加一組 kv-pair。
type StaticEnricher struct {
	key string
	val any
}

func NewStaticEnricher(key string, val any) *StaticEnricher {
	return &StaticEnricher{key: key, val: val}
}

func (s *StaticEnricher) Handle(entry Entry, next func(Entry)) {
	entry.Bound = prepend(entry.Bound, s.key, s.val)
	next(entry)
}

// NoCaller 是一個標記型別，Configure 時用於移除預設 Caller enricher。
// 它本身不是 Handler；config 層在組裝 chain 時處理此標記。
type NoCaller struct{}

// prepend 在 slice 前端插入 key-value pair。
func prepend(existing []any, key string, val any) []any {
	result := make([]any, 0, len(existing)+2)
	result = append(result, key, val)
	result = append(result, existing...)
	return result
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/logs/ -run "TestCallerEnricher|TestStaticEnricher|TestEnricher" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/logs/enrichment.go internal/logs/enrichment_test.go
git commit -m "feat(logs): add CallerEnricher and StaticEnricher with priority-lowest injection"
```

---

### Task 12: Logger Struct & Log Methods (`internal/logs/logger.go`)

**Files:**
- Create: `internal/logs/logger.go`
- Create: `internal/logs/logger_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/logs/logger_test.go
package logs

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureLogger 建立一個 Logger，所有 level 用 PlainFormatter + bytes.Buffer。
func captureLogger() (*Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	formatter := NewPlainFormatter()
	sink := NewSink(formatter, &buf)
	terminal := FanOut([]SinkWriter{sink})

	var chains [4]*Chain
	for i := 0; i < 4; i++ {
		chains[i] = NewChain(nil, terminal)
	}
	logger := NewLogger(chains)
	return logger, &buf
}

func TestLogger_Info_NilClosure(t *testing.T) {
	l, buf := captureLogger()
	l.Info("bare message", nil)
	assert.Contains(t, buf.String(), "bare message")
	assert.Contains(t, buf.String(), "[INFO ]")
}

func TestLogger_Info_WithClosure(t *testing.T) {
	l, buf := captureLogger()
	l.Info("test", func() []any {
		return []any{"key", "value"}
	})
	s := buf.String()
	assert.Contains(t, s, "test")
	assert.Contains(t, s, "key")
	assert.Contains(t, s, "value")
}

func TestLogger_LazyClosure_NilChain_NotExecuted(t *testing.T) {
	var chains [4]*Chain
	// Only Info has a chain, Debug does not
	var buf bytes.Buffer
	sink := NewSink(NewPlainFormatter(), &buf)
	chains[LevelInfo] = NewChain(nil, FanOut([]SinkWriter{sink}))
	l := NewLogger(chains)

	executed := false
	l.Debug("should not execute", func() []any {
		executed = true
		return nil
	})
	assert.False(t, executed, "closure should NOT execute when chain is nil")
}

func TestLogger_LazyClosure_ChainExists_Executed(t *testing.T) {
	l, _ := captureLogger()
	executed := false
	l.Info("test", func() []any {
		executed = true
		return nil
	})
	assert.True(t, executed)
}

func TestLogger_ErrorWith(t *testing.T) {
	l, buf := captureLogger()
	l.ErrorWith("query failed", func() (error, []any) {
		return &fullError{code: "DB", message: "timeout", stack: "s.go:1"}, []any{"table", "users"}
	})
	s := buf.String()
	assert.Contains(t, s, "query failed")
	assert.Contains(t, s, "(error)")
	assert.Contains(t, s, "[DB]")
	assert.Contains(t, s, "table")
}

func TestLogger_With_BindsKVPairs(t *testing.T) {
	l, buf := captureLogger()
	child := l.With("rid", "abc-123")
	child.Info("handling", nil)
	s := buf.String()
	assert.Contains(t, s, "rid")
	assert.Contains(t, s, "abc-123")
}

func TestLogger_With_DoesNotAffectParent(t *testing.T) {
	l, buf := captureLogger()
	child := l.With("rid", "abc")
	_ = child

	buf.Reset()
	l.Info("parent log", nil)
	s := buf.String()
	assert.NotContains(t, s, "rid")
}

func TestLogger_With_ChainShared(t *testing.T) {
	l, buf := captureLogger()
	child := l.With("child", true)

	buf.Reset()
	child.Info("child msg", nil)
	assert.Contains(t, buf.String(), "child msg")

	buf.Reset()
	l.Info("parent msg", nil)
	assert.Contains(t, buf.String(), "parent msg")
}

func TestLogger_With_ChainedWith(t *testing.T) {
	l, buf := captureLogger()
	child := l.With("a", 1).With("b", 2)
	child.Info("test", nil)
	s := buf.String()
	assert.Contains(t, s, "a")
	assert.Contains(t, s, "b")
}

func TestLogger_AllLevels(t *testing.T) {
	l, buf := captureLogger()

	tests := []struct {
		name string
		fn   func()
		want string
	}{
		{"Debug", func() { l.Debug("d", nil) }, "[DEBUG]"},
		{"Info", func() { l.Info("i", nil) }, "[INFO ]"},
		{"Warn", func() { l.Warn("w", nil) }, "[WARN ]"},
		{"Error", func() { l.Error("e", nil) }, "[ERROR]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.fn()
			assert.Contains(t, buf.String(), tt.want)
		})
	}
}

func TestLogger_FanOut_MultipleSinks(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	sink1 := NewSink(NewPlainFormatter(), &buf1)
	sink2 := NewSink(NewPlainFormatter(), &buf2)

	var chains [4]*Chain
	chains[LevelInfo] = NewChain(nil, FanOut([]SinkWriter{sink1, sink2}))
	l := NewLogger(chains)

	l.Info("fanout test", nil)
	assert.Contains(t, buf1.String(), "fanout test")
	assert.Contains(t, buf2.String(), "fanout test")
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/logs/ -run "TestLogger" -v`
Expected: FAIL

- [ ] **Step 3: Implement Logger**

```go
// internal/logs/logger.go
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

// --- kv-only variants ---

func (l *Logger) Debug(msg string, fn func() []any)  { l.logKV(LevelDebug, msg, fn) }
func (l *Logger) Info(msg string, fn func() []any)    { l.logKV(LevelInfo, msg, fn) }
func (l *Logger) Warn(msg string, fn func() []any)    { l.logKV(LevelWarn, msg, fn) }
func (l *Logger) Error(msg string, fn func() []any)   { l.logKV(LevelError, msg, fn) }

// --- with-error variants ---

func (l *Logger) DebugWith(msg string, fn func() (error, []any))  { l.logErr(LevelDebug, msg, fn) }
func (l *Logger) InfoWith(msg string, fn func() (error, []any))   { l.logErr(LevelInfo, msg, fn) }
func (l *Logger) WarnWith(msg string, fn func() (error, []any))   { l.logErr(LevelWarn, msg, fn) }
func (l *Logger) ErrorWith(msg string, fn func() (error, []any))  { l.logErr(LevelError, msg, fn) }

// --- internal helpers ---

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
// 呼叫鏈固定為 caller → public method → logKV/logErr → dispatch，skip=4。
func (l *Logger) dispatch(level Level, msg string, args []any, err error) {
	entry := Entry{
		Time:    time.Now(),
		Level:   level,
		Message: msg,
		Args:    args,
		Error:   err,
		Bound:   l.bound,
	}
	// 預捕獲 caller: skip=4 (runtime.Caller → dispatch → logKV/logErr → Debug/Info/... → caller)
	_, file, line, ok := runtime.Caller(4)
	if ok {
		entry.caller = filepath.Base(file) + ":" + strconv.Itoa(line)
	}
	l.chains[level].Execute(entry)
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/logs/ -run "TestLogger" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/logs/logger.go internal/logs/logger_test.go
git commit -m "feat(logs): add Logger struct with lazy closure API and pre-captured caller"
```

---

### Task 13: Package-Level API & Init (`internal/logs/logs.go`)

**Files:**
- Create: `internal/logs/logs.go`
- Create: `internal/logs/logs_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/logs/logs_test.go
package logs

import (
	"bytes"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackageLevel_Info(t *testing.T) {
	var buf bytes.Buffer
	setupTestDefault(t, &buf)

	Info("package level", nil)
	assert.Contains(t, buf.String(), "package level")
}

func TestPackageLevel_ErrorWith(t *testing.T) {
	var buf bytes.Buffer
	setupTestDefault(t, &buf)

	ErrorWith("test error", func() (error, []any) {
		return &fullError{code: "X", message: "y", stack: ""}, nil
	})
	assert.Contains(t, buf.String(), "[X]")
}

func TestPackageLevel_With(t *testing.T) {
	var buf bytes.Buffer
	setupTestDefault(t, &buf)

	child := With("rid", "abc")
	child.Info("child log", nil)
	assert.Contains(t, buf.String(), "rid")
}

func TestInit_SetsDefaultLogger(t *testing.T) {
	var buf bytes.Buffer
	sink := NewSink(NewPlainFormatter(), &buf)

	var chains [4]*Chain
	for i := 0; i < 4; i++ {
		chains[i] = NewChain(nil, FanOut([]SinkWriter{sink}))
	}

	resetForTest()
	Init(chains)
	Info("after init", nil)
	assert.Contains(t, buf.String(), "after init")
}

func TestEnsureInit_DefaultConfig(t *testing.T) {
	resetForTest()
	// ensureInit 應建立預設 logger (Plain + Console)
	// 只驗證不 panic
	Info("default config", nil)
}

// --- test helpers ---

func setupTestDefault(t *testing.T, buf *bytes.Buffer) {
	t.Helper()
	resetForTest()
	sink := NewSink(NewPlainFormatter(), buf)
	var chains [4]*Chain
	for i := 0; i < 4; i++ {
		chains[i] = NewChain(nil, FanOut([]SinkWriter{sink}))
	}
	Init(chains)
}

func resetForTest() {
	initOnce = sync.Once{}
	defaultLogger = nil
	SetWarnChain(nil)
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/logs/ -run "TestPackageLevel|TestInit|TestEnsureInit" -v`
Expected: FAIL

- [ ] **Step 3: Implement logs.go**

```go
// internal/logs/logs.go
package logs

import (
	"os"
	"sync"
)

var (
	defaultLogger *Logger
	initOnce      sync.Once
)

// Init 以指定的 chains 初始化 defaultLogger。由 pkg/logs.Configure 呼叫。
// 使用 sync.Once 保護——首次呼叫生效，後續為 no-op。
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

// defaultChains 建立預設 chains: PlainFormatter + Console + CallerEnricher。
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

// DefaultLogger 回傳當前 default logger instance。
func DefaultLogger() *Logger {
	ensureInit()
	return defaultLogger
}

// With 回傳綁定 kv-pairs 的新 Logger（透過 defaultLogger）。
func With(args ...any) *Logger {
	ensureInit()
	return defaultLogger.With(args...)
}

// --- package-level 便利函式 ---

func Debug(msg string, fn func() []any)                    { ensureInit(); defaultLogger.logKV(LevelDebug, msg, fn) }
func Info(msg string, fn func() []any)                     { ensureInit(); defaultLogger.logKV(LevelInfo, msg, fn) }
func Warn(msg string, fn func() []any)                     { ensureInit(); defaultLogger.logKV(LevelWarn, msg, fn) }
func Error(msg string, fn func() []any)                    { ensureInit(); defaultLogger.logKV(LevelError, msg, fn) }

func DebugWith(msg string, fn func() (error, []any))       { ensureInit(); defaultLogger.logErr(LevelDebug, msg, fn) }
func InfoWith(msg string, fn func() (error, []any))        { ensureInit(); defaultLogger.logErr(LevelInfo, msg, fn) }
func WarnWith(msg string, fn func() (error, []any))        { ensureInit(); defaultLogger.logErr(LevelWarn, msg, fn) }
func ErrorWith(msg string, fn func() (error, []any))       { ensureInit(); defaultLogger.logErr(LevelError, msg, fn) }
```

> **注意 caller skip**：package-level 函式直接呼叫 `defaultLogger.logKV`（非經由 `defaultLogger.Info`），所以 call depth 與 `Logger.Info → logKV → dispatch` 相同，`dispatch` 內的 `runtime.Caller(4)` 回傳正確的外部呼叫者。

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/logs/ -run "TestPackageLevel|TestInit|TestEnsureInit" -v`
Expected: PASS

- [ ] **Step 5: Run all internal/logs tests**

Run: `go test ./internal/logs/ -v`
Expected: ALL PASS

- [ ] **Step 6: Commit**

```bash
git add internal/logs/logs.go internal/logs/logs_test.go
git commit -m "feat(logs): add defaultLogger, Init, ensureInit, and package-level convenience API"
```

---

### Task 14: RotatingFileWriter (`pkg/logs/rotate.go`)

**Files:**
- Create: `pkg/logs/rotate.go`
- Create: `pkg/logs/rotate_test.go`

- [ ] **Step 1: Write failing tests**

```go
// pkg/logs/rotate_test.go
package logs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRotatingFileWriter_Write_Basic(t *testing.T) {
	dir := t.TempDir()
	w, err := NewRotatingFileWriter(filepath.Join(dir, "app"), ".log", RotateConfig{
		MaxSize: 1024,
	})
	require.NoError(t, err)
	defer w.Close()

	n, err := w.Write([]byte("hello\n"))
	require.NoError(t, err)
	assert.Equal(t, 6, n)
}

func TestRotatingFileWriter_RotatesOnSize(t *testing.T) {
	dir := t.TempDir()
	w, err := NewRotatingFileWriter(filepath.Join(dir, "app"), ".log", RotateConfig{
		MaxSize: 50, // 50 bytes trigger rotation quickly
	})
	require.NoError(t, err)
	defer w.Close()

	data := strings.Repeat("x", 30) + "\n" // 31 bytes
	w.Write([]byte(data))
	w.Write([]byte(data)) // should trigger rotation

	entries, _ := os.ReadDir(dir)
	// Should have at least 2 files: current + 1 rotated
	assert.GreaterOrEqual(t, len(entries), 2)
}

func TestRotatingFileWriter_MaxBackups(t *testing.T) {
	dir := t.TempDir()
	w, err := NewRotatingFileWriter(filepath.Join(dir, "app"), ".log", RotateConfig{
		MaxSize:    20,
		MaxBackups: 2,
	})
	require.NoError(t, err)
	defer w.Close()

	// Write enough to trigger multiple rotations
	for i := 0; i < 10; i++ {
		w.Write([]byte(strings.Repeat("x", 21) + "\n"))
	}

	// Wait for background cleanup
	w.waitCleanup()

	entries, _ := os.ReadDir(dir)
	logFiles := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".log") {
			logFiles++
		}
	}
	// current + MaxBackups rotated files
	assert.LessOrEqual(t, logFiles, 3) // 1 current + 2 backups
}

func TestRotatingFileWriter_ExtFollowsFormat(t *testing.T) {
	dir := t.TempDir()
	w, err := NewRotatingFileWriter(filepath.Join(dir, "app"), ".json", RotateConfig{
		MaxSize: 20,
	})
	require.NoError(t, err)
	defer w.Close()

	w.Write([]byte(strings.Repeat("x", 21) + "\n"))
	w.Write([]byte("y\n")) // trigger rotation

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		assert.True(t, strings.HasSuffix(e.Name(), ".json"), "ext should be .json: %s", e.Name())
	}
}

func TestRotatingFileWriter_CleanupOnlyOnRotation(t *testing.T) {
	dir := t.TempDir()
	w, err := NewRotatingFileWriter(filepath.Join(dir, "app"), ".log", RotateConfig{
		MaxSize:    1024, // large enough to not rotate
		MaxBackups: 1,
	})
	require.NoError(t, err)
	defer w.Close()

	// Create some fake old files
	os.WriteFile(filepath.Join(dir, "app.20260101-120000.log"), []byte("old"), 0644)
	os.WriteFile(filepath.Join(dir, "app.20260102-120000.log"), []byte("old"), 0644)

	// Write without rotation
	w.Write([]byte("small\n"))

	entries, _ := os.ReadDir(dir)
	oldFiles := 0
	for _, e := range entries {
		if strings.Contains(e.Name(), "20260") {
			oldFiles++
		}
	}
	assert.Equal(t, 2, oldFiles, "cleanup should NOT happen without rotation")
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./pkg/logs/ -run TestRotatingFileWriter -v`
Expected: FAIL

- [ ] **Step 3: Implement RotatingFileWriter**

```go
// pkg/logs/rotate.go
package logs

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RotateConfig 控制 log 檔案 rotation 行為。
type RotateConfig struct {
	MaxSize    int64 // 單檔上限 bytes，預設 100MB
	MaxBackups int   // 保留舊檔數量，預設 10
	MaxAge     int   // 保留天數，預設 7，0 = 不限
	Compress   bool  // 舊檔 gzip 壓縮
}

func (c RotateConfig) withDefaults() RotateConfig {
	if c.MaxSize <= 0 {
		c.MaxSize = 100 << 20 // 100MB
	}
	if c.MaxBackups <= 0 {
		c.MaxBackups = 10
	}
	if c.MaxAge < 0 {
		c.MaxAge = 7
	}
	if c.MaxAge == 0 {
		c.MaxAge = 7
	}
	return c
}

// RotatingFileWriter 實作 io.WriteCloser，支援 size-based rotation。
type RotatingFileWriter struct {
	mu       sync.Mutex
	basePath string // e.g., "/var/log/app"
	ext      string // e.g., ".log" or ".json"
	cfg      RotateConfig
	file     *os.File
	size     int64
	cleanupWg sync.WaitGroup
}

// NewRotatingFileWriter 建立並開啟 log 檔案。
func NewRotatingFileWriter(basePath, ext string, cfg RotateConfig) (*RotatingFileWriter, error) {
	cfg = cfg.withDefaults()
	w := &RotatingFileWriter{
		basePath: basePath,
		ext:      ext,
		cfg:      cfg,
	}
	if err := w.openFile(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *RotatingFileWriter) openFile() error {
	path := w.basePath + w.ext
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("stat log file: %w", err)
	}
	w.file = f
	w.size = info.Size()
	return nil
}

// Write 寫入資料。超過 MaxSize 時觸發 rotation。
func (w *RotatingFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.size+int64(len(p)) > w.cfg.MaxSize {
		if err := w.rotate(); err != nil {
			return 0, fmt.Errorf("rotate: %w", err)
		}
	}

	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *RotatingFileWriter) rotate() error {
	if w.file != nil {
		w.file.Close()
	}

	// Rename current → timestamped
	current := w.basePath + w.ext
	ts := time.Now().Format("20060102-150405")
	rotated := w.basePath + "." + ts + w.ext
	if err := os.Rename(current, rotated); err != nil {
		return err
	}

	// Open new file
	if err := w.openFile(); err != nil {
		return err
	}

	// Background cleanup
	w.cleanupWg.Add(1)
	go func() {
		defer w.cleanupWg.Done()
		w.cleanup()
	}()

	return nil
}

func (w *RotatingFileWriter) cleanup() {
	pattern := w.basePath + ".*" + w.ext
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	// Sort by name (timestamp order)
	sort.Strings(matches)

	// Also check .gz variants
	if w.cfg.Compress {
		gzPattern := w.basePath + ".*" + w.ext + ".gz"
		gzMatches, _ := filepath.Glob(gzPattern)
		matches = append(matches, gzMatches...)
		sort.Strings(matches)
	}

	// Remove excess backups
	if len(matches) > w.cfg.MaxBackups {
		excess := matches[:len(matches)-w.cfg.MaxBackups]
		for _, f := range excess {
			os.Remove(f)
		}
		matches = matches[len(matches)-w.cfg.MaxBackups:]
	}

	// Remove by age
	if w.cfg.MaxAge > 0 {
		cutoff := time.Now().AddDate(0, 0, -w.cfg.MaxAge)
		for _, f := range matches {
			info, err := os.Stat(f)
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				os.Remove(f)
			}
		}
	}

	// Compress uncompressed backups
	if w.cfg.Compress {
		remaining, _ := filepath.Glob(w.basePath + ".*" + w.ext)
		for _, f := range remaining {
			if strings.HasSuffix(f, ".gz") {
				continue
			}
			w.compressFile(f)
		}
	}
}

func (w *RotatingFileWriter) compressFile(path string) {
	src, err := os.Open(path)
	if err != nil {
		return
	}
	defer src.Close()

	dst, err := os.Create(path + ".gz")
	if err != nil {
		return
	}
	defer dst.Close()

	gz := gzip.NewWriter(dst)
	if _, err := io.Copy(gz, src); err != nil {
		gz.Close()
		os.Remove(path + ".gz")
		return
	}
	gz.Close()
	os.Remove(path) // remove uncompressed
}

// Close 關閉檔案。
func (w *RotatingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// waitCleanup 等待背景清理完成（僅供測試使用）。
func (w *RotatingFileWriter) waitCleanup() {
	w.cleanupWg.Wait()
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./pkg/logs/ -run TestRotatingFileWriter -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/logs/rotate.go pkg/logs/rotate_test.go
git commit -m "feat(logs): add RotatingFileWriter with size-based rotation and background cleanup"
```

---

### Task 15: pkg/logs Builder Re-exports

**Files:**
- Create: `pkg/logs/output.go`
- Create: `pkg/logs/filter.go`
- Create: `pkg/logs/enrichment.go`
- Create: `pkg/logs/formatter.go`

這四個檔案是 thin wrapper，將 `internal/logs` 的型別和建構函式以 public API 形式曝露。

- [ ] **Step 1: Create output.go**

```go
// pkg/logs/output.go
package logs

import (
	"io"

	ilogs "golan-example/internal/logs"
)

// Output 根據 context 解析出 io.Writer。
type Output = ilogs.Output

// ResolveContext 傳入 Output.Resolve 的參數。
type ResolveContext = ilogs.ResolveContext

// ToConsole 回傳 Console output：Debug+Info → stdout, Warn+Error → stderr。
func ToConsole() Output { return ilogs.NewConsoleOutput() }

// ToStdout 回傳一律寫入 stdout 的 output。
func ToStdout() Output { return ilogs.NewStdoutOutput() }

// ToStderr 回傳一律寫入 stderr 的 output。
func ToStderr() Output { return ilogs.NewStderrOutput() }

// ToWriter 回傳寫入指定 io.Writer 的 output。
func ToWriter(w io.Writer) Output { return ilogs.NewWriterOutput(w) }

// ToFile 回傳 RotatingFileWriter output。ext 由 Formatter 決定（由 Pipe 內部自動處理）。
func ToFile(name string, cfg RotateConfig) Output {
	return &fileOutput{name: name, cfg: cfg}
}

// fileOutput 延遲建立 RotatingFileWriter 直到 Resolve 被呼叫（需要知道 ext）。
type fileOutput struct {
	name string
	cfg  RotateConfig
	ext  string // set by config layer
}

func (o *fileOutput) Resolve(_ ilogs.ResolveContext) io.Writer {
	ext := o.ext
	if ext == "" {
		ext = ".log"
	}
	w, err := NewRotatingFileWriter(o.name, ext, o.cfg)
	if err != nil {
		// Fallback to stderr on file open failure
		ilogs.Warn("failed to open log file, falling back to stderr", func() []any {
			return []any{"path", o.name + ext, "error", err}
		})
		return ilogs.NewStderrOutput().Resolve(ilogs.ResolveContext{})
	}
	return w
}
```

- [ ] **Step 2: Create filter.go**

```go
// pkg/logs/filter.go
package logs

import ilogs "golan-example/internal/logs"

// Filter 是 Handler 的子集，負責決定 entry 是否放行。
type Filter = ilogs.Handler

// FilterByKey 根據 kv-pairs 中指定 key 的 value 過濾。key 不存在時放行。
func FilterByKey(key string, match func(string) bool) Filter {
	return ilogs.NewKeyFilter(key, match)
}

// FilterByMessage 根據 message 內容過濾。match 回傳 true 時放行。
func FilterByMessage(match func(string) bool) Filter {
	return ilogs.NewMessageFilter(match)
}
```

- [ ] **Step 3: Create enrichment.go**

```go
// pkg/logs/enrichment.go
package logs

import ilogs "golan-example/internal/logs"

// Enricher 是 Handler 的子集，負責注入額外 kv-pairs。
type Enricher = ilogs.Handler

// Caller 回傳 CallerEnricher（runtime.Caller 取單一 frame）。預設啟用。
func Caller() Enricher { return &ilogs.CallerEnricher{} }

// Static 回傳固定 kv-pair 的 enricher。
func Static(key string, val any) Enricher { return ilogs.NewStaticEnricher(key, val) }

// NoCaller 回傳 NoCaller 標記。config 層用此移除預設 Caller enricher。
func NoCaller() noCaller { return noCaller{} }

// noCaller 是標記型別，config.go 在組裝 chain 時辨識此型別。
type noCaller struct{}
```

- [ ] **Step 4: Create formatter.go**

```go
// pkg/logs/formatter.go
package logs

import ilogs "golan-example/internal/logs"

// Formatter 將 Entry 格式化為 byte slice。
type Formatter = ilogs.Formatter

// FormatterOption 用於自訂 Formatter 行為。
type FormatterOption = ilogs.FormatterOption

// WithTimeFormat 設定 time 輸出格式（Go time layout string）。
func WithTimeFormat(layout string) FormatterOption {
	return ilogs.WithTimeFormat(layout)
}

// Plain 建立 PlainFormatter。預設 time layout: "060102 15:04:05.000"（local）。
func Plain(opts ...FormatterOption) Formatter {
	return ilogs.NewPlainFormatter(opts...)
}

// JSON 建立 JSONFormatter。預設 time layout: ISO 8601 UTC。
func JSON(opts ...FormatterOption) Formatter {
	return ilogs.NewJSONFormatter(opts...)
}

// formatterExt 回傳 formatter 對應的檔案副檔名。
func formatterExt(f Formatter) string {
	switch f.(type) {
	case *ilogs.JSONFormatter:
		return ".json"
	default:
		return ".log"
	}
}
```

- [ ] **Step 5: Verify compilation**

Run: `go build ./pkg/logs/...`
Expected: success（可能需要先建 config.go stub）

建立 config.go stub 讓編譯通過：

```go
// pkg/logs/config.go（stub — Task 16 完成實作）
package logs
```

- [ ] **Step 6: Commit**

```bash
git add pkg/logs/output.go pkg/logs/filter.go pkg/logs/enrichment.go pkg/logs/formatter.go pkg/logs/config.go
git commit -m "feat(logs): add pkg/logs builder re-exports for Output, Filter, Enrichment, Formatter"
```

---

### Task 16: Configure DSL (`pkg/logs/config.go`)

**Files:**
- Modify: `pkg/logs/config.go` (replace stub)
- Create: `pkg/logs/config_test.go`

- [ ] **Step 1: Write failing tests**

```go
// pkg/logs/config_test.go
package logs

import (
	"bytes"
	"sync"
	"testing"

	ilogs "golan-example/internal/logs"

	"github.com/stretchr/testify/assert"
)

func resetLogsForTest() {
	configureOnce = sync.Once{}
	ilogs.ResetForTest() // 需要在 internal/logs 加一個 test helper
}

func TestConfigure_GlobalPipe(t *testing.T) {
	resetLogsForTest()
	var buf bytes.Buffer

	Configure(
		Pipe(Plain(), ToWriter(&buf)),
	)

	ilogs.Info("hello", nil)
	assert.Contains(t, buf.String(), "hello")
}

func TestConfigure_PerLevel_ForError(t *testing.T) {
	resetLogsForTest()
	var infoBuf, errorBuf bytes.Buffer

	Configure(
		Pipe(Plain(), ToWriter(&infoBuf)),
		ForError(
			Pipe(Plain(), ToWriter(&errorBuf)),
		),
	)

	ilogs.Info("info msg", nil)
	ilogs.Error("error msg", nil)

	assert.Contains(t, infoBuf.String(), "info msg")
	assert.Contains(t, infoBuf.String(), "error msg") // global applies to all
	assert.Contains(t, errorBuf.String(), "error msg")
	assert.NotContains(t, errorBuf.String(), "info msg")
}

func TestConfigure_NoInherit(t *testing.T) {
	resetLogsForTest()
	var globalBuf, debugBuf bytes.Buffer

	Configure(
		Pipe(Plain(), ToWriter(&globalBuf)),
		ForDebug(
			NoInherit(),
			Pipe(Plain(), ToWriter(&debugBuf)),
		),
	)

	ilogs.Debug("debug msg", nil)
	ilogs.Info("info msg", nil)

	assert.Contains(t, debugBuf.String(), "debug msg")
	assert.NotContains(t, globalBuf.String(), "debug msg") // NoInherit blocks global
	assert.Contains(t, globalBuf.String(), "info msg")
}

func TestConfigure_NoCaller(t *testing.T) {
	resetLogsForTest()
	var buf bytes.Buffer

	Configure(
		NoCaller(),
		Pipe(Plain(), ToWriter(&buf)),
	)

	ilogs.Info("test", nil)
	assert.NotContains(t, buf.String(), "caller")
}

func TestConfigure_CallerDefault(t *testing.T) {
	resetLogsForTest()
	var buf bytes.Buffer

	Configure(
		Pipe(Plain(), ToWriter(&buf)),
	)

	ilogs.Info("test", nil)
	assert.Contains(t, buf.String(), "caller")
}

func TestConfigure_WithFilter(t *testing.T) {
	resetLogsForTest()
	var buf bytes.Buffer

	Configure(
		WithFilter(FilterByMessage(func(m string) bool { return m != "skip" })),
		Pipe(Plain(), ToWriter(&buf)),
	)

	ilogs.Info("keep", nil)
	ilogs.Info("skip", nil)

	assert.Contains(t, buf.String(), "keep")
	assert.NotContains(t, buf.String(), "skip")
}

func TestConfigure_OnceOnly(t *testing.T) {
	resetLogsForTest()
	var buf1, buf2 bytes.Buffer

	Configure(Pipe(Plain(), ToWriter(&buf1)))
	Configure(Pipe(Plain(), ToWriter(&buf2))) // should be no-op

	ilogs.Info("test", nil)
	assert.Contains(t, buf1.String(), "test")
	assert.Empty(t, buf2.String())
}
```

- [ ] **Step 2: Add ResetForTest helper to internal/logs**

在 `internal/logs/logs.go` 底部加：

```go
// ResetForTest 重設 internal state（僅供測試使用）。
func ResetForTest() {
	initOnce = sync.Once{}
	defaultLogger = nil
	SetWarnChain(nil)
}
```

- [ ] **Step 3: Run tests to verify failure**

Run: `go test ./pkg/logs/ -run TestConfigure -v`
Expected: FAIL

- [ ] **Step 4: Implement Configure DSL**

```go
// pkg/logs/config.go
package logs

import (
	"sync"

	ilogs "golan-example/internal/logs"
)

var configureOnce sync.Once

// Option 是 Configure 的設定選項。
type Option func(*config)

type config struct {
	global levelConfig
	levels [4]*levelConfig // non-nil = ForXxx 被呼叫
}

type levelConfig struct {
	filters   []ilogs.Handler
	enrichers []ilogs.Handler
	pipes     []pipeConfig
	noCaller  bool
	noInherit bool
}

type pipeConfig struct {
	formatter ilogs.Formatter
	output    Output
}

// Configure 設定 logging 輸出行為。sync.Once 保護，僅首次呼叫生效。
func Configure(opts ...Option) {
	configureOnce.Do(func() {
		c := &config{}
		for _, o := range opts {
			o(c)
		}
		chains := buildChains(c)
		ilogs.Init(chains)
	})
}

// --- Option builders ---

// WithFilter 加入全局 filter。
func WithFilter(filters ...Filter) Option {
	return func(c *config) {
		c.global.filters = append(c.global.filters, filters...)
	}
}

// WithEnrichment 加入全局 enrichment。
func WithEnrichment(enrichers ...Enricher) Option {
	return func(c *config) {
		c.global.enrichers = append(c.global.enrichers, enrichers...)
	}
}

// Pipe 加入全局 Formatter + Output 組合（一個 Sink）。
func Pipe(f Formatter, o Output) Option {
	return func(c *config) {
		c.global.pipes = append(c.global.pipes, pipeConfig{formatter: f, output: o})
	}
}

// ForDebug 設定 Debug level 的追加選項。
func ForDebug(opts ...Option) Option { return forLevel(ilogs.LevelDebug, opts) }

// ForInfo 設定 Info level 的追加選項。
func ForInfo(opts ...Option) Option { return forLevel(ilogs.LevelInfo, opts) }

// ForWarn 設定 Warn level 的追加選項。
func ForWarn(opts ...Option) Option { return forLevel(ilogs.LevelWarn, opts) }

// ForError 設定 Error level 的追加選項。
func ForError(opts ...Option) Option { return forLevel(ilogs.LevelError, opts) }

func forLevel(level ilogs.Level, opts []Option) Option {
	return func(c *config) {
		if c.levels[level] == nil {
			c.levels[level] = &levelConfig{}
		}
		// 用 sub-config 收集 level-scope options
		sub := &config{}
		for _, o := range opts {
			o(sub)
		}
		lc := c.levels[level]
		lc.filters = append(lc.filters, sub.global.filters...)
		lc.enrichers = append(lc.enrichers, sub.global.enrichers...)
		lc.pipes = append(lc.pipes, sub.global.pipes...)
		if sub.global.noCaller {
			lc.noCaller = true
		}
		if sub.global.noInherit {
			lc.noInherit = true
		}
	}
}

// NoInherit 讓該 level 不繼承全局設定。只在 ForXxx 內有效。
func NoInherit() Option {
	return func(c *config) {
		c.global.noInherit = true
	}
}

// --- chain building ---

func buildChains(c *config) [4]*ilogs.Chain {
	var chains [4]*ilogs.Chain
	for i := 0; i < 4; i++ {
		level := ilogs.Level(i)
		lc := c.levels[i]

		var merged levelConfig
		if lc != nil && lc.noInherit {
			merged = *lc
		} else {
			merged = mergeConfigs(c.global, lc)
		}

		// 組裝 handlers: filters → enrichers
		var handlers []ilogs.Handler
		handlers = append(handlers, merged.filters...)

		// Caller enricher: 預設啟用，除非 noCaller
		if !merged.noCaller {
			handlers = append(handlers, &ilogs.CallerEnricher{})
		}

		handlers = append(handlers, merged.enrichers...)

		// 組裝 sinks
		if len(merged.pipes) == 0 {
			// 無 pipe → 此 level 無輸出
			continue
		}

		sinks := make([]ilogs.SinkWriter, 0, len(merged.pipes))
		for _, p := range merged.pipes {
			writer := p.output.Resolve(ilogs.ResolveContext{Level: level})
			// Set ext for fileOutput
			if fo, ok := p.output.(*fileOutput); ok {
				fo.ext = formatterExt(p.formatter)
			}
			sinks = append(sinks, ilogs.NewSink(p.formatter, writer))
		}

		chains[i] = ilogs.NewChain(handlers, ilogs.FanOut(sinks))
	}
	return chains
}

func mergeConfigs(global levelConfig, level *levelConfig) levelConfig {
	merged := levelConfig{
		filters:   append([]ilogs.Handler{}, global.filters...),
		enrichers: append([]ilogs.Handler{}, global.enrichers...),
		pipes:     append([]pipeConfig{}, global.pipes...),
		noCaller:  global.noCaller,
	}
	if level != nil {
		merged.filters = append(merged.filters, level.filters...)
		merged.enrichers = append(merged.enrichers, level.enrichers...)
		merged.pipes = append(merged.pipes, level.pipes...)
		if level.noCaller {
			merged.noCaller = true
		}
	}
	return merged
}
```

- [ ] **Step 5: Run tests to verify pass**

Run: `go test ./pkg/logs/ -run TestConfigure -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/logs/config.go pkg/logs/config_test.go internal/logs/logs.go
git commit -m "feat(logs): add Configure DSL with per-level config, merge rules, and NoCaller/NoInherit"
```

---

### Task 17: Integration Verification

**Files:**
- Modify: `README.md` (update project structure)

- [ ] **Step 1: Verify full compilation**

Run: `go build ./...`
Expected: zero errors

- [ ] **Step 2: Run all tests**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 3: Run static analysis**

Run: `go vet ./...`
Expected: zero warnings

- [ ] **Step 4: Verify old logger is gone**

Run: `grep -r "pkg/logger" --include="*.go" .`
Expected: zero matches

- [ ] **Step 5: Update README.md project structure**

更新 README.md 的專案結構區塊，加入 `internal/logs/` 和 `pkg/logs/`，移除 `pkg/logger/`。

```
├── internal/             # 私有程式碼（不可被外部 import）
│   ├── logs/             # Logging 引擎（Entry, Handler chain, Formatters, Logger）
│   ├── config/           # 應用程式設定
│   ├── handler/          # HTTP 處理器
│   ├── service/          # 商業邏輯層
│   └── repository/       # 資料存取層
├── pkg/                  # 可被外部 import 的共用套件
│   ├── errs/             # 錯誤處理（error code + stack trace + cause chain）
│   │   ├── errs.go       # Error 型別、New/Newf 建構與 fmt.Formatter 實作
│   │   ├── stack.go      # Frame/Stack 型別與 call stack 捕獲
│   │   └── wrap.go       # Wrap/Wrapf 包裝既有 error
│   └── logs/             # Logging 設定入口（Configure DSL + RotatingFileWriter）
│       ├── config.go     # Configure()、Option、ForXxx、Pipe、WithFilter、WithEnrichment
│       ├── rotate.go     # RotatingFileWriter、RotateConfig
│       └── ...           # Output/Filter/Enrichment/Formatter builder re-exports
```

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat(logs): complete logging module integration and update README"
```

---

## Caller Skip Validation Note

`Logger.dispatch` 使用 `runtime.Caller(4)` 預捕獲呼叫者位置。呼叫鏈固定為：

```
runtime.Caller (skip=0)
  ← dispatch (skip=1)
    ← logKV/logErr (skip=2)
      ← Debug/Info/Warn/Error/... (skip=3)
        ← actual caller (skip=4) ✓
```

Package-level 便利函式直接呼叫 `defaultLogger.logKV`（不經由 `defaultLogger.Info`），因此與 `Logger.Info → logKV → dispatch` 的深度相同：

```
runtime.Caller (skip=0)
  ← dispatch (skip=1)
    ← logKV (skip=2)
      ← package-level Info (skip=3)
        ← actual caller (skip=4) ✓
```

Task 13 的 `TestPackageLevel` 和 Task 12 的 `TestCallerEnricher_WithActualCaller` 會驗證此行為。若 skip 不正確，caller 會指向錯誤的檔案/行號，測試會捕捉到。
