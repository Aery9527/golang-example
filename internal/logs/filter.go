package logs

import "fmt"

type MessageFilter struct {
	match func(string) bool
}

func NewMessageFilter(match func(string) bool) *MessageFilter {
	return &MessageFilter{match: match}
}

func (f *MessageFilter) Handle(entry Entry, next func(Entry)) {
	if f.match(entry.Message) {
		next(entry)
	}
}

type KeyFilter struct {
	key   string
	match func(string) bool
}

func NewKeyFilter(key string, match func(string) bool) *KeyFilter {
	return &KeyFilter{key: key, match: match}
}

func (f *KeyFilter) Handle(entry Entry, next func(Entry)) {
	val, found := findKeyValue(f.key, entry.Bound, entry.Args)
	if !found || f.match(val) {
		next(entry)
	}
}

func findKeyValue(key string, bound, args []any) (string, bool) {
	for _, kvs := range [2][]any{bound, args} {
		for i := 0; i < len(kvs)-1; i += 2 {
			if fmt.Sprintf("%v", kvs[i]) == key {
				return fmt.Sprintf("%v", kvs[i+1]), true
			}
		}
	}
	return "", false
}
