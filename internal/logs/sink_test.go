package logs

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubFormatter struct {
	output []byte
	err    error
}

func (f *stubFormatter) Format(entry Entry) ([]byte, error) {
	return f.output, f.err
}

func TestSink_Write_Success(t *testing.T) {
	var buf bytes.Buffer
	sink := NewSink(&stubFormatter{output: []byte("hello\n")}, &buf)

	sink.Write(Entry{Message: "test"})
	assert.Equal(t, "hello\n", buf.String())
}

func TestSink_Write_FormatError_InternalEntry(t *testing.T) {
	var buf bytes.Buffer
	sink := NewSink(&stubFormatter{err: errors.New("bad format")}, &buf)

	sink.Write(Entry{Message: "test", internal: true})
	assert.Empty(t, buf.String())
}

func TestSink_Write_WriteError_InternalEntry(t *testing.T) {
	sink := NewSink(
		&stubFormatter{output: []byte("data")},
		&failWriter{err: errors.New("disk full")},
	)
	// Should not panic
	sink.Write(Entry{Message: "test", internal: true})
}

type failWriter struct {
	err error
}

func (w *failWriter) Write(p []byte) (int, error) {
	return 0, w.err
}
