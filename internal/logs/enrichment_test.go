package logs

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCallerEnricher_AddsCallerToFront(t *testing.T) {
	e := &CallerEnricher{}
	entry := Entry{
		caller: "service.go:42",
		Bound:  []any{"existing", "val"},
	}
	var got Entry
	e.Handle(entry, func(e Entry) { got = e })

	assert.Equal(t, "caller", got.Bound[0])
	assert.Equal(t, "service.go:42", got.Bound[1])
	assert.Equal(t, "existing", got.Bound[2])
}

func TestCallerEnricher_EmptyCaller_NoOp(t *testing.T) {
	e := &CallerEnricher{}
	entry := Entry{Bound: []any{"k", "v"}}
	var got Entry
	e.Handle(entry, func(e Entry) { got = e })

	assert.Equal(t, []any{"k", "v"}, got.Bound)
}

func TestStaticEnricher_AddsKVToFront(t *testing.T) {
	e := NewStaticEnricher("service", "api")
	entry := Entry{Bound: []any{"k", "v"}}
	var got Entry
	e.Handle(entry, func(e Entry) { got = e })

	assert.Equal(t, "service", got.Bound[0])
	assert.Equal(t, "api", got.Bound[1])
	assert.Equal(t, "k", got.Bound[2])
}

func TestEnricher_PriorityLowest(t *testing.T) {
	enricher := NewStaticEnricher("env", "default")
	entry := Entry{
		Bound: []any{"env", "prod"},
		Args:  []any{"x", 1},
	}
	var got Entry
	enricher.Handle(entry, func(e Entry) { got = e })

	// enricher 的 "env" 在前，With 綁定的 "env" 在後
	assert.Equal(t, "env", got.Bound[0])
	assert.Equal(t, "default", got.Bound[1])
	assert.Equal(t, "env", got.Bound[2])
	assert.Equal(t, "prod", got.Bound[3])
}

func TestCallerEnricher_Format(t *testing.T) {
	entry := Entry{caller: "enrichment_test.go:99"}
	e := &CallerEnricher{}
	var got Entry
	e.Handle(entry, func(e Entry) { got = e })

	callerVal := got.Bound[1].(string)
	assert.True(t, strings.Contains(callerVal, ".go:"))
}
