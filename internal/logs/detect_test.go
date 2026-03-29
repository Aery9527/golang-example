package logs

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fullError struct {
	code    string
	message string
	stack   string
}

func (e *fullError) Error() string       { return "[" + e.code + "] " + e.message }
func (e *fullError) Code() string        { return e.code }
func (e *fullError) Message() string     { return e.message }
func (e *fullError) FormatStack() string { return e.stack }

type formatterError struct {
	msg   string
	trace string
}

func (e *formatterError) Error() string { return e.msg }
func (e *formatterError) Format(f fmt.State, verb rune) {
	if verb == 'v' && f.Flag('+') {
		fmt.Fprintf(f, "%s\n%s", e.msg, e.trace)
		return
	}
	fmt.Fprint(f, e.msg)
}

func TestExtractError_FullDuckTyping(t *testing.T) {
	err := &fullError{code: "DB_FAIL", message: "conn lost", stack: "svc.Load (svc.go:42)"}
	info := ExtractError(err)

	assert.Equal(t, "DB_FAIL", info.Code)
	assert.Equal(t, "conn lost", info.Message)
	assert.Equal(t, "svc.Load (svc.go:42)", info.Stack)
}

func TestExtractError_PlainError(t *testing.T) {
	err := errors.New("something broke")
	info := ExtractError(err)

	assert.Equal(t, "", info.Code)
	assert.Equal(t, "something broke", info.Message)
	assert.Equal(t, "", info.Stack)
}

func TestExtractError_FmtFormatterFallback(t *testing.T) {
	err := &formatterError{msg: "fail", trace: "at main.go:1"}
	info := ExtractError(err)

	assert.Equal(t, "", info.Code)
	assert.Equal(t, "fail", info.Message)
	assert.Contains(t, info.Stack, "at main.go:1")
}

func TestExtractError_Nil(t *testing.T) {
	info := ExtractError(nil)
	assert.Equal(t, ErrorInfo{}, info)
}
