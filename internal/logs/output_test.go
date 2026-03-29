package logs

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConsoleOutput_DebugInfo_Stdout(t *testing.T) {
	out := NewConsoleOutput()
	assert.Equal(t, os.Stdout, out.Resolve(ResolveContext{Level: LevelDebug}))
	assert.Equal(t, os.Stdout, out.Resolve(ResolveContext{Level: LevelInfo}))
}

func TestConsoleOutput_WarnError_Stderr(t *testing.T) {
	out := NewConsoleOutput()
	assert.Equal(t, os.Stderr, out.Resolve(ResolveContext{Level: LevelWarn}))
	assert.Equal(t, os.Stderr, out.Resolve(ResolveContext{Level: LevelError}))
}

func TestStdoutOutput(t *testing.T) {
	out := NewStdoutOutput()
	assert.Equal(t, os.Stdout, out.Resolve(ResolveContext{Level: LevelError}))
}

func TestStderrOutput(t *testing.T) {
	out := NewStderrOutput()
	assert.Equal(t, os.Stderr, out.Resolve(ResolveContext{Level: LevelDebug}))
}

func TestWriterOutput(t *testing.T) {
	var buf bytes.Buffer
	out := NewWriterOutput(&buf)
	assert.Equal(t, &buf, out.Resolve(ResolveContext{Level: LevelInfo}))
}
