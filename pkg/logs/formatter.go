package logs

import ilogs "golan-example/internal/logs"

type Formatter = ilogs.Formatter
type FormatterOption = ilogs.FormatterOption

func WithTimeFormat(layout string) FormatterOption {
	return ilogs.WithTimeFormat(layout)
}

func Plain(opts ...FormatterOption) Formatter {
	return ilogs.NewPlainFormatter(opts...)
}

func JSON(opts ...FormatterOption) Formatter {
	return ilogs.NewJSONFormatter(opts...)
}

func formatterExt(f Formatter) string {
	switch f.(type) {
	case *ilogs.JSONFormatter:
		return ".json"
	default:
		return ".log"
	}
}
