package logs

import (
	"fmt"
	"strconv"
	"strings"
)

const defaultPlainTimeLayout = "060102 15:04:05.000"

type PlainFormatter struct {
	cfg formatterConfig
}

func NewPlainFormatter(opts ...FormatterOption) *PlainFormatter {
	cfg := applyFormatterOpts(formatterConfig{timeLayout: defaultPlainTimeLayout}, opts)
	return &PlainFormatter{cfg: cfg}
}

func (f *PlainFormatter) Format(entry Entry) ([]byte, error) {
	var b strings.Builder

	// Header line: timestamp [LEVEL] message
	b.WriteString(entry.Time.Format(f.cfg.timeLayout))
	b.WriteString(" [")
	b.WriteString(fmt.Sprintf("%-5s", entry.Level.String()))
	b.WriteString("] ")
	b.WriteString(entry.Message)

	allKVs := mergeKVPairs(entry.Bound, entry.Args)
	hasError := entry.Error != nil

	if len(allKVs) == 0 && !hasError {
		b.WriteByte('\n')
		return []byte(b.String()), nil
	}

	kvCount := len(allKVs) / 2
	indexWidth := computeIndexWidth(kvCount, hasError)

	// 掃描最長 key 以對齊 `:` 欄位
	maxKeyLen := 0
	for i := 0; i < len(allKVs)-1; i += 2 {
		if k, ok := allKVs[i].(string); ok && len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}

	// 輸出 kv-pairs
	idx := 1
	for i := 0; i < len(allKVs)-1; i += 2 {
		key := fmt.Sprintf("%v", allKVs[i])
		val := fmt.Sprintf("%v", allKVs[i+1])
		b.WriteByte('\n')
		b.WriteString("  (")
		b.WriteString(padIndex(idx, indexWidth))
		b.WriteString(") ")
		b.WriteString(padRight(key, maxKeyLen))
		b.WriteString(" : ")
		b.WriteString(val)
		idx++
	}

	// Error block
	if hasError {
		info := ExtractError(entry.Error)
		b.WriteByte('\n')
		b.WriteString("  (error) ")
		if info.Code != "" {
			b.WriteByte('[')
			b.WriteString(info.Code)
			b.WriteString("] ")
		}
		b.WriteString(info.Message)

		if info.Stack != "" {
			for _, line := range strings.Split(info.Stack, "\n") {
				if line != "" {
					b.WriteString("\n    at ")
					b.WriteString(line)
				}
			}
		}
	}

	b.WriteByte('\n')
	return []byte(b.String()), nil
}

func mergeKVPairs(bound, args []any) []any {
	if len(bound) == 0 && len(args) == 0 {
		return nil
	}
	result := make([]any, 0, len(bound)+len(args))
	result = append(result, bound...)
	result = append(result, args...)
	return result
}

// computeIndexWidth: 有 error → 固定 5（對齊 "error"），無 error → kv 數量的位數。
func computeIndexWidth(kvCount int, hasError bool) int {
	if hasError {
		return 5
	}
	if kvCount == 0 {
		return 1
	}
	return len(strconv.Itoa(kvCount))
}

func padIndex(i, width int) string {
	s := strconv.Itoa(i)
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
