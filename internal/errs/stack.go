package errs

import (
	"path/filepath"
	"runtime"
)

// Frame 代表 call stack 中的一個位置。
type Frame struct {
	Function string // 完整函式名稱，例如 "golan-example/internal/errs_test.TestNew"
	File     string // 原始碼檔案 basename，例如 "errs_test.go"
	Line     int    // 原始碼行號
}

// Stack 代表建立 error 時捕獲的 call stack。
type Stack []Frame

// capture 捕獲當前 goroutine 的 call stack。skip 參數指定從頂端跳過多少
// frame；呼叫端應設定 skip 使第一個記錄的 frame 為 New/Wrap 等函式的呼叫者。
func capture(skip int) Stack {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(skip, pcs[:])
	if n == 0 {
		return nil
	}

	frames := runtime.CallersFrames(pcs[:n])
	st := make(Stack, 0, n)
	for {
		frame, more := frames.Next()
		st = append(st, Frame{
			Function: frame.Function,
			File:     filepath.Base(frame.File),
			Line:     frame.Line,
		})
		if !more {
			break
		}
	}
	return st
}
