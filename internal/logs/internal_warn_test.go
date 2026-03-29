package logs

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInternalWarn_WithWarnChain(t *testing.T) {
	var captured Entry
	terminal := func(e Entry) { captured = e }
	chain := NewChain(nil, terminal)
	SetWarnChain(chain)
	defer SetWarnChain(nil)

	internalWarn("test warning", "key", "val")

	assert.Equal(t, LevelWarn, captured.Level)
	assert.Equal(t, "test warning", captured.Message)
	assert.True(t, captured.internal)
	assert.Equal(t, []any{"key", "val"}, captured.Args)
}

func TestInternalWarn_NoWarnChain_Stderr(t *testing.T) {
	SetWarnChain(nil)

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	internalWarn("no chain", "detail", "x")

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	assert.Contains(t, buf.String(), "no chain")
}

func TestInternalWarn_RecursionProtection(t *testing.T) {
	failSink := NewSink(
		&stubFormatter{output: nil, err: assert.AnError},
		&bytes.Buffer{},
	)
	chain := NewChain(nil, func(e Entry) { failSink.Write(e) })
	SetWarnChain(chain)
	defer SetWarnChain(nil)

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	internalWarn("will fail in sink", "k", "v")

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	s := buf.String()
	assert.True(t, strings.Contains(s, "format failed") || strings.Contains(s, "LOGS_INTERNAL"))
}
