package errs

import "fmt"

// Wrap 以新的 code、message 與 stack trace 包裝既有 error。若 err 為 nil，
// Wrap 回傳 nil——遵循 Go 慣例保留 nil error，呼叫端可安全地寫：
//
//	return errs.Wrap(err, "DB_FAIL", "query failed")
//
// 若要建立全新的根錯誤（不包裝），請改用 New 或 Newf。
func Wrap(err error, code, message string) *Error {
	if err == nil {
		return nil
	}
	return &Error{
		code:    code,
		message: message,
		cause:   err,
		stack:   capture(3),
	}
}

// Wrapf 以新的 code 與格式化 message 包裝既有 error。若 err 為 nil，Wrapf
// 回傳 nil。行為細節請參考 Wrap。
func Wrapf(err error, code, format string, args ...any) *Error {
	if err == nil {
		return nil
	}
	return &Error{
		code:    code,
		message: fmt.Sprintf(format, args...),
		cause:   err,
		stack:   capture(3),
	}
}
