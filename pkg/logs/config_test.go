package logs

import (
	"bytes"
	"sync"
	"testing"

	ilogs "golan-example/internal/logs"

	"github.com/stretchr/testify/assert"
)

func resetLogsForTest() {
	configureOnce = sync.Once{}
	ilogs.ResetForTest()
}

func TestConfigure_GlobalPipe(t *testing.T) {
	resetLogsForTest()
	var buf bytes.Buffer
	Configure(Pipe(Plain(), ToWriter(&buf)))
	ilogs.Info("hello", nil)
	assert.Contains(t, buf.String(), "hello")
}

func TestConfigure_PerLevel_ForError(t *testing.T) {
	resetLogsForTest()
	var infoBuf, errorBuf bytes.Buffer
	Configure(
		Pipe(Plain(), ToWriter(&infoBuf)),
		ForError(Pipe(Plain(), ToWriter(&errorBuf))),
	)
	ilogs.Info("info msg", nil)
	ilogs.Error("error msg", nil)
	assert.Contains(t, infoBuf.String(), "info msg")
	assert.Contains(t, infoBuf.String(), "error msg")
	assert.Contains(t, errorBuf.String(), "error msg")
	assert.NotContains(t, errorBuf.String(), "info msg")
}

func TestConfigure_NoInherit(t *testing.T) {
	resetLogsForTest()
	var globalBuf, debugBuf bytes.Buffer
	Configure(
		Pipe(Plain(), ToWriter(&globalBuf)),
		ForDebug(NoInherit(), Pipe(Plain(), ToWriter(&debugBuf))),
	)
	ilogs.Debug("debug msg", nil)
	ilogs.Info("info msg", nil)
	assert.Contains(t, debugBuf.String(), "debug msg")
	assert.NotContains(t, globalBuf.String(), "debug msg")
	assert.Contains(t, globalBuf.String(), "info msg")
}

func TestConfigure_NoCaller(t *testing.T) {
	resetLogsForTest()
	var buf bytes.Buffer
	Configure(NoCaller(), Pipe(Plain(), ToWriter(&buf)))
	ilogs.Info("test", nil)
	assert.NotContains(t, buf.String(), "caller")
}

func TestConfigure_CallerDefault(t *testing.T) {
	resetLogsForTest()
	var buf bytes.Buffer
	Configure(Pipe(Plain(), ToWriter(&buf)))
	ilogs.Info("test", nil)
	assert.Contains(t, buf.String(), "caller")
}

func TestConfigure_WithFilter(t *testing.T) {
	resetLogsForTest()
	var buf bytes.Buffer
	Configure(
		WithFilter(FilterByMessage(func(m string) bool { return m != "skip" })),
		Pipe(Plain(), ToWriter(&buf)),
	)
	ilogs.Info("keep", nil)
	ilogs.Info("skip", nil)
	assert.Contains(t, buf.String(), "keep")
	assert.NotContains(t, buf.String(), "skip")
}

func TestConfigure_OnceOnly(t *testing.T) {
	resetLogsForTest()
	var buf1, buf2 bytes.Buffer
	Configure(Pipe(Plain(), ToWriter(&buf1)))
	Configure(Pipe(Plain(), ToWriter(&buf2)))
	ilogs.Info("test", nil)
	assert.Contains(t, buf1.String(), "test")
	assert.Empty(t, buf2.String())
}
