package logs

import (
	"io"

	ilogs "golan-example/internal/logs"
)

// Output 根據 context 解析出 io.Writer。
type Output = ilogs.Output

// ResolveContext 傳入 Output.Resolve 的參數。
type ResolveContext = ilogs.ResolveContext

func ToConsole() Output               { return ilogs.NewConsoleOutput() }
func ToStdout() Output                { return ilogs.NewStdoutOutput() }
func ToStderr() Output                { return ilogs.NewStderrOutput() }
func ToWriter(w io.Writer) Output     { return ilogs.NewWriterOutput(w) }

// ToFile 回傳 RotatingFileWriter output。
func ToFile(name string, cfg RotateConfig) Output {
	return &fileOutput{name: name, cfg: cfg}
}

// fileOutput 延遲建立 RotatingFileWriter 直到 Resolve 被呼叫。
type fileOutput struct {
	name string
	cfg  RotateConfig
	ext  string // set by config layer before Resolve
}

func (o *fileOutput) Resolve(_ ilogs.ResolveContext) io.Writer {
	ext := o.ext
	if ext == "" {
		ext = ".log"
	}
	w, err := NewRotatingFileWriter(o.name, ext, o.cfg)
	if err != nil {
		// Fallback to stderr on failure
		return ilogs.NewStderrOutput().Resolve(ilogs.ResolveContext{})
	}
	return w
}
