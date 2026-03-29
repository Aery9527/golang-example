package logs

import "fmt"

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

	if mp, ok := err.(messageProvider); ok {
		info.Message = mp.Message()
	} else {
		info.Message = err.Error()
	}

	if cp, ok := err.(codeProvider); ok {
		info.Code = cp.Code()
	}

	if sp, ok := err.(stackProvider); ok {
		info.Stack = sp.FormatStack()
	} else if _, ok := err.(fmt.Formatter); ok {
		info.Stack = fmt.Sprintf("%+v", err)
	}

	return info
}
