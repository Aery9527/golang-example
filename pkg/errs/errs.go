package errs

import (
	"errors"
	"fmt"
	"io"
)

// 編譯期 interface 驗證（golang-guidelines Rule 1）。
var _ error = (*Error)(nil)
var _ fmt.Formatter = (*Error)(nil)

// Error 代表帶有 error code 與 stack trace 的應用層錯誤。實作 error 與
// fmt.Formatter interface。使用 New / Newf 建立根錯誤，Wrap / Wrapf 包裝既有
// 錯誤並附加上下文。
//
//	err := errs.New("NOT_FOUND", "user not found")
//	wrapped := errs.Wrap(dbErr, "DB_FAIL", "query failed")
//	fmt.Printf("%+v\n", wrapped) // Java-style stack trace 含 cause chain
type Error struct {
	code    string // 應用層 error code，例如 "USER_NOT_FOUND"；依呼叫者契約必須非空
	message string // 人類可讀的錯誤描述
	cause   error  // 被包裝的 cause，用於 chain 支援；root error 時為 nil
	stack   Stack  // 建立時自動捕獲的 call stack
}

// New 以指定的 code 與 message 建立根 Error。code 應為非空字串，用於識別錯誤
// 類別（例如 "USER_NOT_FOUND"）。Stack trace 會自動從呼叫者的角度捕獲。
func New(code, message string) *Error {
	return &Error{
		code:    code,
		message: message,
		stack:   capture(3),
	}
}

// Newf 以指定的 code 與格式化 message 建立根 Error。code 與 stack trace
// 行為請參考 New。
func Newf(code, format string, args ...any) *Error {
	return &Error{
		code:    code,
		message: fmt.Sprintf(format, args...),
		stack:   capture(3),
	}
}

// Code 回傳 error code。
func (e *Error) Code() string {
	if e == nil {
		return ""
	}
	return e.code
}

// Message 回傳不帶 [CODE] prefix 的原始 message。下游消費者（例如結構化日誌）
// 應使用此方法而非解析 Error() 的輸出。
func (e *Error) Message() string {
	if e == nil {
		return ""
	}
	return e.message
}

// StackTrace 回傳捕獲的 stack trace 的防禦性複本。呼叫端可安全修改回傳的
// slice 而不影響原始 Error。
func (e *Error) StackTrace() Stack {
	if e == nil {
		return nil
	}
	cp := make(Stack, len(e.stack))
	copy(cp, e.stack)
	return cp
}

// Error 回傳格式為 "[CODE] message" 的字串。
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	return "[" + e.code + "] " + e.message
}

// Unwrap 回傳底層 cause，使 errors.Is 與 errors.As 能走訪 error chain。
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// Format 實作 fmt.Formatter。
//
//	%s, %v  — 等同 Error()
//	%q      — Error() 結果的 quoted string
//	%+v     — 完整 stack trace 含 cause chain（Java printStackTrace 風格）
func (e *Error) Format(f fmt.State, verb rune) {
	if e == nil {
		_, _ = io.WriteString(f, "<nil>")
		return
	}
	switch verb {
	case 'v':
		if f.Flag('+') {
			writeVerbose(f, e)
			return
		}
		_, _ = io.WriteString(f, e.Error())
	case 's':
		_, _ = io.WriteString(f, e.Error())
	case 'q':
		_, _ = fmt.Fprintf(f, "%q", e.Error())
	}
}

// writeVerbose 輸出 e 的完整 Java-style stack trace，包含 cause chain。
func writeVerbose(w io.Writer, e *Error) {
	_, _ = fmt.Fprintf(w, "[%s] %s", e.code, e.message)
	writeStack(w, e.stack)
	if e.cause != nil {
		writeCause(w, e.cause)
	}
}

// writeCause 走訪 error chain 並輸出每個 cause。*Error 節點包含其 stack
// trace；一般 error 僅輸出 message 字串。
func writeCause(w io.Writer, cause error) {
	for cause != nil {
		if e, ok := cause.(*Error); ok {
			_, _ = fmt.Fprintf(w, "\nCaused by: [%s] %s", e.code, e.message)
			writeStack(w, e.stack)
		} else {
			_, _ = fmt.Fprintf(w, "\nCaused by: %s", cause.Error())
		}
		cause = errors.Unwrap(cause)
	}
}

// writeStack 以 "    at Function (File:Line)" 格式輸出 stack frame。
func writeStack(w io.Writer, st Stack) {
	for _, frame := range st {
		_, _ = fmt.Fprintf(w, "\n    at %s (%s:%d)", frame.Function, frame.File, frame.Line)
	}
}
