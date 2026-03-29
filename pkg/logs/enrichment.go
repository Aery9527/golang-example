package logs

import ilogs "golan-example/internal/logs"

// Enricher 是 Handler 的子集，負責注入額外 kv-pairs。
type Enricher = ilogs.Handler

func Caller() Enricher                    { return &ilogs.CallerEnricher{} }
func Static(key string, val any) Enricher { return ilogs.NewStaticEnricher(key, val) }

// noCaller 是標記型別，config.go 在組裝 chain 時辨識此型別。
type noCaller = ilogs.NoCaller
