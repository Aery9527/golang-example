package logs

import ilogs "golan-example/internal/logs"

// Filter 是 Handler 的子集，負責決定 entry 是否放行。
type Filter = ilogs.Handler

func FilterByKey(key string, match func(string) bool) Filter {
	return ilogs.NewKeyFilter(key, match)
}

func FilterByMessage(match func(string) bool) Filter {
	return ilogs.NewMessageFilter(match)
}
