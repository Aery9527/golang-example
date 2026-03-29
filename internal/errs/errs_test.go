package errs_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golan-example/internal/errs"
)

// E1: New 建立含 code + 原始 message + stack 的 error。
func TestNew(t *testing.T) {
	err := errs.New("NOT_FOUND", "user not found")

	require.NotNil(t, err)
	assert.Equal(t, "NOT_FOUND", err.Code())
	assert.Equal(t, "user not found", err.Message())
	assert.Equal(t, "[NOT_FOUND] user not found", err.Error())
	assert.NotEmpty(t, err.StackTrace())
}

// E2: Newf 支援 format string 插值。
func TestNewf(t *testing.T) {
	err := errs.Newf("INVALID", "field %s is %d", "age", -1)

	require.NotNil(t, err)
	assert.Equal(t, "INVALID", err.Code())
	assert.Equal(t, "field age is -1", err.Message())
	assert.Equal(t, "[INVALID] field age is -1", err.Error())
	assert.NotEmpty(t, err.StackTrace())
}

// E3: Wrap 保留 cause chain；errors.Is 可走訪。
func TestWrap(t *testing.T) {
	t.Run("plain_error_cause", func(t *testing.T) {
		original := errors.New("connection refused")
		wrapped := errs.Wrap(original, "DB_FAIL", "query failed")

		require.NotNil(t, wrapped)
		assert.Equal(t, "DB_FAIL", wrapped.Code())
		assert.Equal(t, "query failed", wrapped.Message())
		assert.True(t, errors.Is(wrapped, original))
		assert.NotEmpty(t, wrapped.StackTrace())
	})

	t.Run("errs_error_cause", func(t *testing.T) {
		inner := errs.New("INNER", "inner error")
		outer := errs.Wrap(inner, "OUTER", "outer error")

		require.NotNil(t, outer)
		assert.True(t, errors.Is(outer, inner))

		var target *errs.Error
		require.True(t, errors.As(outer, &target))
		assert.Equal(t, "OUTER", target.Code())
	})
}

// E4: Wrapf 支援 format string 並保留 chain。
func TestWrapf(t *testing.T) {
	original := errors.New("timeout")
	wrapped := errs.Wrapf(original, "TIMEOUT", "request to %s timed out", "api.example.com")

	require.NotNil(t, wrapped)
	assert.Equal(t, "TIMEOUT", wrapped.Code())
	assert.Equal(t, "request to api.example.com timed out", wrapped.Message())
	assert.True(t, errors.Is(wrapped, original))
}

// E5: Code() 與 Message() 回傳原始欄位供下游消費者使用。
func TestAccessors(t *testing.T) {
	err := errs.New("CODE_X", "raw message")

	assert.Equal(t, "CODE_X", err.Code())
	assert.Equal(t, "raw message", err.Message())
	assert.Equal(t, "[CODE_X] raw message", err.Error())

	// Message 不得包含 [CODE] 格式
	assert.NotContains(t, err.Message(), "[")
	assert.NotContains(t, err.Message(), "]")
}

// E6: StackTrace() 回傳防禦性複本。
func TestStackTraceCopy(t *testing.T) {
	err := errs.New("TEST", "test")
	st1 := err.StackTrace()
	require.NotEmpty(t, st1)

	// 修改回傳的 slice
	st1[0].Function = "MUTATED"

	// 重新取得——原始資料不得受影響
	st2 := err.StackTrace()
	assert.NotEqual(t, "MUTATED", st2[0].Function)
}

// E7: Unwrap 使 errors.As 能走訪 chain。
func TestAs(t *testing.T) {
	inner := errs.New("INNER", "inner")
	middle := errs.Wrap(inner, "MIDDLE", "middle")
	outer := errs.Wrap(middle, "OUTER", "outer")

	// errors.As 先找到最外層的 *Error
	var target *errs.Error
	require.True(t, errors.As(outer, &target))
	assert.Equal(t, "OUTER", target.Code())

	// 手動走訪驗證 chain 完整性
	cause := outer.Unwrap()
	require.NotNil(t, cause)
	var middleTarget *errs.Error
	require.True(t, errors.As(cause, &middleTarget))
	assert.Equal(t, "MIDDLE", middleTarget.Code())
}

// E8: %s / %v / %q 產生確定性輸出。
func TestFormatBasicVerbs(t *testing.T) {
	err := errs.New("TEST_CODE", "test message")

	tests := []struct {
		name   string
		format string
		want   string
	}{
		{"percent_s", "%s", "[TEST_CODE] test message"},
		{"percent_v", "%v", "[TEST_CODE] test message"},
		{"percent_q", "%q", `"[TEST_CODE] test message"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fmt.Sprintf(tt.format, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// E9: %+v 印出完整 stack trace，同時處理 *Error 與一般 error cause。
func TestFormatVerbose(t *testing.T) {
	t.Run("errs_error_cause", func(t *testing.T) {
		inner := errs.New("INNER", "inner error")
		outer := errs.Wrap(inner, "OUTER", "outer error")

		output := fmt.Sprintf("%+v", outer)

		assert.Contains(t, output, "[OUTER] outer error")
		assert.Contains(t, output, "Caused by: [INNER] inner error")

		// 內層 *Error cause 必須包含 stack frame
		parts := strings.SplitN(output, "Caused by:", 2)
		require.Len(t, parts, 2)
		assert.Contains(t, parts[1], "at ")
	})

	t.Run("plain_error_cause", func(t *testing.T) {
		plainErr := errors.New("plain failure")
		wrapped := errs.Wrap(plainErr, "WRAP", "wrapped")

		output := fmt.Sprintf("%+v", wrapped)

		assert.Contains(t, output, "[WRAP] wrapped")
		assert.Contains(t, output, "Caused by: plain failure")

		// 一般 error cause 不得有偽造的 stack
		parts := strings.SplitN(output, "Caused by:", 2)
		require.Len(t, parts, 2)
		assert.NotContains(t, parts[1], "at ")
	})

	t.Run("three_level_chain", func(t *testing.T) {
		root := errors.New("io timeout")
		mid := errs.Wrap(root, "CONN", "connection failed")
		top := errs.Wrap(mid, "SVC", "service unavailable")

		output := fmt.Sprintf("%+v", top)

		assert.Contains(t, output, "[SVC] service unavailable")
		assert.Contains(t, output, "Caused by: [CONN] connection failed")
		assert.Contains(t, output, "Caused by: io timeout")
	})
}

// E10: 第一個 stack frame 為呼叫者，檔名僅為 basename。
func TestStackSkip(t *testing.T) {
	err := errs.New("TEST", "test") // 這行即為預期的第一個 frame
	st := err.StackTrace()
	require.NotEmpty(t, st)

	first := st[0]
	assert.Contains(t, first.Function, "TestStackSkip",
		"第一個 frame 應為呼叫端的測試函式")
	assert.NotContains(t, first.Function, "errs.New")
	assert.NotContains(t, first.Function, "errs.capture")

	// 檔名必須為 basename——不含路徑分隔符
	assert.Equal(t, "errs_test.go", first.File)
	assert.NotContains(t, first.File, string('/'))
	assert.NotContains(t, first.File, string('\\'))
}

// E11: nil receiver 不 panic。
func TestNilReceiver(t *testing.T) {
	var err *errs.Error

	assert.Equal(t, "", err.Code())
	assert.Equal(t, "", err.Message())
	assert.Nil(t, err.StackTrace())
	assert.Equal(t, "<nil>", err.Error())
	assert.Nil(t, err.Unwrap())

	// 所有 format verb 都不得 panic
	assert.Equal(t, "<nil>", fmt.Sprintf("%s", err))
	assert.Equal(t, "<nil>", fmt.Sprintf("%v", err))
	assert.Equal(t, "<nil>", fmt.Sprintf("%+v", err))
	assert.Equal(t, "<nil>", fmt.Sprintf("%q", err))
}

// E11: nil cause 不 panic，且 %+v 不輸出 "Caused by:"。
func TestNilCause(t *testing.T) {
	err := errs.New("CODE", "no cause")

	assert.Nil(t, err.Unwrap())

	output := fmt.Sprintf("%+v", err)
	assert.NotContains(t, output, "Caused by:")
}

// E12: Wrap(nil, ...) 與 Wrapf(nil, ...) 回傳 nil。
func TestWrapNil(t *testing.T) {
	assert.Nil(t, errs.Wrap(nil, "CODE", "msg"))
	assert.Nil(t, errs.Wrapf(nil, "CODE", "msg %s", "arg"))
}

// E15: FormatStack() 回傳格式化的 stack trace 字串，回傳 string 型別以利 duck-typing。
func TestFormatStack(t *testing.T) {
	t.Run("returns_formatted_stack", func(t *testing.T) {
		err := errs.New("TEST", "test")
		s := err.FormatStack()
		require.NotEmpty(t, s)

		lines := strings.Split(s, "\n")
		// 第一行應包含呼叫者函式名稱
		assert.Contains(t, lines[0], "TestFormatStack")
		// 每行格式 "Function (File:Line)"
		assert.Contains(t, lines[0], "(errs_test.go:")
		// 不應有 "    at " prefix（Java-style 是 %+v 的格式，FormatStack 用簡潔格式）
		assert.NotContains(t, lines[0], "    at ")
	})

	t.Run("nil_receiver", func(t *testing.T) {
		var err *errs.Error
		assert.Equal(t, "", err.FormatStack())
	})

	t.Run("consistent_with_stack_trace", func(t *testing.T) {
		err := errs.New("TEST", "test")
		st := err.StackTrace()
		s := err.FormatStack()
		lines := strings.Split(s, "\n")
		// FormatStack 行數應與 StackTrace frame 數一致
		assert.Equal(t, len(st), len(lines))
	})
}
