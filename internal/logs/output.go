package logs

import (
	"io"
	"os"
)

type ResolveContext struct {
	Level Level
}

type Output interface {
	Resolve(ctx ResolveContext) io.Writer
}

type consoleOutput struct{}

func NewConsoleOutput() Output { return &consoleOutput{} }
func (*consoleOutput) Resolve(ctx ResolveContext) io.Writer {
	if ctx.Level <= LevelInfo {
		return os.Stdout
	}
	return os.Stderr
}

type stdoutOutput struct{}

func NewStdoutOutput() Output                              { return &stdoutOutput{} }
func (*stdoutOutput) Resolve(_ ResolveContext) io.Writer   { return os.Stdout }

type stderrOutput struct{}

func NewStderrOutput() Output                              { return &stderrOutput{} }
func (*stderrOutput) Resolve(_ ResolveContext) io.Writer   { return os.Stderr }

type writerOutput struct{ w io.Writer }

func NewWriterOutput(w io.Writer) Output                   { return &writerOutput{w: w} }
func (o *writerOutput) Resolve(_ ResolveContext) io.Writer { return o.w }
