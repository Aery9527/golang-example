package logs

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jsonTestTime 固定 UTC 時間，供 JSON 格式化測試使用。
var jsonTestTime = time.Date(2026, 3, 28, 14, 5, 23, 456000000, time.UTC)

func makeJSONEntry() Entry {
	return Entry{
		Time:    jsonTestTime,
		Level:   LevelInfo,
		Message: "hello",
	}
}

func parseJSON(t *testing.T, data []byte) map[string]any {
	t.Helper()
	// 去掉結尾 \n 後 unmarshal
	trimmed := strings.TrimRight(string(data), "\n")
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(trimmed), &m))
	return m
}

// TestJSONFormatter_BasicStructure — time / level / msg 固定欄位，time 格式 ISO 8601
func TestJSONFormatter_BasicStructure(t *testing.T) {
	f := NewJSONFormatter()
	entry := makeJSONEntry()

	data, err := f.Format(entry)
	require.NoError(t, err)

	m := parseJSON(t, data)
	assert.Equal(t, "INFO", m["level"])
	assert.Equal(t, "hello", m["msg"])

	timeStr, ok := m["time"].(string)
	require.True(t, ok)
	// ISO 8601 基本格式驗證：包含 T 分隔符與 Z 時區
	assert.Contains(t, timeStr, "T")
	assert.Equal(t, "2026-03-28T14:05:23.456Z", timeStr)
}

// TestJSONFormatter_KVPairs — Bound + Args 平鋪在 JSON object 中
func TestJSONFormatter_KVPairs(t *testing.T) {
	f := NewJSONFormatter()
	entry := makeJSONEntry()
	entry.Bound = []any{"service", "auth"}
	entry.Args = []any{"user_id", 42}

	data, err := f.Format(entry)
	require.NoError(t, err)

	m := parseJSON(t, data)
	assert.Equal(t, "auth", m["service"])
	assert.Equal(t, float64(42), m["user_id"])
}

// TestJSONFormatter_KeyConflict — 重複 key 加上 _2 後綴
func TestJSONFormatter_KeyConflict(t *testing.T) {
	f := NewJSONFormatter()
	entry := makeJSONEntry()
	entry.Args = []any{"key", "first", "key", "second"}

	data, err := f.Format(entry)
	require.NoError(t, err)

	m := parseJSON(t, data)
	assert.Equal(t, "first", m["key"])
	assert.Equal(t, "second", m["key_2"])
}

// TestJSONFormatter_ErrorFields — fullError 帶 code/msg/stack，三個欄位都存在且有值
func TestJSONFormatter_ErrorFields(t *testing.T) {
	f := NewJSONFormatter()
	entry := makeJSONEntry()
	entry.Error = &fullError{
		code:    "DB_ERR",
		message: "connection lost",
		stack:   "svc.Load (svc.go:42)",
	}

	data, err := f.Format(entry)
	require.NoError(t, err)

	m := parseJSON(t, data)
	assert.Equal(t, "DB_ERR", m["err_code"])
	assert.Equal(t, "connection lost", m["err_msg"])
	assert.Equal(t, "svc.Load (svc.go:42)", m["err_stack"])
}

// TestJSONFormatter_ErrorFieldsPlainError — errors.New → err_code: null, err_stack: null
func TestJSONFormatter_ErrorFieldsPlainError(t *testing.T) {
	f := NewJSONFormatter()
	entry := makeJSONEntry()
	entry.Error = errors.New("plain error")

	data, err := f.Format(entry)
	require.NoError(t, err)

	// 用原始 JSON 檢查 null 值（map unmarshal null → nil）
	trimmed := strings.TrimRight(string(data), "\n")
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(trimmed), &raw))

	assert.Equal(t, json.RawMessage("null"), raw["err_code"])
	assert.Equal(t, json.RawMessage(`"plain error"`), raw["err_msg"])
	assert.Equal(t, json.RawMessage("null"), raw["err_stack"])
}

// TestJSONFormatter_ErrorFieldsAlwaysPresent — err_ 三個 key 一定都存在於輸出字串
func TestJSONFormatter_ErrorFieldsAlwaysPresent(t *testing.T) {
	f := NewJSONFormatter()
	entry := makeJSONEntry()
	entry.Error = errors.New("any error")

	data, err := f.Format(entry)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, `"err_code"`)
	assert.Contains(t, output, `"err_msg"`)
	assert.Contains(t, output, `"err_stack"`)
}

// TestJSONFormatter_NumberPreservesType — int → json.Unmarshal 得到 float64；float64 同樣保留
func TestJSONFormatter_NumberPreservesType(t *testing.T) {
	f := NewJSONFormatter()
	entry := makeJSONEntry()
	entry.Args = []any{"count", 99, "ratio", 0.75}

	data, err := f.Format(entry)
	require.NoError(t, err)

	m := parseJSON(t, data)
	assert.Equal(t, float64(99), m["count"])
	assert.Equal(t, float64(0.75), m["ratio"])
}

// TestJSONFormatter_RawMessageValid — 合法 json.RawMessage 嵌入為 nested object
func TestJSONFormatter_RawMessageValid(t *testing.T) {
	f := NewJSONFormatter()
	entry := makeJSONEntry()
	entry.Args = []any{"meta", json.RawMessage(`{"nested":true}`)}

	data, err := f.Format(entry)
	require.NoError(t, err)

	m := parseJSON(t, data)
	meta, ok := m["meta"].(map[string]any)
	require.True(t, ok, "meta should be a nested object")
	assert.Equal(t, true, meta["nested"])
}

// TestJSONFormatter_RawMessageInvalid — 非法 json.RawMessage 降格為 string
func TestJSONFormatter_RawMessageInvalid(t *testing.T) {
	f := NewJSONFormatter()
	entry := makeJSONEntry()
	entry.Args = []any{"bad", json.RawMessage(`{broken)`)}

	data, err := f.Format(entry)
	require.NoError(t, err)

	m := parseJSON(t, data)
	// 降格後為 string 型別
	_, isStr := m["bad"].(string)
	assert.True(t, isStr, "invalid RawMessage should degrade to string")
}

// TestJSONFormatter_OddArgs — 奇數個 args → _arg0 key
func TestJSONFormatter_OddArgs(t *testing.T) {
	f := NewJSONFormatter()
	entry := makeJSONEntry()
	entry.Args = []any{"orphan"}

	data, err := f.Format(entry)
	require.NoError(t, err)

	m := parseJSON(t, data)
	assert.Equal(t, "orphan", m["_arg0"])
}

// TestJSONFormatter_CustomTimeFormat — WithTimeFormat(time.Kitchen) → time 欄位為 "2:05PM"
func TestJSONFormatter_CustomTimeFormat(t *testing.T) {
	f := NewJSONFormatter(WithTimeFormat(time.Kitchen))
	entry := makeJSONEntry()

	data, err := f.Format(entry)
	require.NoError(t, err)

	m := parseJSON(t, data)
	assert.Equal(t, "2:05PM", m["time"])
}
