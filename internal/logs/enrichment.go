package logs

// CallerEnricher 將 entry 預捕獲的 caller 資訊注入 Bound 前端。
type CallerEnricher struct{}

func (c *CallerEnricher) Handle(entry Entry, next func(Entry)) {
	if entry.caller != "" {
		entry.Bound = prepend(entry.Bound, "caller", entry.caller)
	}
	next(entry)
}

// StaticEnricher 每筆 log 固定附加一組 kv-pair。
type StaticEnricher struct {
	key string
	val any
}

func NewStaticEnricher(key string, val any) *StaticEnricher {
	return &StaticEnricher{key: key, val: val}
}

func (s *StaticEnricher) Handle(entry Entry, next func(Entry)) {
	entry.Bound = prepend(entry.Bound, s.key, s.val)
	next(entry)
}

// NoCaller 是一個標記型別，Configure 時用於移除預設 Caller enricher。
type NoCaller struct{}

func prepend(existing []any, key string, val any) []any {
	result := make([]any, 0, len(existing)+2)
	result = append(result, key, val)
	result = append(result, existing...)
	return result
}
