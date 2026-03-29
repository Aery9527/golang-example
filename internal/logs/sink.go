package logs

import "io"

// Formatter 將 Entry 格式化為 byte slice。
type Formatter interface {
	Format(entry Entry) ([]byte, error)
}

// formatterConfig 是 PlainFormatter / JSONFormatter 共用的設定。
type formatterConfig struct {
	timeLayout string
}

// FormatterOption 用於自訂 Formatter 行為。
type FormatterOption func(*formatterConfig)

// WithTimeFormat 設定 time 輸出格式（Go time layout string）。
func WithTimeFormat(layout string) FormatterOption {
	return func(c *formatterConfig) {
		c.timeLayout = layout
	}
}

func applyFormatterOpts(defaults formatterConfig, opts []FormatterOption) formatterConfig {
	cfg := defaults
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

// Sink 是 chain 的 terminal node——format + write。
type Sink struct {
	formatter Formatter
	writer    io.Writer
}

// NewSink 建立 Sink。
func NewSink(f Formatter, w io.Writer) *Sink {
	return &Sink{formatter: f, writer: w}
}

// Write 格式化 entry 並寫入 writer。失敗時走 internal warn 或 stderr fallback。
func (s *Sink) Write(entry Entry) {
	data, err := s.formatter.Format(entry)
	if err != nil {
		s.handleError("format failed", err, entry)
		return
	}
	if _, err := s.writer.Write(data); err != nil {
		s.handleError("write failed", err, entry)
	}
}

func (s *Sink) handleError(msg string, err error, original Entry) {
	if original.internal {
		stderrFallback(msg, err)
		return
	}
	internalWarn(msg, "error", err)
}
