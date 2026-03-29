package logs

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func captureLogger() (*Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	formatter := NewPlainFormatter()
	sink := NewSink(formatter, &buf)
	terminal := FanOut([]SinkWriter{sink})

	var chains [4]*Chain
	for i := 0; i < 4; i++ {
		chains[i] = NewChain(nil, terminal)
	}
	return NewLogger(chains), &buf
}

func TestLogger_Info_NilClosure(t *testing.T) {
	l, buf := captureLogger()
	l.Info("bare message", nil)
	assert.Contains(t, buf.String(), "bare message")
	assert.Contains(t, buf.String(), "[INFO ]")
}

func TestLogger_Info_WithClosure(t *testing.T) {
	l, buf := captureLogger()
	l.Info("test", func() []any { return []any{"key", "value"} })
	s := buf.String()
	assert.Contains(t, s, "key")
	assert.Contains(t, s, "value")
}

func TestLogger_LazyClosure_NilChain_NotExecuted(t *testing.T) {
	var chains [4]*Chain
	var buf bytes.Buffer
	sink := NewSink(NewPlainFormatter(), &buf)
	chains[LevelInfo] = NewChain(nil, FanOut([]SinkWriter{sink}))
	l := NewLogger(chains)

	executed := false
	l.Debug("should not", func() []any {
		executed = true
		return nil
	})
	assert.False(t, executed, "closure should NOT execute when chain is nil")
}

func TestLogger_LazyClosure_ChainExists_Executed(t *testing.T) {
	l, _ := captureLogger()
	executed := false
	l.Info("test", func() []any {
		executed = true
		return nil
	})
	assert.True(t, executed)
}

func TestLogger_ErrorWith(t *testing.T) {
	l, buf := captureLogger()
	l.ErrorWith("query failed", func() (error, []any) {
		return &fullError{code: "DB", message: "timeout", stack: "s.go:1"}, []any{"table", "users"}
	})
	s := buf.String()
	assert.Contains(t, s, "query failed")
	assert.Contains(t, s, "(error)")
	assert.Contains(t, s, "[DB]")
	assert.Contains(t, s, "table")
}

func TestLogger_With_BindsKVPairs(t *testing.T) {
	l, buf := captureLogger()
	child := l.With("rid", "abc-123")
	child.Info("handling", nil)
	s := buf.String()
	assert.Contains(t, s, "rid")
	assert.Contains(t, s, "abc-123")
}

func TestLogger_With_DoesNotAffectParent(t *testing.T) {
	l, buf := captureLogger()
	_ = l.With("rid", "abc")
	buf.Reset()
	l.Info("parent log", nil)
	assert.NotContains(t, buf.String(), "rid")
}

func TestLogger_With_ChainShared(t *testing.T) {
	l, buf := captureLogger()
	child := l.With("child", true)

	buf.Reset()
	child.Info("child msg", nil)
	assert.Contains(t, buf.String(), "child msg")

	buf.Reset()
	l.Info("parent msg", nil)
	assert.Contains(t, buf.String(), "parent msg")
}

func TestLogger_With_ChainedWith(t *testing.T) {
	l, buf := captureLogger()
	child := l.With("a", 1).With("b", 2)
	child.Info("test", nil)
	s := buf.String()
	assert.Contains(t, s, "a")
	assert.Contains(t, s, "b")
}

func TestLogger_AllLevels(t *testing.T) {
	l, buf := captureLogger()
	tests := []struct {
		name string
		fn   func()
		want string
	}{
		{"Debug", func() { l.Debug("d", nil) }, "[DEBUG]"},
		{"Info", func() { l.Info("i", nil) }, "[INFO ]"},
		{"Warn", func() { l.Warn("w", nil) }, "[WARN ]"},
		{"Error", func() { l.Error("e", nil) }, "[ERROR]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.fn()
			assert.Contains(t, buf.String(), tt.want)
		})
	}
}

func TestLogger_FanOut_MultipleSinks(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	sink1 := NewSink(NewPlainFormatter(), &buf1)
	sink2 := NewSink(NewPlainFormatter(), &buf2)

	var chains [4]*Chain
	chains[LevelInfo] = NewChain(nil, FanOut([]SinkWriter{sink1, sink2}))
	l := NewLogger(chains)

	l.Info("fanout test", nil)
	assert.Contains(t, buf1.String(), "fanout test")
	assert.Contains(t, buf2.String(), "fanout test")
}
