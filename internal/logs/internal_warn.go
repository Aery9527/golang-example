package logs

import (
	"fmt"
	"os"
	"time"
)

// warnChain 是 write-once-then-read 的全域變數。
// Init/ensureInit 透過 sync.Once 寫入一次，之後只讀取。
// ResetForTest 僅供測試使用（單 goroutine 環境）。
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

// stderrFallback 是最終 fallback——直接寫 stderr。
func stderrFallback(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "[LOGS_INTERNAL] %s: %v\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "[LOGS_INTERNAL] %s\n", msg)
	}
}
