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
