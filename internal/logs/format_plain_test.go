package logs

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTime 固定測試時間，避免時間依賴造成 flaky test。
var testTime = time.Date(2026, 3, 28, 14, 5, 23, 456000000, time.Local)

func TestPlainFormatter_BasicMessage(t *testing.T) {
	f := NewPlainFormatter()
	entry := Entry{
		Time:    testTime,
		Level:   LevelInfo,
		Message: "hello world",
	}

	data, err := f.Format(entry)
	require.NoError(t, err)

	out := string(data)
	assert.True(t, strings.HasSuffix(out, "\n"), "output must end with newline")
	assert.Contains(t, out, "260328 14:05:23.456")
	assert.Contains(t, out, "[INFO ]")
	assert.Contains(t, out, "hello world")
	// 無 kv，應只有一行
	assert.Equal(t, 1, strings.Count(out, "\n"), "basic message should be single line")
}

func TestPlainFormatter_KVPairs_Numbering(t *testing.T) {
	f := NewPlainFormatter()
	entry := Entry{
		Time:    testTime,
		Level:   LevelInfo,
		Message: "user logged in",
		Args:    []any{"request_id", "abc-123", "user_id", 42},
	}

	data, err := f.Format(entry)
	require.NoError(t, err)

	out := string(data)
	assert.Contains(t, out, "(1)")
	assert.Contains(t, out, "(2)")
	assert.Contains(t, out, "request_id")
	assert.Contains(t, out, "abc-123")
	assert.Contains(t, out, "user_id")
	assert.Contains(t, out, "42")
}

func TestPlainFormatter_BoundAndArgs_SharedCounter(t *testing.T) {
	f := NewPlainFormatter()
	entry := Entry{
		Time:    testTime,
		Level:   LevelInfo,
		Message: "request handled",
		Bound:   []any{"service", "auth"},
		Args:    []any{"user_id", 99},
	}

	data, err := f.Format(entry)
	require.NoError(t, err)

	out := string(data)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// header + 2 kv lines
	require.Len(t, lines, 3)

	// Bound 先：(1) service : auth
	assert.Contains(t, lines[1], "(1)")
	assert.Contains(t, lines[1], "service")
	assert.Contains(t, lines[1], "auth")

	// Args 後：(2) user_id : 99
	assert.Contains(t, lines[2], "(2)")
	assert.Contains(t, lines[2], "user_id")
	assert.Contains(t, lines[2], "99")
}

func TestPlainFormatter_KeyAlignment(t *testing.T) {
	f := NewPlainFormatter()
	entry := Entry{
		Time:    testTime,
		Level:   LevelInfo,
		Message: "alignment test",
		Args:    []any{"request_id", "abc-123", "user_id", 42},
	}

	data, err := f.Format(entry)
	require.NoError(t, err)

	lines := strings.Split(string(data), "\n")
	// lines[1] = "  (1) request_id : abc-123"
	// lines[2] = "  (2) user_id    : 42"
	require.True(t, len(lines) >= 3)

	// 找 `:` 的欄位位置，兩行應相同
	colonPos1 := strings.Index(lines[1], " : ")
	colonPos2 := strings.Index(lines[2], " : ")
	assert.Equal(t, colonPos1, colonPos2, "colon columns must be aligned")
}

func TestPlainFormatter_WithError_WidthAligned(t *testing.T) {
	f := NewPlainFormatter()
	entry := Entry{
		Time:    testTime,
		Level:   LevelError,
		Message: "query failed",
		Args:    []any{"table", "users", "id", 42},
		Error: &fullError{
			code:    "DB_TIMEOUT",
			message: "connection pool exhausted",
			stack:   "service.LoadUser (service.go:42)\nhandler.GetUser (handler.go:18)",
		},
	}

	data, err := f.Format(entry)
	require.NoError(t, err)

	out := string(data)
	// index width=5，對齊 "(error)"
	assert.Contains(t, out, "(    1)")
	assert.Contains(t, out, "(    2)")
	assert.Contains(t, out, "(error)")
	assert.Contains(t, out, "[DB_TIMEOUT]")
	assert.Contains(t, out, "connection pool exhausted")
	assert.Contains(t, out, "    at service.LoadUser (service.go:42)")
	assert.Contains(t, out, "    at handler.GetUser (handler.go:18)")
}

func TestPlainFormatter_NoError_NarrowWidth(t *testing.T) {
	f := NewPlainFormatter()
	entry := Entry{
		Time:    testTime,
		Level:   LevelInfo,
		Message: "narrow width",
		Args:    []any{"key", "val"},
	}

	data, err := f.Format(entry)
	require.NoError(t, err)

	out := string(data)
	// 無 error，1 個 kv，width=1 → "(1)" 而非 "(    1)"
	assert.Contains(t, out, "(1)")
	assert.NotContains(t, out, "(    1)")
}

func TestPlainFormatter_ErrorNoCode(t *testing.T) {
	f := NewPlainFormatter()
	entry := Entry{
		Time:    testTime,
		Level:   LevelError,
		Message: "plain error",
		Error:   errors.New("something went wrong"),
	}

	data, err := f.Format(entry)
	require.NoError(t, err)

	out := string(data)
	assert.Contains(t, out, "(error)")
	assert.Contains(t, out, "something went wrong")
	// 無 Code，error block 不應出現方括號（header 的 [ERROR] 不在此考量）
	lines := strings.Split(out, "\n")
	var errorLine string
	for _, l := range lines {
		if strings.Contains(l, "(error)") {
			errorLine = l
			break
		}
	}
	require.NotEmpty(t, errorLine, "error line must exist")
	assert.NotContains(t, errorLine, "[", "error block should not have code brackets when no code")
}

func TestPlainFormatter_CustomTimeFormat(t *testing.T) {
	f := NewPlainFormatter(WithTimeFormat("2006-01-02"))
	entry := Entry{
		Time:    testTime,
		Level:   LevelDebug,
		Message: "custom time",
	}

	data, err := f.Format(entry)
	require.NoError(t, err)

	out := string(data)
	assert.Contains(t, out, "2026-03-28")
	// 確認舊格式不在輸出
	assert.NotContains(t, out, "260328")
}

func TestPlainFormatter_NilArgsNilBound(t *testing.T) {
	f := NewPlainFormatter()
	entry := Entry{
		Time:    testTime,
		Level:   LevelWarn,
		Message: "no kv pairs",
		Bound:   nil,
		Args:    nil,
	}

	data, err := f.Format(entry)
	require.NoError(t, err)

	out := string(data)
	// 只有 header，無 kv block
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	assert.Len(t, lines, 1)
	assert.Contains(t, out, "[WARN ]")
	assert.Contains(t, out, "no kv pairs")
}
