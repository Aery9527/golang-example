package logs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type spyHandler struct {
	called bool
	entry  Entry
}

func (s *spyHandler) Handle(entry Entry, next func(Entry)) {
	s.called = true
	s.entry = entry
	next(entry)
}

type blockHandler struct{}

func (blockHandler) Handle(entry Entry, next func(Entry)) {}

func TestChain_Execute_NoHandlers(t *testing.T) {
	var got Entry
	terminal := func(e Entry) { got = e }
	chain := NewChain(nil, terminal)

	chain.Execute(Entry{Message: "hello"})
	assert.Equal(t, "hello", got.Message)
}

func TestChain_Execute_SingleHandler(t *testing.T) {
	spy := &spyHandler{}
	var got Entry
	terminal := func(e Entry) { got = e }
	chain := NewChain([]Handler{spy}, terminal)

	chain.Execute(Entry{Message: "test"})
	assert.True(t, spy.called)
	assert.Equal(t, "test", got.Message)
}

func TestChain_Execute_HandlerModifiesEntry(t *testing.T) {
	modifier := HandlerFunc(func(entry Entry, next func(Entry)) {
		entry.Message = "modified"
		next(entry)
	})
	var got Entry
	chain := NewChain([]Handler{modifier}, func(e Entry) { got = e })

	chain.Execute(Entry{Message: "original"})
	assert.Equal(t, "modified", got.Message)
}

func TestChain_Execute_HandlerBlocks(t *testing.T) {
	reached := false
	chain := NewChain([]Handler{blockHandler{}}, func(e Entry) { reached = true })

	chain.Execute(Entry{})
	assert.False(t, reached)
}

func TestChain_Execute_MultipleHandlers_Order(t *testing.T) {
	var order []string
	h1 := HandlerFunc(func(e Entry, next func(Entry)) {
		order = append(order, "h1")
		next(e)
	})
	h2 := HandlerFunc(func(e Entry, next func(Entry)) {
		order = append(order, "h2")
		next(e)
	})
	chain := NewChain([]Handler{h1, h2}, func(e Entry) { order = append(order, "terminal") })

	chain.Execute(Entry{})
	assert.Equal(t, []string{"h1", "h2", "terminal"}, order)
}

func TestFanOut(t *testing.T) {
	var count int
	s1 := &mockSinkWriter{fn: func(Entry) { count++ }}
	s2 := &mockSinkWriter{fn: func(Entry) { count++ }}
	terminal := FanOut([]SinkWriter{s1, s2})

	terminal(Entry{})
	assert.Equal(t, 2, count)
}

type mockSinkWriter struct {
	fn func(Entry)
}

func (m *mockSinkWriter) Write(entry Entry) { m.fn(entry) }
