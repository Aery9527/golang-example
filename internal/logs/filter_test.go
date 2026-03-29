package logs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageFilter_Pass(t *testing.T) {
	f := NewMessageFilter(func(m string) bool { return m != "heartbeat" })
	reached := false
	f.Handle(Entry{Message: "request"}, func(e Entry) { reached = true })
	assert.True(t, reached)
}

func TestMessageFilter_Block(t *testing.T) {
	f := NewMessageFilter(func(m string) bool { return m != "heartbeat" })
	reached := false
	f.Handle(Entry{Message: "heartbeat"}, func(e Entry) { reached = true })
	assert.False(t, reached)
}

func TestKeyFilter_Pass(t *testing.T) {
	f := NewKeyFilter("env", func(v string) bool { return v == "prod" })
	reached := false
	f.Handle(Entry{Args: []any{"env", "prod"}}, func(e Entry) { reached = true })
	assert.True(t, reached)
}

func TestKeyFilter_Block(t *testing.T) {
	f := NewKeyFilter("env", func(v string) bool { return v == "prod" })
	reached := false
	f.Handle(Entry{Args: []any{"env", "dev"}}, func(e Entry) { reached = true })
	assert.False(t, reached)
}

func TestKeyFilter_KeyNotFound_Pass(t *testing.T) {
	f := NewKeyFilter("env", func(v string) bool { return v == "prod" })
	reached := false
	f.Handle(Entry{Args: []any{"other", "val"}}, func(e Entry) { reached = true })
	assert.True(t, reached, "key not found should pass through")
}

func TestKeyFilter_SearchesBoundAndArgs(t *testing.T) {
	f := NewKeyFilter("rid", func(v string) bool { return v == "abc" })
	reached := false
	f.Handle(Entry{Bound: []any{"rid", "abc"}, Args: []any{"x", 1}}, func(e Entry) { reached = true })
	assert.True(t, reached)
}

func TestMultipleFilters_AND(t *testing.T) {
	f1 := NewMessageFilter(func(m string) bool { return m == "ok" })
	f2 := NewKeyFilter("env", func(v string) bool { return v == "prod" })

	reached := false
	chain := NewChain([]Handler{f1, f2}, func(e Entry) { reached = true })

	chain.Execute(Entry{Message: "ok", Args: []any{"env", "prod"}})
	assert.True(t, reached)

	reached = false
	chain.Execute(Entry{Message: "nope", Args: []any{"env", "prod"}})
	assert.False(t, reached)
}
