package logs

import (
	"bytes"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupTestDefault(t *testing.T, buf *bytes.Buffer) {
	t.Helper()
	resetForTest()
	sink := NewSink(NewPlainFormatter(), buf)
	var chains [4]*Chain
	for i := 0; i < 4; i++ {
		chains[i] = NewChain(nil, FanOut([]SinkWriter{sink}))
	}
	Init(chains)
}

func resetForTest() {
	initOnce = sync.Once{}
	defaultLogger = nil
	SetWarnChain(nil)
}

func TestPackageLevel_Info(t *testing.T) {
	var buf bytes.Buffer
	setupTestDefault(t, &buf)
	Info("package level", nil)
	assert.Contains(t, buf.String(), "package level")
}

func TestPackageLevel_ErrorWith(t *testing.T) {
	var buf bytes.Buffer
	setupTestDefault(t, &buf)
	ErrorWith("test error", func() (error, []any) {
		return &fullError{code: "X", message: "y", stack: ""}, nil
	})
	assert.Contains(t, buf.String(), "[X]")
}

func TestPackageLevel_With(t *testing.T) {
	var buf bytes.Buffer
	setupTestDefault(t, &buf)
	child := With("rid", "abc")
	child.Info("child log", nil)
	assert.Contains(t, buf.String(), "rid")
}

func TestInit_SetsDefaultLogger(t *testing.T) {
	var buf bytes.Buffer
	sink := NewSink(NewPlainFormatter(), &buf)
	var chains [4]*Chain
	for i := 0; i < 4; i++ {
		chains[i] = NewChain(nil, FanOut([]SinkWriter{sink}))
	}
	resetForTest()
	Init(chains)
	Info("after init", nil)
	assert.Contains(t, buf.String(), "after init")
}

func TestEnsureInit_DefaultConfig(t *testing.T) {
	resetForTest()
	// Should not panic — uses default Plain + Console
	Info("default config", nil)
}

func TestInit_OnceOnly(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	resetForTest()

	sink1 := NewSink(NewPlainFormatter(), &buf1)
	var chains1 [4]*Chain
	for i := 0; i < 4; i++ {
		chains1[i] = NewChain(nil, FanOut([]SinkWriter{sink1}))
	}
	Init(chains1)

	sink2 := NewSink(NewPlainFormatter(), &buf2)
	var chains2 [4]*Chain
	for i := 0; i < 4; i++ {
		chains2[i] = NewChain(nil, FanOut([]SinkWriter{sink2}))
	}
	Init(chains2) // should be no-op

	Info("test", nil)
	assert.Contains(t, buf1.String(), "test")
	assert.Empty(t, buf2.String())
}
