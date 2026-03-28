package errs

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- capture() 白箱測試 ----

// TestCaptureReturnsFrames 驗證 capture 在正常情況下回傳非空 Stack，
// 且第一個 frame 指向呼叫端。
func TestCaptureReturnsFrames(t *testing.T) {
	// skip=1: runtime.Callers → capture → 呼叫端（本函式）
	st := capture(1)
	require.NotEmpty(t, st)

	first := st[0]
	assert.Contains(t, first.Function, "capture")
	assert.Equal(t, "stack.go", first.File)
	assert.Greater(t, first.Line, 0)
}

// TestCaptureSkipAll 驗證當 skip 超過實際 stack 深度時回傳 nil。
// 此測試覆蓋 capture() 中 n == 0 的防禦性分支。
func TestCaptureSkipAll(t *testing.T) {
	// 跳過極大數量的 frame，使 runtime.Callers 回傳 0
	st := capture(9999)
	assert.Nil(t, st)
}

// TestCaptureSkipCalibration 驗證不同 skip 值產生的第一個 frame 符合預期。
func TestCaptureSkipCalibration(t *testing.T) {
	tests := []struct {
		name     string
		skip     int
		wantFunc string // 預期第一個 frame 包含的函式名片段
	}{
		{
			name:     "skip_1_is_capture_itself",
			skip:     1,
			wantFunc: "capture",
		},
		{
			name:     "skip_2_is_caller",
			skip:     2,
			wantFunc: "TestCaptureSkipCalibration",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := capture(tt.skip)
			require.NotEmpty(t, st)
			assert.Contains(t, st[0].Function, tt.wantFunc)
		})
	}
}

// TestCaptureBasename 驗證 capture 儲存的檔名一律為 basename（不含路徑）。
func TestCaptureBasename(t *testing.T) {
	st := capture(2)
	require.NotEmpty(t, st)
	for i, f := range st {
		assert.NotContains(t, f.File, "/",
			"frame %d File 不得包含 /", i)
		assert.NotContains(t, f.File, "\\",
			"frame %d File 不得包含 \\", i)
	}
}

// ---- writeStack() 白箱測試 ----

// TestWriteStackEmpty 驗證空 Stack 不輸出任何內容。
func TestWriteStackEmpty(t *testing.T) {
	var buf bytes.Buffer
	writeStack(&buf, nil)
	assert.Empty(t, buf.String())

	buf.Reset()
	writeStack(&buf, Stack{})
	assert.Empty(t, buf.String())
}

// TestWriteStackSingleFrame 驗證單一 frame 輸出格式。
func TestWriteStackSingleFrame(t *testing.T) {
	var buf bytes.Buffer
	writeStack(&buf, Stack{
		{Function: "main.run", File: "main.go", Line: 42},
	})
	assert.Equal(t, "\n    at main.run (main.go:42)", buf.String())
}

// TestWriteStackMultipleFrames 驗證多個 frame 的輸出順序與格式。
func TestWriteStackMultipleFrames(t *testing.T) {
	var buf bytes.Buffer
	writeStack(&buf, Stack{
		{Function: "pkg.A", File: "a.go", Line: 10},
		{Function: "pkg.B", File: "b.go", Line: 20},
		{Function: "pkg.C", File: "c.go", Line: 30},
	})
	lines := strings.Split(buf.String(), "\n")
	// 第一行為空（因為開頭有 \n），後面三行是 frames
	require.Len(t, lines, 4)
	assert.Equal(t, "", lines[0])
	assert.Equal(t, "    at pkg.A (a.go:10)", lines[1])
	assert.Equal(t, "    at pkg.B (b.go:20)", lines[2])
	assert.Equal(t, "    at pkg.C (c.go:30)", lines[3])
}

// ---- writeCause() 白箱測試 ----

// customUnwrapError 實作 Unwrap() 的自訂 error，用於驗證 writeCause 能正確
// 走訪非 *Error 的 Unwrap chain。
type customUnwrapError struct {
	msg   string
	inner error
}

func (e *customUnwrapError) Error() string { return e.msg }
func (e *customUnwrapError) Unwrap() error { return e.inner }

// TestWriteCauseNil 驗證 cause 為 nil 時不輸出任何內容。
func TestWriteCauseNil(t *testing.T) {
	var buf bytes.Buffer
	writeCause(&buf, nil)
	assert.Empty(t, buf.String())
}

// TestWriteCausePlainError 驗證一般 error cause 僅輸出 message，無 stack。
func TestWriteCausePlainError(t *testing.T) {
	var buf bytes.Buffer
	writeCause(&buf, errors.New("disk full"))

	output := buf.String()
	assert.Equal(t, "\nCaused by: disk full", output)
	assert.NotContains(t, output, "at ")
}

// TestWriteCauseErrsError 驗證 *Error cause 包含完整的 code、message 與 stack。
func TestWriteCauseErrsError(t *testing.T) {
	inner := &Error{
		code:    "IO",
		message: "read failed",
		stack: Stack{
			{Function: "os.Read", File: "os.go", Line: 100},
		},
	}

	var buf bytes.Buffer
	writeCause(&buf, inner)

	output := buf.String()
	assert.Contains(t, output, "Caused by: [IO] read failed")
	assert.Contains(t, output, "at os.Read (os.go:100)")
}

// TestWriteCauseMixedChain 驗證 *Error → plain error → *Error 的混合 chain
// 能正確輸出每層 cause。
func TestWriteCauseMixedChain(t *testing.T) {
	root := &Error{
		code:    "ROOT",
		message: "root cause",
		stack:   Stack{{Function: "root.Fn", File: "root.go", Line: 1}},
	}
	middle := &customUnwrapError{msg: "middleware timeout", inner: root}
	top := &Error{
		code:    "TOP",
		message: "top level",
		cause:   middle,
		stack:   Stack{{Function: "top.Fn", File: "top.go", Line: 99}},
	}

	var buf bytes.Buffer
	// writeCause 從 top.cause 開始走訪，不含 top 自身
	writeCause(&buf, top.cause)

	output := buf.String()
	// 第一層 cause: plain error (customUnwrapError)
	assert.Contains(t, output, "Caused by: middleware timeout")
	// 第二層 cause: *Error
	assert.Contains(t, output, "Caused by: [ROOT] root cause")
	assert.Contains(t, output, "at root.Fn (root.go:1)")
}

// ---- writeVerbose() 白箱測試 ----

// TestWriteVerboseManualError 驗證 writeVerbose 對手動建構的 Error 輸出
// 正確格式（不依賴 New/Wrap）。
func TestWriteVerboseManualError(t *testing.T) {
	e := &Error{
		code:    "MANUAL",
		message: "test",
		stack: Stack{
			{Function: "x.Fn", File: "x.go", Line: 7},
		},
	}

	var buf bytes.Buffer
	writeVerbose(&buf, e)

	output := buf.String()
	assert.Contains(t, output, "[MANUAL] test")
	assert.Contains(t, output, "at x.Fn (x.go:7)")
	assert.NotContains(t, output, "Caused by:")
}

// TestWriteVerboseWithCause 驗證 writeVerbose 含 cause 時輸出 chain。
func TestWriteVerboseWithCause(t *testing.T) {
	inner := errors.New("raw io error")
	e := &Error{
		code:    "WRAP",
		message: "operation failed",
		cause:   inner,
		stack: Stack{
			{Function: "svc.Do", File: "svc.go", Line: 55},
		},
	}

	var buf bytes.Buffer
	writeVerbose(&buf, e)

	output := buf.String()
	assert.Contains(t, output, "[WRAP] operation failed")
	assert.Contains(t, output, "at svc.Do (svc.go:55)")
	assert.Contains(t, output, "Caused by: raw io error")
}

// TestWriteVerboseEmptyStack 驗證 Error 無 stack（nil）時不 panic 且不輸出
// "at" 行。
func TestWriteVerboseEmptyStack(t *testing.T) {
	e := &Error{
		code:    "NO_STACK",
		message: "no stack",
		stack:   nil,
	}

	var buf bytes.Buffer
	writeVerbose(&buf, e)

	output := buf.String()
	assert.Equal(t, "[NO_STACK] no stack", output)
	assert.NotContains(t, output, "at ")
}

// ---- Error struct 內部欄位邊界測試 ----

// TestErrorEmptyFields 驗證 code 或 message 為空字串時的行為。
// 空 code 在契約上不建議，但實作不得 panic。
func TestErrorEmptyFields(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		message string
		want    string
	}{
		{"empty_code", "", "msg", "[] msg"},
		{"empty_message", "CODE", "", "[CODE] "},
		{"both_empty", "", "", "[] "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Error{code: tt.code, message: tt.message}
			assert.Equal(t, tt.want, e.Error())
			assert.Equal(t, tt.code, e.Code())
			assert.Equal(t, tt.message, e.Message())
		})
	}
}

// TestErrorZeroValue 驗證 Error 的零值（非 nil pointer）可安全使用。
func TestErrorZeroValue(t *testing.T) {
	e := &Error{}

	assert.Equal(t, "", e.Code())
	assert.Equal(t, "", e.Message())
	assert.Equal(t, "[] ", e.Error())
	assert.Nil(t, e.Unwrap())
	assert.Empty(t, e.StackTrace())
}

// TestStackTraceEmptyStack 驗證 stack 為空 slice 時 StackTrace() 回傳空 slice
// 而非 nil。
func TestStackTraceEmptyStack(t *testing.T) {
	e := &Error{stack: Stack{}}
	st := e.StackTrace()
	assert.NotNil(t, st)
	assert.Empty(t, st)
}

// TestDirectFieldAccess 驗證 New 建構出的 Error 內部欄位全部正確設定。
func TestDirectFieldAccess(t *testing.T) {
	e := New("CODE", "msg")

	assert.Equal(t, "CODE", e.code)
	assert.Equal(t, "msg", e.message)
	assert.Nil(t, e.cause)
	assert.NotEmpty(t, e.stack)
}

// TestWrapDirectFieldAccess 驗證 Wrap 建構出的 Error 內部 cause 欄位正確指向
// 被包裝的原始 error。
func TestWrapDirectFieldAccess(t *testing.T) {
	original := errors.New("original")
	e := Wrap(original, "W", "wrapped")

	assert.Equal(t, "W", e.code)
	assert.Equal(t, "wrapped", e.message)
	assert.Same(t, original, e.cause)
	assert.NotEmpty(t, e.stack)
}
